package handlers

import (
	"strings"
	"testing"
)

// Real descriptor shape captured from `samba-tool dsacl get` on Samba 4.23,
// trimmed to a representative subset. The first ACE is an explicitly-applied
// reset-password delegation to a domain principal; the rest are class defaults
// (SY/DA, non-inherited) and inherited ACEs (CIID).
const sampleSDDL = `O:DAG:DAD:AI` +
	`(OA;CI;CR;00299570-246d-11d0-a768-00aa006e0529;bf967aba-0de6-11d0-a285-00aa003049e2;S-1-5-21-2987504718-2361157560-2967585114-1112)` +
	`(OA;CI;WP;bf967a0a-0de6-11d0-a285-00aa003049e2;bf967aba-0de6-11d0-a285-00aa003049e2;S-1-5-21-2987504718-2361157560-2967585114-1112)` +
	`(A;;CCDCLCSWRPWPDTLOCRSDRCWDWO;;;SY)` +
	`(A;;CCDCLCSWRPWPDTLOCRSDRCWDWO;;;DA)` +
	`(A;CIID;CCDCLCSWRPWPDTLOCRSDRCWDWO;;;EA)` +
	`(A;CIID;LC;;;RU)` +
	`S:AI(OU;CIIDSA;WP;f30e3bbe-9ff0-11d1-b603-0000f80367c1;bf967aa5-0de6-11d0-a285-00aa003049e2;WD)`

func TestSDDLComponent(t *testing.T) {
	dacl := sddlComponent(sampleSDDL, 'D')
	if !strings.HasPrefix(dacl, "AI(OA;CI;CR;") {
		t.Fatalf("DACL component wrong start: %q", dacl[:20])
	}
	if strings.Contains(dacl, "(OU;CIIDSA;WP;") {
		t.Error("DACL leaked into the SACL (S:) section")
	}
	sacl := sddlComponent(sampleSDDL, 'S')
	if !strings.Contains(sacl, "(OU;CIIDSA;WP;") {
		t.Error("SACL component missing its ACE")
	}
	if sddlComponent(sampleSDDL, 'O') != "DA" {
		t.Errorf("owner = %q, want DA", sddlComponent(sampleSDDL, 'O'))
	}
}

func TestParseDACL(t *testing.T) {
	aces := parseDACL(sampleSDDL)
	if len(aces) != 6 {
		t.Fatalf("expected 6 DACL ACEs, got %d", len(aces))
	}

	// First ACE: explicit reset-password to a domain principal.
	first := aces[0]
	if first.inherited {
		t.Error("reset-password ACE should not be inherited (flags CI, no ID)")
	}
	if first.trustee != "S-1-5-21-2987504718-2361157560-2967585114-1112" {
		t.Errorf("trustee = %q", first.trustee)
	}
	if first.rights != "CR" || first.objectGUID != guidResetPassword {
		t.Errorf("rights=%q objectGUID=%q", first.rights, first.objectGUID)
	}
	if first.raw != "(OA;CI;CR;"+guidResetPassword+";"+guidClassUser+";S-1-5-21-2987504718-2361157560-2967585114-1112)" {
		t.Errorf("raw ACE not round-trippable: %q", first.raw)
	}

	// The EA ACE (CIID) must be flagged inherited.
	var sawInherited bool
	for _, a := range aces {
		if a.trustee == "EA" {
			if !a.inherited {
				t.Error("EA ACE has CIID flags — should be inherited")
			}
			sawInherited = true
		}
	}
	if !sawInherited {
		t.Error("did not find the inherited EA ACE")
	}
}

func TestParseDACL_TemplateAttribution(t *testing.T) {
	aces := parseDACL(sampleSDDL)
	// Both explicit domain-principal ACEs belong to reset_passwords.
	for _, a := range aces[:2] {
		key := aceBodyToTemplate[aceBodyKey(a.raw)]
		if key != "reset_passwords" {
			t.Errorf("ACE %q attributed to %q, want reset_passwords", a.raw, key)
		}
	}
}

