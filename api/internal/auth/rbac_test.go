package auth

import "testing"

func TestExtractCN(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"CN=Domain Admins,CN=Users,DC=dzsec,DC=net", "Domain Admins"},
		{"CN=DnsAdmins,CN=Users,DC=dzsec,DC=net", "DnsAdmins"},
		{"CN=Account Operators,CN=Builtin,DC=dzsec,DC=net", "Account Operators"},
		{"Domain Admins", "Domain Admins"},
		{"cn=lowercase,DC=test,DC=com", "lowercase"},
		{"CN=single", "single"},
		{"", ""},
	}

	for _, tt := range tests {
		got := extractCN(tt.input)
		if got != tt.want {
			t.Errorf("extractCN(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestHasRole(t *testing.T) {
	tests := []struct {
		name   string
		groups []string
		role   Role
		want   bool
	}{
		{
			name:   "nil session",
			groups: nil,
			role:   RoleAuthenticated,
			want:   false,
		},
		{
			name:   "authenticated role always passes",
			groups: []string{"Domain Users"},
			role:   RoleAuthenticated,
			want:   true,
		},
		{
			name:   "domain admin has admin role",
			groups: []string{"CN=Domain Admins,CN=Users,DC=dzsec,DC=net"},
			role:   RoleAdmin,
			want:   true,
		},
		{
			name:   "bare CN domain admin has admin role",
			groups: []string{"Domain Admins"},
			role:   RoleAdmin,
			want:   true,
		},
		{
			name:   "enterprise admin has admin role",
			groups: []string{"Enterprise Admins"},
			role:   RoleAdmin,
			want:   true,
		},
		{
			name:   "domain admin has operator role",
			groups: []string{"Domain Admins"},
			role:   RoleOperator,
			want:   true,
		},
		{
			name:   "domain admin has dns admin role",
			groups: []string{"Domain Admins"},
			role:   RoleDNSAdmin,
			want:   true,
		},
		{
			name:   "account operator has operator role",
			groups: []string{"Account Operators"},
			role:   RoleOperator,
			want:   true,
		},
		{
			name:   "account operator lacks admin role",
			groups: []string{"Account Operators"},
			role:   RoleAdmin,
			want:   false,
		},
		{
			name:   "dns admin has dns admin role",
			groups: []string{"DnsAdmins"},
			role:   RoleDNSAdmin,
			want:   true,
		},
		{
			name:   "dns admin lacks operator role",
			groups: []string{"DnsAdmins"},
			role:   RoleOperator,
			want:   false,
		},
		{
			name:   "regular user lacks admin role",
			groups: []string{"Domain Users"},
			role:   RoleAdmin,
			want:   false,
		},
		{
			name:   "case insensitive match",
			groups: []string{"domain admins"},
			role:   RoleAdmin,
			want:   true,
		},
		{
			name:   "full DN with case insensitive match",
			groups: []string{"CN=domain admins,CN=Users,DC=dzsec,DC=net"},
			role:   RoleAdmin,
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sess *Session
			if tt.groups != nil {
				sess = &Session{
					Username: "testuser",
					Groups:   tt.groups,
				}
			}
			got := HasRole(sess, tt.role)
			if got != tt.want {
				t.Errorf("HasRole(%v, %d) = %v, want %v", tt.groups, tt.role, got, tt.want)
			}
		})
	}
}
