package auth

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"time"

	goldap "github.com/go-ldap/ldap/v3"
)

// LDAPAuthenticator authenticates users by performing an LDAP bind
// with their credentials against a Samba AD domain controller.
type LDAPAuthenticator struct {
	dcAddress string // host:port of primary DC
	dcHost    string // hostname for TLS ServerName
	baseDN    string
	useTLS    bool
}

// AuthResult contains the user info returned after successful authentication.
type AuthResult struct {
	Username string
	DN       string
	Groups   []string
}

// NewLDAPAuthenticator creates an authenticator targeting a specific DC.
func NewLDAPAuthenticator(dcAddress, dcHost, baseDN string, useTLS bool) *LDAPAuthenticator {
	return &LDAPAuthenticator{
		dcAddress: dcAddress,
		dcHost:    dcHost,
		baseDN:    baseDN,
		useTLS:    useTLS,
	}
}

// Authenticate verifies credentials by performing an LDAP bind as the user.
// Accepts username (sAMAccountName), UPN (user@domain), or full DN.
func (a *LDAPAuthenticator) Authenticate(ctx context.Context, username, password string) (*AuthResult, error) {
	if username == "" || password == "" {
		return nil, fmt.Errorf("username and password required")
	}

	// Build the bind DN from the username
	bindDN := a.resolveBindDN(username)

	// Connect to DC
	conn, err := a.connect()
	if err != nil {
		return nil, fmt.Errorf("connect to DC: %w", err)
	}
	defer conn.Close()

	// Bind as the user to verify credentials
	if err := conn.Bind(bindDN, password); err != nil {
		slog.Debug("auth: LDAP bind failed", "username", username, "error", err)
		return nil, fmt.Errorf("authentication failed")
	}

	// Bind succeeded — now search for the user's info
	result, err := a.searchUser(conn, username)
	if err != nil {
		slog.Warn("auth: user search failed after successful bind", "username", username, "error", err)
		// Auth succeeded even if search fails — return minimal info
		return &AuthResult{
			Username: username,
			DN:       bindDN,
		}, nil
	}

	slog.Info("auth: login successful", "username", result.Username, "dn", result.DN, "groups", len(result.Groups))
	return result, nil
}

// resolveBindDN converts a username to a bind DN.
// Supports: sAMAccountName, UPN (user@domain), or full DN.
func (a *LDAPAuthenticator) resolveBindDN(username string) string {
	// Already a DN
	if strings.Contains(strings.ToUpper(username), "CN=") || strings.Contains(strings.ToUpper(username), "DC=") {
		return username
	}
	// UPN format (user@domain) — Samba accepts this directly for bind
	if strings.Contains(username, "@") {
		return username
	}
	// Plain sAMAccountName — convert to UPN using the base DN
	domain := dnToDomain(a.baseDN)
	return username + "@" + domain
}

// dnToDomain converts "DC=dzsec,DC=net" to "dzsec.net"
func dnToDomain(baseDN string) string {
	parts := strings.Split(baseDN, ",")
	var domains []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(strings.ToUpper(part), "DC=") {
			domains = append(domains, part[3:])
		}
	}
	return strings.Join(domains, ".")
}

func (a *LDAPAuthenticator) connect() (*goldap.Conn, error) {
	dialer := &net.Dialer{Timeout: 5 * time.Second}

	if a.useTLS {
		tlsCfg := &tls.Config{
			ServerName:         a.dcHost,
			InsecureSkipVerify: false,
			MinVersion:         tls.VersionTLS12,
		}
		return goldap.DialURL(
			fmt.Sprintf("ldaps://%s", a.dcAddress),
			goldap.DialWithTLSConfig(tlsCfg),
			goldap.DialWithDialer(dialer),
		)
	}

	return goldap.DialURL(
		fmt.Sprintf("ldap://%s", a.dcAddress),
		goldap.DialWithDialer(dialer),
	)
}

func (a *LDAPAuthenticator) searchUser(conn *goldap.Conn, username string) (*AuthResult, error) {
	// Search by sAMAccountName or UPN
	var filter string
	if strings.Contains(username, "@") {
		filter = fmt.Sprintf("(userPrincipalName=%s)", goldap.EscapeFilter(username))
	} else if strings.Contains(strings.ToUpper(username), "CN=") {
		// DN — search by distinguishedName
		filter = fmt.Sprintf("(distinguishedName=%s)", goldap.EscapeFilter(username))
	} else {
		filter = fmt.Sprintf("(sAMAccountName=%s)", goldap.EscapeFilter(username))
	}

	sr := goldap.NewSearchRequest(
		a.baseDN,
		goldap.ScopeWholeSubtree,
		goldap.NeverDerefAliases,
		1, 5, false,
		filter,
		[]string{"distinguishedName", "sAMAccountName", "memberOf", "displayName"},
		nil,
	)

	result, err := conn.Search(sr)
	if err != nil {
		return nil, fmt.Errorf("user search: %w", err)
	}
	if len(result.Entries) == 0 {
		return nil, fmt.Errorf("user not found")
	}

	entry := result.Entries[0]
	return &AuthResult{
		Username: entry.GetAttributeValue("sAMAccountName"),
		DN:       entry.DN,
		Groups:   entry.GetAttributeValues("memberOf"),
	}, nil
}
