package directory

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"time"

	goldap "github.com/go-ldap/ldap/v3"

	"github.com/nickdawson/sambmin/internal/ldap"
	"github.com/nickdawson/sambmin/internal/models"
)

// Client provides typed directory queries backed by an LDAP connection pool.
type Client struct {
	pool   *ldap.Pool
	baseDN string
}

// NewClient creates a directory client.
func NewClient(pool *ldap.Pool, baseDN string) *Client {
	return &Client{pool: pool, baseDN: baseDN}
}

// Pool returns the underlying LDAP connection pool.
func (c *Client) Pool() *ldap.Pool {
	return c.pool
}

// Health tests LDAP connectivity by performing a base-level search.
func (c *Client) Health(ctx context.Context) error {
	conn, err := c.pool.Get(ctx)
	if err != nil {
		return fmt.Errorf("directory health: %w", err)
	}
	defer c.pool.Put(conn)

	sr := goldap.NewSearchRequest(
		c.baseDN,
		goldap.ScopeBaseObject,
		goldap.NeverDerefAliases,
		0, 5, false,
		"(objectClass=*)",
		[]string{"distinguishedName"},
		nil,
	)
	_, err = conn.Search(sr)
	if err != nil {
		return fmt.Errorf("directory health search: %w", err)
	}
	return nil
}

// GetSamAccountName looks up the sAMAccountName for a given DN.
func (c *Client) GetSamAccountName(ctx context.Context, dn string) (string, error) {
	conn, err := c.pool.Get(ctx)
	if err != nil {
		return "", err
	}
	defer c.pool.Put(conn)

	sr := goldap.NewSearchRequest(
		dn,
		goldap.ScopeBaseObject,
		goldap.NeverDerefAliases,
		0, 5, false,
		"(objectClass=*)",
		[]string{"sAMAccountName"},
		nil,
	)
	result, err := conn.Search(sr)
	if err != nil {
		return "", fmt.Errorf("lookup sAMAccountName for %s: %w", dn, err)
	}
	if len(result.Entries) == 0 {
		return "", fmt.Errorf("no object found at DN: %s", dn)
	}
	return result.Entries[0].GetAttributeValue("sAMAccountName"), nil
}

// Metrics returns aggregate counts for the dashboard.
func (c *Client) Metrics(ctx context.Context) (*models.DashboardMetrics, error) {
	conn, err := c.pool.Get(ctx)
	if err != nil {
		return nil, err
	}
	defer c.pool.Put(conn)

	m := &models.DashboardMetrics{}

	// Count users
	m.TotalUsers, _ = c.countObjects(conn, FilterUsers())
	// Count computers
	m.TotalComputers, _ = c.countObjects(conn, FilterComputers())
	// Count groups
	m.TotalGroups, _ = c.countObjects(conn, FilterGroups())
	// Count disabled users
	m.DisabledUsers, _ = c.countObjects(conn, FilterDisabledUsers())
	// Count locked accounts
	m.LockedAccounts, _ = c.countObjects(conn, FilterLockedUsers())

	slog.Debug("directory: metrics gathered",
		"users", m.TotalUsers, "computers", m.TotalComputers,
		"groups", m.TotalGroups, "dc", conn.DC())

	return m, nil
}

func (c *Client) countObjects(conn *ldap.Conn, filter string) (int, error) {
	sr := goldap.NewSearchRequest(
		c.baseDN,
		goldap.ScopeWholeSubtree,
		goldap.NeverDerefAliases,
		0, 0, false,
		filter,
		[]string{"dn"},
		nil,
	)
	result, err := conn.SearchWithPaging(sr, 500)
	if err != nil {
		return 0, err
	}
	return len(result.Entries), nil
}