func TestDescribeACE(t *testing.T) {
	tests := []struct {
		name string
		ace  parsedACE
		want string
	}{
		{
			"reset password",
			parsedACE{aceType: "OA", rights: "CR", objectGUID: guidResetPassword, inheritGUID: guidClassUser},
			"Reset Password (on user objects)",
		},
		{
			"full control",
			parsedACE{aceType: "A", rights: "GA"},
			"Full control",
		},
		{
			"manage membership",
			parsedACE{aceType: "OA", rights: "RPWP", objectGUID: guidAttrMember, inheritGUID: guidClassGroup},
			"read property + write property of member (on group objects)",
		},
		{
			"replicate all",
			parsedACE{aceType: "OA", rights: "CR", objectGUID: guidReplChangesAll},
			"Replicating Directory Changes (All)",
		},
		{
			"deny full control",
			parsedACE{aceType: "D", rights: "GA"},
			"DENY: Full control",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := describeACE(tt.ace); got != tt.want {
				t.Errorf("describeACE = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderACEs(t *testing.T) {
	tmpl := templateByKey["reset_passwords"]
	got := renderACEs(tmpl.aces, "S-1-5-21-1-2-3-1104")
	want := "(OA;CI;CR;" + guidResetPassword + ";" + guidClassUser + ";S-1-5-21-1-2-3-1104)" +
		"(OA;CI;WP;" + guidAttrPwdLastSet + ";" + guidClassUser + ";S-1-5-21-1-2-3-1104)"
	if got != want {
		t.Errorf("renderACEs =\n %q\nwant\n %q", got, want)
	}
	if strings.Contains(got, "%SID%") {
		t.Error("placeholder not substituted")
	}
}

func TestCARTemplateAttribution(t *testing.T) {
	// The ACE samba writes for get-changes-all must map back to the template.
	raw := "(OA;;CR;" + guidReplChangesAll + ";;S-1-5-21-1-2-3-1120)"
	if key := aceBodyToTemplate[aceBodyKey(raw)]; key != "replicate_changes_all" {
		t.Errorf("CAR ACE attributed to %q, want replicate_changes_all", key)
	}
}

func TestFullControlCanonicalization(t *testing.T) {
	// A container-inheritable Generic All is stored as two ACEs; both must
	// attribute back to full_control, and describeACE must call the expanded
	// mask "Full control" too.
	inheritOnly := "(A;CIIO;GA;;;S-1-5-21-1-2-3-1104)"
	expanded := "(A;;" + fullControlMask + ";;;S-1-5-21-1-2-3-1104)"
	if k := aceBodyToTemplate[aceBodyKey(inheritOnly)]; k != "full_control" {
		t.Errorf("inherit-only GA attributed to %q, want full_control", k)
	}
	if k := aceBodyToTemplate[aceBodyKey(expanded)]; k != "full_control" {
		t.Errorf("expanded mask attributed to %q, want full_control", k)
	}
	got := describeACE(parsedACE{aceType: "A", rights: fullControlMask})
	if got != "Full control" {
		t.Errorf("expanded mask describes as %q, want Full control", got)
	}
}

func TestExtractParenGroups(t *testing.T) {
	got := extractParenGroups("AI(A;;X;;;SY)(A;;Y;;;DA)")
	if len(got) != 2 || got[0] != "A;;X;;;SY" || got[1] != "A;;Y;;;DA" {
		t.Errorf("extractParenGroups = %#v", got)
	}
}

func TestIsDomainPrincipalSID(t *testing.T) {
	if !isDomainPrincipalSID("S-1-5-21-1-2-3-1104") {
		t.Error("domain SID should be a principal")
	}
	for _, tok := range []string{"SY", "DA", "S-1-5-18", "S-1-1-0"} {
		if isDomainPrincipalSID(tok) {
			t.Errorf("%q should not be a domain principal", tok)
		}
	}
}
