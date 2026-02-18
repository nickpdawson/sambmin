package directory

import (
	"context"
	"fmt"

	goldap "github.com/go-ldap/ldap/v3"

	"github.com/nickdawson/sambmin/internal/ldap"
	"github.com/nickdawson/sambmin/internal/models"
)

// ListUsersOptions configures user listing behavior.
type ListUsersOptions struct {
	Filter string // "all", "enabled", "disabled", "locked"
	Search string // text search across name/email/sam
	Limit  int
	Offset int
}

// ListUsers returns users from the directory, with optional filtering.
func (c *Client) ListUsers(ctx context.Context, opts ListUsersOptions) ([]models.User, int, error) {
	conn, err := c.pool.Get(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	defer c.pool.Put(conn)

	// Build filter
	filter := FilterUsers()
	switch opts.Filter {
	case "enabled":
		filter = FilterEnabledUsers()
	case "disabled":
		filter = FilterDisabledUsers()
	case "locked":
		filter = FilterLockedUsers()
	}

	// Add text search if provided
	if opts.Search != "" {
		filter = FilterTextSearch(opts.Search, filter)
	}

	sr := goldap.NewSearchRequest(
		c.baseDN,
		goldap.ScopeWholeSubtree,
		goldap.NeverDerefAliases,
		0, 0, false,
		filter,
		ldap.UserAttrs,
		nil,
	)

	result, err := conn.SearchWithPaging(sr, 500)
	if err != nil {
		return nil, 0, fmt.Errorf("search users: %w", err)
	}

	total := len(result.Entries)

	// Apply offset/limit for client-side pagination
	entries := result.Entries
	if opts.Offset > 0 && opts.Offset < len(entries) {
		entries = entries[opts.Offset:]
	}
	if opts.Limit > 0 && opts.Limit < len(entries) {
		entries = entries[:opts.Limit]
	}

	users := make([]models.User, 0, len(entries))
	for _, entry := range entries {
		users = append(users, userFromEntry(entry))
	}

	return users, total, nil
}

// GetUser returns a single user by DN.
func (c *Client) GetUser(ctx context.Context, dn string) (*models.User, error) {
	conn, err := c.pool.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	defer c.pool.Put(conn)

	sr := goldap.NewSearchRequest(
		dn,
		goldap.ScopeBaseObject,
		goldap.NeverDerefAliases,
		0, 1, false,
		"(objectClass=user)",
		ldap.UserAttrs,
		nil,
	)

	result, err := conn.Search(sr)
	if err != nil {
		return nil, fmt.Errorf("search user %s: %w", dn, err)
	}

	if len(result.Entries) == 0 {
		return nil, fmt.Errorf("user not found: %s", dn)
	}

	u := userFromEntry(result.Entries[0])
	return &u, nil
}

// GetUserBySAM looks up a user by sAMAccountName.
func (c *Client) GetUserBySAM(ctx context.Context, sam string) (*models.User, error) {
	conn, err := c.pool.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("get user by sam: %w", err)
	}
	defer c.pool.Put(conn)

	filter := fmt.Sprintf("(&%s%s)", FilterUsers(), FilterBySAM(sam))

	sr := goldap.NewSearchRequest(
		c.baseDN,
		goldap.ScopeWholeSubtree,
		goldap.NeverDerefAliases,
		0, 1, false,
		filter,
		ldap.UserAttrs,
		nil,
	)

	result, err := conn.Search(sr)
	if err != nil {
		return nil, fmt.Errorf("search user by sam %s: %w", sam, err)
	}

	if len(result.Entries) == 0 {
		return nil, fmt.Errorf("user not found: %s", sam)
	}

	u := userFromEntry(result.Entries[0])
	return &u, nil
}