// ModifyAttributes updates LDAP attributes on an object, authenticated as the
// specified user (not the service account). Creates a temporary connection
// bound as the user for the modification.
func (c *Client) ModifyAttributes(ctx context.Context, objectDN string, attrs map[string]string, userDN, userPassword string) error {
	// Get a temporary connection to the primary DC, bound as the acting user
	dcs := c.pool.DCs()
	if len(dcs) == 0 {
		return fmt.Errorf("no domain controllers available")
	}

	// Use the first DC (primary)
	dc := dcs[0]
	addr := fmt.Sprintf("%s:%d", dc.Address, dc.Port)

	dialer := &net.Dialer{Timeout: 10 * time.Second}
	tlsCfg := &tls.Config{
		ServerName:         dc.Hostname,
		InsecureSkipVerify: false,
		MinVersion:         tls.VersionTLS12,
	}

	conn, err := goldap.DialURL(
		fmt.Sprintf("ldaps://%s", addr),
		goldap.DialWithTLSConfig(tlsCfg),
		goldap.DialWithDialer(dialer),
	)
	if err != nil {
		return fmt.Errorf("connect for modify: %w", err)
	}
	defer conn.Close()

	// Bind as the acting user
	if err := conn.Bind(userDN, userPassword); err != nil {
		return fmt.Errorf("bind as user: %w", err)
	}

	// Build modify request
	modReq := goldap.NewModifyRequest(objectDN, nil)
	for attr, val := range attrs {
		modReq.Replace(attr, []string{val})
	}

	if err := conn.Modify(modReq); err != nil {
		return fmt.Errorf("modify %s: %w", objectDN, err)
	}

	slog.Info("directory: attributes modified", "dn", objectDN, "attrs", len(attrs))
	return nil
}

// DeleteObject deletes an LDAP object, authenticated as the specified user.
func (c *Client) DeleteObject(ctx context.Context, objectDN, userDN, userPassword string) error {
	dcs := c.pool.DCs()
	if len(dcs) == 0 {
		return fmt.Errorf("no domain controllers available")
	}

	dc := dcs[0]
	addr := fmt.Sprintf("%s:%d", dc.Address, dc.Port)

	dialer := &net.Dialer{Timeout: 10 * time.Second}
	tlsCfg := &tls.Config{
		ServerName:         dc.Hostname,
		InsecureSkipVerify: false,
		MinVersion:         tls.VersionTLS12,
	}

	conn, err := goldap.DialURL(
		fmt.Sprintf("ldaps://%s", addr),
		goldap.DialWithTLSConfig(tlsCfg),
		goldap.DialWithDialer(dialer),
	)
	if err != nil {
		return fmt.Errorf("connect for delete: %w", err)
	}
	defer conn.Close()

	if err := conn.Bind(userDN, userPassword); err != nil {
		return fmt.Errorf("bind as user: %w", err)
	}

	delReq := goldap.NewDelRequest(objectDN, nil)
	if err := conn.Del(delReq); err != nil {
		return fmt.Errorf("delete %s: %w", objectDN, err)
	}

	slog.Info("directory: object deleted", "dn", objectDN)
	return nil
}

// --- Attribute helpers ---

func getAttr(entry *goldap.Entry, name string) string {
	return entry.GetAttributeValue(name)
}

func getAttrs(entry *goldap.Entry, name string) []string {
	return entry.GetAttributeValues(name)
}

