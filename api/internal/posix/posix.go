// Package posix manages auto-assignment of RFC2307 POSIX attributes
// (uidNumber, gidNumber, unixHomeDirectory, loginShell) on new users and
// groups created through Sambmin.
//
// Why this exists: winbind on a member host configured with
// `idmap config <domain> : backend = ad / schema_mode = RFC2307` only maps
// AD users and groups that already have uidNumber / gidNumber populated on
// the AD object. Anything without those attributes lands in winbind's
// negative cache and disappears from getpwnam/getgrnam — which blocks UI
// dropdowns on tools like TrueNAS that enumerate via NSS. To avoid that
// failure mode, when the domain is already using RFC2307 we mirror what
// LDAP Account Manager does: pick the highest uidNumber/gidNumber in use
// today and assign next+1, floored at the configured minimum.
package posix

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	goldap "github.com/go-ldap/ldap/v3"

	"github.com/nickdawson/sambmin/internal/config"
	"github.com/nickdawson/sambmin/internal/directory"
)

// Allocator allocates POSIX IDs and sets POSIX attributes on new objects.
type Allocator struct {
	dir *directory.Client
	cfg config.RFC2307Config

	mu             sync.Mutex
	enabledKnown   bool
	enabledValue   bool
	enabledCheckAt time.Time
}

// New constructs an Allocator backed by the given directory client and
// settings.
func New(dir *directory.Client, cfg config.RFC2307Config) *Allocator {
	return &Allocator{dir: dir, cfg: cfg}
}

// detectTTL is how long an IsEnabled result is trusted before re-probing.
const detectTTL = 5 * time.Minute

// IsEnabled reports whether the domain appears to use RFC2307 already. It
// is true if any user object has uidNumber set. Result is cached for
// detectTTL since admins rarely toggle this mid-day, and the probe is
// otherwise repeated on every create.
func (a *Allocator) IsEnabled(ctx context.Context) (bool, error) {
	a.mu.Lock()
	if a.enabledKnown && time.Since(a.enabledCheckAt) < detectTTL {
		v := a.enabledValue
		a.mu.Unlock()
		return v, nil
	}
	a.mu.Unlock()

	conn, err := a.dir.GetConn(ctx)
	if err != nil {
		return false, fmt.Errorf("posix detect: %w", err)
	}
	defer a.dir.PutConn(conn)

	sr := goldap.NewSearchRequest(
		a.dir.BaseDN(),
		goldap.ScopeWholeSubtree,
		goldap.NeverDerefAliases,
		1, 5, false, // SizeLimit=1 — one hit is enough to confirm
		"(&(objectClass=user)(uidNumber=*))",
		[]string{"uidNumber"},
		nil,
	)
	result, err := conn.Search(sr)
	// A size-limit-exceeded error still means "we found at least one" — treat
	// as enabled. Other errors propagate.
	enabled := false
	if err != nil {
		if lerr, ok := err.(*goldap.Error); ok && lerr.ResultCode == goldap.LDAPResultSizeLimitExceeded {
			enabled = true
		} else {
			return false, fmt.Errorf("posix detect search: %w", err)
		}
	} else {
		enabled = len(result.Entries) > 0
	}

	a.mu.Lock()
	a.enabledKnown = true
	a.enabledValue = enabled
	a.enabledCheckAt = time.Now()
	a.mu.Unlock()
	return enabled, nil
}

// AllocateUID returns the next free uidNumber: max(existing) + 1, floored
// at cfg.MinUID.
func (a *Allocator) AllocateUID(ctx context.Context) (int, error) {
	return a.allocateNext(ctx, "(&(objectClass=user)(uidNumber=*))", "uidNumber", a.cfg.MinUID)
}

// AllocateGID returns the next free gidNumber: max(existing) + 1, floored
// at cfg.MinGID.
func (a *Allocator) AllocateGID(ctx context.Context) (int, error) {
	return a.allocateNext(ctx, "(&(objectClass=group)(gidNumber=*))", "gidNumber", a.cfg.MinGID)
}

