package validate

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// SAMAccountName: alphanumeric start, then alphanumeric/._- /space, optional trailing $ (computer accounts), max 64 chars.
	reSAMAccountName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._\- ]{0,62}\$?$`)

	// GroupName / OUName: same pattern as SAMAccountName.
	reGroupName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9 ._-]{0,63}$`)

	// DNSName: labels, dots, wildcards, @.
	reDNSName = regexp.MustCompile(`^[a-zA-Z0-9._@*-]{1,253}$`)

	// SPN: service/host format.
	reSPN = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*/[a-zA-Z0-9._-]+$`)

	// dnsTypes is the set of valid DNS record types.
	dnsTypes = map[string]bool{
		"A": true, "AAAA": true, "CNAME": true, "MX": true,
		"NS": true, "PTR": true, "SRV": true, "TXT": true, "SOA": true,
	}

	// sensitiveAttributes that must never be returned from LDAP search.
	sensitiveAttributes = map[string]bool{
		"unicodepwd":              true,
		"supplementalcredentials": true,
		"dbcspwd":                 true,
		"lmpwdhistory":            true,
		"ntpwdhistory":            true,
	}
)

// SAMAccountName validates a sAMAccountName.
func SAMAccountName(name string) error {
	if name == "" {
		return fmt.Errorf("account name is required")
	}
	if !reSAMAccountName.MatchString(name) {
		return fmt.Errorf("invalid account name: must be 1-64 alphanumeric characters, dots, underscores, hyphens, or spaces")
	}
	return nil
}

// GroupName validates a group or OU name.
func GroupName(name string) error {
	if name == "" {
		return fmt.Errorf("name is required")
	}
	if !reGroupName.MatchString(name) {
		return fmt.Errorf("invalid name: must be 1-64 alphanumeric characters, dots, underscores, hyphens, or spaces")
	}
	return nil
}

// OUName is an alias for GroupName validation.
func OUName(name string) error {
	return GroupName(name)
}

// DNSType validates a DNS record type against the whitelist.
func DNSType(t string) error {
	if !dnsTypes[strings.ToUpper(t)] {
		return fmt.Errorf("invalid DNS record type %q: must be one of A, AAAA, CNAME, MX, NS, PTR, SRV, TXT, SOA", t)
	}
	return nil
}

// DNSName validates a DNS name (hostname, zone name, record name).
func DNSName(name string) error {
	if name == "" {
		return fmt.Errorf("DNS name is required")
	}
	if !reDNSName.MatchString(name) {
		return fmt.Errorf("invalid DNS name %q: contains invalid characters", name)
	}
	return nil
}

// SPN validates a Service Principal Name.
func SPN(spn string) error {
	if spn == "" {
		return fmt.Errorf("SPN is required")
	}
	if !reSPN.MatchString(spn) {
		return fmt.Errorf("invalid SPN format: expected service/host (e.g., HTTP/web.example.com)")
	}
	return nil
}

// Password validates password constraints (no null bytes, max length).
func Password(pw string) error {
	if pw == "" {
		return fmt.Errorf("password is required")
	}
	if len(pw) > 128 {
		return fmt.Errorf("password exceeds maximum length of 128 characters")
	}
	if strings.ContainsRune(pw, 0) {
		return fmt.Errorf("password contains invalid characters")
	}
	return nil
}

// NoFlagInjection rejects strings starting with a hyphen to prevent
// accidental flag injection in CLI arguments.
func NoFlagInjection(value, fieldName string) error {
	if strings.HasPrefix(value, "-") {
		return fmt.Errorf("%s must not start with a hyphen", fieldName)
	}
	return nil
}

// BaseDN validates that a base DN ends with the expected domain suffix.
func BaseDN(baseDN, domainSuffix string) error {
	if baseDN == "" {
		return fmt.Errorf("baseDN is required")
	}
	if domainSuffix != "" && !strings.HasSuffix(strings.ToLower(baseDN), strings.ToLower(domainSuffix)) {
		return fmt.Errorf("baseDN must end with %s", domainSuffix)
	}
	return nil
}

// RawFilter validates an LDAP raw filter string.
func RawFilter(filter string) error {
	if filter == "" {
		return nil // not using raw filter
	}
	if len(filter) > 2048 {
		return fmt.Errorf("filter exceeds maximum length of 2048 characters")
	}
	if strings.ContainsRune(filter, 0) {
		return fmt.Errorf("filter contains null bytes")
	}
	// Check balanced parentheses
	depth := 0
	for _, c := range filter {
		switch c {
		case '(':
			depth++
		case ')':
			depth--
		}
		if depth < 0 {
			return fmt.Errorf("filter has unbalanced parentheses")
		}
	}
	if depth != 0 {
		return fmt.Errorf("filter has unbalanced parentheses")
	}
	return nil
}

// Attributes checks that no sensitive attributes are being requested.
func Attributes(attrs []string) error {
	for _, a := range attrs {
		if sensitiveAttributes[strings.ToLower(a)] {
			return fmt.Errorf("attribute %q is not allowed", a)
		}
	}
	return nil
}

// IsSensitiveAttribute checks a single attribute name.
func IsSensitiveAttribute(attr string) bool {
	return sensitiveAttributes[strings.ToLower(attr)]
}