func parseADTimestamp(s string) time.Time {
	if s == "" || s == "0" {
		return time.Time{}
	}
	// AD generalized time: 20240115090000.0Z
	for _, layout := range []string{
		"20060102150405.0Z",
		"20060102150405Z",
		"2006-01-02T15:04:05Z",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	// Windows FILETIME (100ns intervals since 1601-01-01)
	if n, err := strconv.ParseInt(s, 10, 64); err == nil && n > 0 {
		// Convert Windows FILETIME to Unix timestamp
		const ticksPerSecond = 10_000_000
		const epochDiff = 11644473600 // seconds between 1601 and 1970
		unixSec := (n / ticksPerSecond) - epochDiff
		if unixSec > 0 {
			return time.Unix(unixSec, 0)
		}
	}
	return time.Time{}
}

func parseUAC(entry *goldap.Entry) (enabled, lockedOut, passwordExpired bool) {
	uacStr := getAttr(entry, ldap.AttrUAC)
	uac, _ := strconv.ParseInt(uacStr, 10, 32)

	enabled = (uac & int64(ldap.UACAccountDisable)) == 0

	// Prefer computed lockout attribute (accurate, not cached)
	computedStr := getAttr(entry, ldap.AttrUACComputed)
	if computedStr != "" {
		computed, _ := strconv.ParseInt(computedStr, 10, 32)
		lockedOut = (computed & int64(ldap.UACLockout)) != 0
	} else {
		// Fallback: check lockoutTime attribute
		lockTime := getAttr(entry, ldap.AttrLockoutTime)
		if lockTime != "" && lockTime != "0" {
			lockedOut = true
		}
	}

	// Password expired: pwdLastSet == 0 means must change at next logon
	pwdLastSet := getAttr(entry, ldap.AttrPwdLastSet)
	if pwdLastSet == "0" {
		passwordExpired = true
	}

	return
}

func parseGroupType(gtStr string) (scope, kind string) {
	gt, _ := strconv.ParseInt(gtStr, 10, 32)
	gt32 := int32(gt)

	if gt32 < 0 { // sign bit set = security group
		kind = "security"
	} else {
		kind = "distribution"
	}

	// Mask off the security bit to get scope
	scopeBits := gt32 & 0x0000000F
	switch {
	case scopeBits&int32(ldap.GroupTypeUniversal) != 0:
		scope = "universal"
	case scopeBits&int32(ldap.GroupTypeDomainLocal) != 0:
		scope = "domainLocal"
	default:
		scope = "global"
	}

	return
}

// cnFromDN extracts the CN value from a distinguished name.
func cnFromDN(dn string) string {
	parts := strings.SplitN(dn, ",", 2)
	if len(parts) == 0 {
		return dn
	}
	cn := parts[0]
	if strings.HasPrefix(strings.ToUpper(cn), "CN=") {
		return cn[3:]
	}
	return cn
}

func userFromEntry(entry *goldap.Entry) models.User {
	enabled, lockedOut, pwdExpired := parseUAC(entry)
	return models.User{
		DN:              entry.DN,
		SamAccountName:  getAttr(entry, ldap.AttrSAM),
		DisplayName:     getAttr(entry, ldap.AttrDisplayName),
		GivenName:       getAttr(entry, ldap.AttrGivenName),
		Surname:         getAttr(entry, ldap.AttrSurname),
		Email:           getAttr(entry, ldap.AttrEmail),
		UPN:             getAttr(entry, ldap.AttrUPN),
		Department:      getAttr(entry, ldap.AttrDepartment),
		Title:           getAttr(entry, ldap.AttrTitle),
		Manager:         getAttr(entry, ldap.AttrManager),
		Enabled:         enabled,
		LockedOut:       lockedOut,
		PasswordExpired: pwdExpired,
		LastLogon:       parseADTimestamp(getAttr(entry, ldap.AttrLastLogon)),
		WhenCreated:     parseADTimestamp(getAttr(entry, ldap.AttrWhenCreated)),
		WhenChanged:     parseADTimestamp(getAttr(entry, ldap.AttrWhenChanged)),
		MemberOf:        getAttrs(entry, ldap.AttrMemberOf),
	}
}

func groupFromEntry(entry *goldap.Entry) models.Group {
	scope, kind := parseGroupType(getAttr(entry, ldap.AttrGroupType))
	return models.Group{
		DN:             entry.DN,
		Name:           getAttr(entry, ldap.AttrCN),
		SamAccountName: getAttr(entry, ldap.AttrSAM),
		Description:    getAttr(entry, ldap.AttrDescription),
		GroupType:      kind,
		GroupScope:     scope,
		Members:        getAttrs(entry, ldap.AttrMember),
		MemberOf:       getAttrs(entry, ldap.AttrMemberOf),
	}
}

func computerFromEntry(entry *goldap.Entry) models.Computer {
	uacStr := getAttr(entry, ldap.AttrUAC)
	uac, _ := strconv.ParseInt(uacStr, 10, 32)
	enabled := (uac & int64(ldap.UACAccountDisable)) == 0

	return models.Computer{
		DN:             entry.DN,
		Name:           getAttr(entry, ldap.AttrCN),
		SamAccountName: getAttr(entry, ldap.AttrSAM),
		DNSHostName:    getAttr(entry, ldap.AttrDNSHostName),
		OS:             getAttr(entry, ldap.AttrOS),
		OSVersion:      getAttr(entry, ldap.AttrOSVersion),
		Enabled:        enabled,
		LastLogon:      parseADTimestamp(getAttr(entry, ldap.AttrLastLogon)),
		WhenCreated:    parseADTimestamp(getAttr(entry, ldap.AttrWhenCreated)),
	}
}

func ouFromEntry(entry *goldap.Entry) models.OU {
	return models.OU{
		DN:          entry.DN,
		Name:        getAttr(entry, ldap.AttrOU),
		Description: getAttr(entry, ldap.AttrDescription),
	}
}
