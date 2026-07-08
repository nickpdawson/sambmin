package directory

import (
	"context"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"

	goldap "github.com/go-ldap/ldap/v3"
)

// sidToString decodes a binary NT security identifier into its canonical
// string form (e.g. "S-1-5-21-...-1103"). Returns "" for malformed input.
//
// Layout: [revision:1][subAuthorityCount:1][identifierAuthority:6 big-endian]
// followed by subAuthorityCount little-endian uint32 sub-authorities.
func sidToString(b []byte) string {
	if len(b) < 8 {
		return ""
	}
	revision := b[0]
	subCount := int(b[1])
	if len(b) < 8+subCount*4 {
		return ""
	}

	var authority uint64
	for i := 2; i < 8; i++ {
		authority = authority<<8 | uint64(b[i])
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "S-%d-%d", revision, authority)
	for i := 0; i < subCount; i++ {
		sub := binary.LittleEndian.Uint32(b[8+i*4:])
		fmt.Fprintf(&sb, "-%d", sub)
	}
	return sb.String()
}

// ResolveSID returns the string objectSid of the object at dn.
func (c *Client) ResolveSID(ctx context.Context, dn string) (string, error) {
	conn, err := c.pool.Get(ctx)
	if err != nil {
		return "", err
	}
	defer c.pool.Put(conn)

	sr := goldap.NewSearchRequest(
		dn, goldap.ScopeBaseObject, goldap.NeverDerefAliases, 0, 5, false,
		"(objectClass=*)", []string{"objectSid"}, nil,
	)
	res, err := conn.Search(sr)
	if err != nil {
		return "", fmt.Errorf("lookup objectSid for %s: %w", dn, err)
	}
	if len(res.Entries) == 0 {
		return "", fmt.Errorf("no object found at DN: %s", dn)
	}
	raw := res.Entries[0].GetRawAttributeValue("objectSid")
	sid := sidToString(raw)
	if sid == "" {
		return "", fmt.Errorf("object %s has no usable objectSid", dn)
	}
	return sid, nil
}

// Trustee is a security principal that can appear in an ACE.
type Trustee struct {
	SID            string `json:"sid"`
	SamAccountName string `json:"samAccountName"`
	DN             string `json:"dn"`
	Class          string `json:"class"` // "user", "group", "computer"
}

// SIDNameMap returns a map of string SID -> Trustee for every user, group, and
// computer in the directory. Used to render the trustees of existing ACEs
// (which store only SIDs) as human-readable names. The directory is small, so
// a single subtree search is cheaper than per-SID lookups.
func (c *Client) SIDNameMap(ctx context.Context) (map[string]Trustee, error) {
	conn, err := c.pool.Get(ctx)
	if err != nil {
		return nil, err
	}
	defer c.pool.Put(conn)

	sr := goldap.NewSearchRequest(
		c.baseDN, goldap.ScopeWholeSubtree, goldap.NeverDerefAliases, 0, 0, false,
		"(|(objectClass=user)(objectClass=group))",
		[]string{"objectSid", "sAMAccountName", "objectClass"}, nil,
	)
	res, err := conn.SearchWithPaging(sr, 500)
	if err != nil {
		return nil, fmt.Errorf("enumerate principals: %w", err)
	}

	out := make(map[string]Trustee, len(res.Entries))
	for _, e := range res.Entries {
		sid := sidToString(e.GetRawAttributeValue("objectSid"))
		if sid == "" {
			continue
		}
		out[sid] = Trustee{
			SID:            sid,
			SamAccountName: e.GetAttributeValue("sAMAccountName"),
			DN:             e.DN,
			Class:          classFromObjectClasses(e.GetAttributeValues("objectClass")),
		}
	}
	return out, nil
}

func classFromObjectClasses(classes []string) string {
	// Most specific wins: computer is also a user; group is distinct.
	has := func(name string) bool {
		for _, c := range classes {
			if strings.EqualFold(c, name) {
				return true
			}
		}
		return false
	}
	switch {
	case has("computer"):
		return "computer"
	case has("group"):
		return "group"
	case has("user"):
		return "user"
	default:
		return "object"
	}
}

// wellKnownSDDLSIDs maps the two-letter SDDL aliases that samba emits for
// well-known principals to human-readable names, so inherited/default ACEs
// referencing them can be labelled without an LDAP lookup.
var wellKnownSDDLSIDs = map[string]string{
	"AO": "Account Operators",
	"AU": "Authenticated Users",
	"BA": "Administrators",
	"BO": "Backup Operators",
	"CO": "Creator Owner",
	"DA": "Domain Admins",
	"DU": "Domain Users",
	"EA": "Enterprise Admins",
	"ED": "Enterprise Domain Controllers",
	"PA": "Group Policy Creator Owners",
	"PO": "Printer Operators",
	"PS": "Principal Self",
	"RO": "Enterprise Read-Only Domain Controllers",
	"RU": "Pre-Windows 2000 Compatible Access",
	"SO": "Server Operators",
	"SY": "Local System",
	"WD": "Everyone",
}

// TrusteeName resolves an SDDL trustee token (either a full "S-1-5-..." SID or
// a two-letter well-known alias) to a display name using the supplied SID map.
func TrusteeName(token string, sidMap map[string]Trustee) string {
	if t, ok := sidMap[token]; ok && t.SamAccountName != "" {
		return t.SamAccountName
	}
	if name, ok := wellKnownSDDLSIDs[token]; ok {
		return name
	}
	if strings.HasPrefix(token, "S-1-") {
		// Well-known SIDs not aliased above (e.g. S-1-1-0 = Everyone).
		if name := wellKnownFullSID(token); name != "" {
			return name
		}
	}
	return token
}

func wellKnownFullSID(sid string) string {
	switch sid {
	case "S-1-1-0":
		return "Everyone"
	case "S-1-5-11":
		return "Authenticated Users"
	case "S-1-5-18":
		return "Local System"
	case "S-1-5-9":
		return "Enterprise Domain Controllers"
	}
	// Domain RID-suffixed well-knowns (…-512 Domain Admins, etc.)
	if i := strings.LastIndex(sid, "-"); i > 0 {
		if rid, err := strconv.Atoi(sid[i+1:]); err == nil {
			switch rid {
			case 512:
				return "Domain Admins"
			case 513:
				return "Domain Users"
			case 519:
				return "Enterprise Admins"
			case 518:
				return "Schema Admins"
			}
		}
	}
	return ""
}