func (a *Allocator) allocateNext(ctx context.Context, filter, attr string, floor int) (int, error) {
	conn, err := a.dir.GetConn(ctx)
	if err != nil {
		return 0, fmt.Errorf("posix allocate %s: %w", attr, err)
	}
	defer a.dir.PutConn(conn)

	sr := goldap.NewSearchRequest(
		a.dir.BaseDN(),
		goldap.ScopeWholeSubtree,
		goldap.NeverDerefAliases,
		0, 30, false,
		filter,
		[]string{attr},
		nil,
	)
	// Page through results so a large domain doesn't trip the server-side
	// size limit and silently truncate our max calculation.
	result, err := conn.SearchWithPaging(sr, 500)
	if err != nil {
		return 0, fmt.Errorf("posix allocate %s search: %w", attr, err)
	}

	max := floor - 1
	for _, entry := range result.Entries {
		raw := entry.GetAttributeValue(attr)
		if raw == "" {
			continue
		}
		n, err := strconv.Atoi(raw)
		if err != nil {
			continue
		}
		if n > max {
			max = n
		}
	}
	return max + 1, nil
}

// ResolvePrimaryGroupGID returns the gidNumber of the given group DN. If
// the group has no gidNumber yet, one is allocated and written using the
// acting user's credentials, then returned. The two-trip nature is
// intentional: a brand-new domain that hasn't enabled RFC2307 yet will
// have IsEnabled() == false and never reach this code; once a single user
// gets a uidNumber, every subsequent user-create will hit this path, and
// Domain Users (or whatever the primary group is) gets its gidNumber on
// the first run.
func (a *Allocator) ResolvePrimaryGroupGID(ctx context.Context, groupDN, actorDN, actorPW string) (int, error) {
	conn, err := a.dir.GetConn(ctx)
	if err != nil {
		return 0, fmt.Errorf("posix resolve primary gid: %w", err)
	}

	sr := goldap.NewSearchRequest(
		groupDN,
		goldap.ScopeBaseObject,
		goldap.NeverDerefAliases,
		0, 5, false,
		"(objectClass=*)",
		[]string{"gidNumber"},
		nil,
	)
	result, err := conn.Search(sr)
	a.dir.PutConn(conn)
	if err != nil {
		return 0, fmt.Errorf("posix lookup primary group %s: %w", groupDN, err)
	}
	if len(result.Entries) == 0 {
		return 0, fmt.Errorf("posix: primary group not found at %s", groupDN)
	}

	if raw := result.Entries[0].GetAttributeValue("gidNumber"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			return 0, fmt.Errorf("posix: primary group %s has invalid gidNumber %q: %w", groupDN, raw, err)
		}
		return n, nil
	}

	gid, err := a.AllocateGID(ctx)
	if err != nil {
		return 0, err
	}
	if err := a.dir.ModifyAttributes(ctx, groupDN, map[string]string{"gidNumber": strconv.Itoa(gid)}, actorDN, actorPW); err != nil {
		return 0, fmt.Errorf("posix: backfill gidNumber on primary group %s: %w", groupDN, err)
	}
	return gid, nil
}

// UserAttrs builds the attribute map to apply to a freshly-created user.
// samAccountName is interpolated into the configured home template.
func (a *Allocator) UserAttrs(uid, gid int, samAccountName string) map[string]string {
	return map[string]string{
		"uidNumber":         strconv.Itoa(uid),
		"gidNumber":         strconv.Itoa(gid),
		"unixHomeDirectory": fmt.Sprintf(a.cfg.HomeTemplate, samAccountName),
		"loginShell":        a.cfg.DefaultShell,
	}
}

// GroupAttrs builds the attribute map to apply to a freshly-created group.
func (a *Allocator) GroupAttrs(gid int) map[string]string {
	return map[string]string{
		"gidNumber": strconv.Itoa(gid),
	}
}
