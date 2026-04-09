package validate

import "testing"

func TestSAMAccountName(t *testing.T) {
	valid := []string{"jdoe", "John.Doe", "admin-01", "user_test", "A", "myhost$", "WORKSTATION01$"}
	for _, v := range valid {
		if err := SAMAccountName(v); err != nil {
			t.Errorf("SAMAccountName(%q) should be valid, got %v", v, err)
		}
	}

	invalid := []string{"", ".dot", "-dash", " space", "a" + string(make([]byte, 64))}
	for _, v := range invalid {
		if err := SAMAccountName(v); err == nil {
			t.Errorf("SAMAccountName(%q) should be invalid", v)
		}
	}
}

func TestGroupName(t *testing.T) {
	valid := []string{"Domain Admins", "IT-Staff", "Group.Name", "A"}
	for _, v := range valid {
		if err := GroupName(v); err != nil {
			t.Errorf("GroupName(%q) should be valid, got %v", v, err)
		}
	}

	invalid := []string{"", " leading", "-dash"}
	for _, v := range invalid {
		if err := GroupName(v); err == nil {
			t.Errorf("GroupName(%q) should be invalid", v)
		}
	}
}

func TestDNSType(t *testing.T) {
	valid := []string{"A", "AAAA", "CNAME", "MX", "NS", "PTR", "SRV", "TXT", "SOA", "a", "cname"}
	for _, v := range valid {
		if err := DNSType(v); err != nil {
			t.Errorf("DNSType(%q) should be valid, got %v", v, err)
		}
	}

	invalid := []string{"", "AXFR", "ANY", "INVALID"}
	for _, v := range invalid {
		if err := DNSType(v); err == nil {
			t.Errorf("DNSType(%q) should be invalid", v)
		}
	}
}

func TestDNSName(t *testing.T) {
	valid := []string{"www", "mail.example.com", "*.example.com", "@", "test-host"}
	for _, v := range valid {
		if err := DNSName(v); err != nil {
			t.Errorf("DNSName(%q) should be valid, got %v", v, err)
		}
	}

	invalid := []string{"", "host name with spaces", "host;injection"}
	for _, v := range invalid {
		if err := DNSName(v); err == nil {
			t.Errorf("DNSName(%q) should be invalid", v)
		}
	}
}

func TestSPN(t *testing.T) {
	valid := []string{"HTTP/web.example.com", "cifs/fileserver.test.com", "MSSQLSvc/sql01.example.com"}
	for _, v := range valid {
		if err := SPN(v); err != nil {
			t.Errorf("SPN(%q) should be valid, got %v", v, err)
		}
	}

	invalid := []string{"", "noslash", "/notype", "bad spaces/host", "1bad/host"}
	for _, v := range invalid {
		if err := SPN(v); err == nil {
			t.Errorf("SPN(%q) should be invalid", v)
		}
	}
}

func TestPassword(t *testing.T) {
	valid := []string{"Pass123!", "Complex-P@ss_w0rd!", "short"}
	for _, v := range valid {
		if err := Password(v); err != nil {
			t.Errorf("Password should be valid, got %v", err)
		}
	}

	if err := Password(""); err == nil {
		t.Error("empty password should be invalid")
	}
	if err := Password("has\x00null"); err == nil {
		t.Error("password with null byte should be invalid")
	}
	if err := Password(string(make([]byte, 129))); err == nil {
		t.Error("password over 128 chars should be invalid")
	}
}

func TestNoFlagInjection(t *testing.T) {
	if err := NoFlagInjection("normal", "test"); err != nil {
		t.Errorf("normal string should be valid, got %v", err)
	}
	if err := NoFlagInjection("-flag", "test"); err == nil {
		t.Error("string starting with hyphen should be rejected")
	}
}

func TestRawFilter(t *testing.T) {
	valid := []string{"(objectClass=user)", "(&(cn=test)(objectClass=*))", ""}
	for _, v := range valid {
		if err := RawFilter(v); err != nil {
			t.Errorf("RawFilter(%q) should be valid, got %v", v, err)
		}
	}

	if err := RawFilter("(unbalanced"); err == nil {
		t.Error("unbalanced parens should be invalid")
	}
	if err := RawFilter("has\x00null"); err == nil {
		t.Error("filter with null byte should be invalid")
	}
	if err := RawFilter(string(make([]byte, 2049))); err == nil {
		t.Error("filter over 2048 chars should be invalid")
	}
}

func TestAttributes(t *testing.T) {
	if err := Attributes([]string{"cn", "sAMAccountName", "mail"}); err != nil {
		t.Errorf("normal attributes should be valid, got %v", err)
	}
	if err := Attributes([]string{"cn", "unicodePwd"}); err == nil {
		t.Error("unicodePwd should be rejected")
	}
	if err := Attributes([]string{"supplementalCredentials"}); err == nil {
		t.Error("supplementalCredentials should be rejected")
	}
}

func TestBaseDN(t *testing.T) {
	if err := BaseDN("CN=Users,DC=example,DC=com", "DC=example,DC=com"); err != nil {
		t.Errorf("valid baseDN should pass, got %v", err)
	}
	if err := BaseDN("CN=Users,DC=evil,DC=com", "DC=example,DC=com"); err == nil {
		t.Error("baseDN not ending in domain suffix should be rejected")
	}
	if err := BaseDN("", "DC=example,DC=com"); err == nil {
		t.Error("empty baseDN should be rejected")
	}
}
