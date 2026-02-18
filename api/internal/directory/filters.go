package directory

import (
	"fmt"
	"strings"

	goldap "github.com/go-ldap/ldap/v3"
)

// LDAP matching rule OIDs for bitwise operations on AD attributes.
const (
	// Bitwise AND — tests if specific bits are set
	matchRuleBitAnd = "1.2.840.113556.1.4.803"
	// Bitwise OR — tests if any of the specified bits are set
	matchRuleBitOr = "1.2.840.113556.1.4.804"
)

// FilterUsers matches user accounts (excluding computer accounts).
func FilterUsers() string {
	return "(&(objectClass=user)(!(objectClass=computer)))"
}

// FilterEnabledUsers matches enabled user accounts.
func FilterEnabledUsers() string {
	return fmt.Sprintf(
		"(&(objectClass=user)(!(objectClass=computer))(!(userAccountControl:%s:=2)))",
		matchRuleBitAnd,
	)
}

// FilterDisabledUsers matches disabled user accounts.
func FilterDisabledUsers() string {
	return fmt.Sprintf(
		"(&(objectClass=user)(!(objectClass=computer))(userAccountControl:%s:=2))",
		matchRuleBitAnd,
	)
}

// FilterLockedUsers matches locked-out user accounts.
// Uses lockoutTime > 0 which is more reliable than UAC bit.
func FilterLockedUsers() string {
	return "(&(objectClass=user)(!(objectClass=computer))(lockoutTime>=1))"
}

// FilterGroups matches all groups.
func FilterGroups() string {
	return "(objectClass=group)"
}

// FilterSecurityGroups matches security groups (sign bit set in groupType).
func FilterSecurityGroups() string {
	return fmt.Sprintf(
		"(&(objectClass=group)(groupType:%s:=2147483648))",
		matchRuleBitAnd,
	)
}

// FilterComputers matches computer accounts.
func FilterComputers() string {
	return "(objectClass=computer)"
}

// FilterOUs matches organizational units.
func FilterOUs() string {
	return "(objectClass=organizationalUnit)"
}

// FilterBySAM matches a specific sAMAccountName.
func FilterBySAM(sam string) string {
	return fmt.Sprintf("(sAMAccountName=%s)", goldap.EscapeFilter(sam))
}

// FilterByUPN matches a specific userPrincipalName.
func FilterByUPN(upn string) string {
	return fmt.Sprintf("(userPrincipalName=%s)", goldap.EscapeFilter(upn))
}

// FilterTextSearch builds an OR filter searching across multiple attributes.
func FilterTextSearch(query string, baseFilter string) string {
	escaped := goldap.EscapeFilter(query)
	searchFields := []string{
		"displayName", "sAMAccountName", "mail", "cn", "description",
	}
	var conditions []string
	for _, field := range searchFields {
		conditions = append(conditions, fmt.Sprintf("(%s=*%s*)", field, escaped))
	}
	return fmt.Sprintf("(&%s(|%s))", baseFilter, strings.Join(conditions, ""))
}
