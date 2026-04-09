package auth

import (
	"net/http"
	"strings"
)

// Role represents a required authorization level.
type Role int

const (
	// RoleAuthenticated — any logged-in user (read endpoints, self-service).
	RoleAuthenticated Role = iota
	// RoleOperator — Account Operators + Domain Admins (user/group/computer/contact/OU CRUD).
	RoleOperator
	// RoleDNSAdmin — DnsAdmins + Domain Admins (DNS mutations).
	RoleDNSAdmin
	// RoleAdmin — Domain Admins / Enterprise Admins (password policy, GPO, SPN, delegation, FSMO, replication, keytab).
	RoleAdmin
)

// roleGroups maps each role to the AD group CNs that satisfy it.
var roleGroups = map[Role][]string{
	RoleOperator: {"Account Operators", "Domain Admins", "Enterprise Admins"},
	RoleDNSAdmin: {"DnsAdmins", "Domain Admins", "Enterprise Admins"},
	RoleAdmin:    {"Domain Admins", "Enterprise Admins"},
}

// HasRole checks whether the session's group memberships satisfy the given role.
func HasRole(sess *Session, role Role) bool {
	if sess == nil {
		return false
	}
	if role == RoleAuthenticated {
		return true
	}

	required, ok := roleGroups[role]
	if !ok {
		return false
	}

	for _, group := range sess.Groups {
		cn := extractCN(group)
		for _, req := range required {
			if strings.EqualFold(cn, req) {
				return true
			}
		}
	}
	return false
}

// extractCN pulls the CN value from a full DN string, or returns the
// input unchanged if it's already a bare CN (for test compatibility).
//
//	"CN=Domain Admins,CN=Users,DC=example,DC=com" → "Domain Admins"
//	"Domain Admins" → "Domain Admins"
func extractCN(group string) string {
	upper := strings.ToUpper(group)
	if !strings.HasPrefix(upper, "CN=") {
		return group // bare CN
	}
	// Find the first comma after "CN="
	rest := group[3:]
	if idx := strings.IndexByte(rest, ','); idx >= 0 {
		return rest[:idx]
	}
	return rest
}

// RequireRole returns middleware that checks the session (from context) has
// the given role. Returns 403 if the user lacks the required role.
func RequireRole(role Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sess, _ := r.Context().Value(SessionKey).(*Session)
			if sess == nil {
				http.Error(w, `{"error":"authentication required"}`, http.StatusUnauthorized)
				return
			}
			if !HasRole(sess, role) {
				http.Error(w, `{"error":"insufficient permissions"}`, http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
