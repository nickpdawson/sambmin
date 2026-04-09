package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSettingsPath(t *testing.T) {
	got := SettingsPath("/etc/sambmin/config.yaml")
	want := "/etc/sambmin/settings.json"
	if got != want {
		t.Errorf("SettingsPath = %q, want %q", got, want)
	}
}

func TestNewSettingsStore_Empty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	store, err := NewSettingsStore(path)
	if err != nil {
		t.Fatalf("NewSettingsStore: %v", err)
	}

	overlay := store.Overlay()
	if overlay.Connection != nil || overlay.Auth != nil || overlay.RBAC != nil || overlay.Application != nil {
		t.Error("expected empty overlay for nonexistent file")
	}
}

func TestNewSettingsStore_ExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	data := `{"auth":{"session_timeout":12}}`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	store, err := NewSettingsStore(path)
	if err != nil {
		t.Fatalf("NewSettingsStore: %v", err)
	}

	overlay := store.Overlay()
	if overlay.Auth == nil || overlay.Auth.SessionTimeout == nil || *overlay.Auth.SessionTimeout != 12 {
		t.Error("expected session_timeout=12 from loaded file")
	}
}

func TestSaveAndReload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	store, err := NewSettingsStore(path)
	if err != nil {
		t.Fatal(err)
	}

	timeout := 4
	store.UpdateAuth(AuthOverlay{SessionTimeout: &timeout})

	if err := store.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify file exists and is valid JSON
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var loaded SettingsOverlay
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if loaded.Auth == nil || loaded.Auth.SessionTimeout == nil || *loaded.Auth.SessionTimeout != 4 {
		t.Error("saved file should have session_timeout=4")
	}

	// Reload into a new store
	store2, err := NewSettingsStore(path)
	if err != nil {
		t.Fatalf("NewSettingsStore reload: %v", err)
	}
	overlay := store2.Overlay()
	if overlay.Auth == nil || overlay.Auth.SessionTimeout == nil || *overlay.Auth.SessionTimeout != 4 {
		t.Error("reloaded store should have session_timeout=4")
	}
}

func TestApplyTo(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	store, _ := NewSettingsStore(path)

	base := &Config{
		BaseDN:         "DC=old,DC=com",
		SessionTimeout: 8,
		DCs: []DCConfig{
			{Hostname: "dc1", Address: "1.1.1.1", Port: 636},
		},
	}

	// Apply with connection overlay
	store.UpdateConnection(ConnectionOverlay{
		DCs:    []DCConfig{{Hostname: "dc2", Address: "2.2.2.2", Port: 636}},
		BaseDN: "DC=new,DC=com",
	})
	timeout := 12
	store.UpdateAuth(AuthOverlay{SessionTimeout: &timeout})

	merged := store.ApplyTo(base)

	// Base should be unchanged
	if base.BaseDN != "DC=old,DC=com" {
		t.Error("base config was mutated")
	}
	if base.SessionTimeout != 8 {
		t.Error("base session timeout was mutated")
	}

	// Merged should have overlay values
	if merged.BaseDN != "DC=new,DC=com" {
		t.Errorf("merged BaseDN = %q, want DC=new,DC=com", merged.BaseDN)
	}
	if merged.SessionTimeout != 12 {
		t.Errorf("merged SessionTimeout = %d, want 12", merged.SessionTimeout)
	}
	if len(merged.DCs) != 1 || merged.DCs[0].Hostname != "dc2" {
		t.Error("merged DCs should reflect overlay")
	}
}

func TestUpdateConnection_RestartFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	store, _ := NewSettingsStore(path)

	// First update — everything is new
	fields := store.UpdateConnection(ConnectionOverlay{
		DCs:    []DCConfig{{Hostname: "dc1", Address: "1.1.1.1", Port: 636}},
		BaseDN: "DC=test,DC=com",
	})
	if len(fields) != 2 {
		t.Errorf("first update should have 2 restart fields, got %d: %v", len(fields), fields)
	}

	// Same values — no restart needed
	fields = store.UpdateConnection(ConnectionOverlay{
		DCs:    []DCConfig{{Hostname: "dc1", Address: "1.1.1.1", Port: 636}},
		BaseDN: "DC=test,DC=com",
	})
	if len(fields) != 0 {
		t.Errorf("same values should have 0 restart fields, got %d: %v", len(fields), fields)
	}

	// Change BaseDN only
	fields = store.UpdateConnection(ConnectionOverlay{
		DCs:    []DCConfig{{Hostname: "dc1", Address: "1.1.1.1", Port: 636}},
		BaseDN: "DC=other,DC=com",
	})
	if len(fields) != 1 || fields[0] != "base_dn" {
		t.Errorf("BaseDN change should trigger base_dn restart, got %v", fields)
	}
}

func TestUpdateAuth_RestartFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	store, _ := NewSettingsStore(path)

	// Session timeout change — no restart
	timeout := 4
	fields := store.UpdateAuth(AuthOverlay{SessionTimeout: &timeout})
	if len(fields) != 0 {
		t.Errorf("session timeout change should not require restart, got %v", fields)
	}

	// Kerberos change — restart
	enabled := true
	fields = store.UpdateAuth(AuthOverlay{
		Kerberos: &KerberosOverlay{
			Enabled:    &enabled,
			KeytabPath: "/etc/krb5.keytab",
		},
	})
	if len(fields) != 1 || fields[0] != "kerberos" {
		t.Errorf("kerberos change should trigger restart, got %v", fields)
	}
}

func TestUpdateRBAC(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	store, _ := NewSettingsStore(path)

	store.UpdateRBAC(RBACOverlay{
		Roles: []RoleMapping{
			{Role: "Admin", Groups: []string{"Domain Admins"}, Permissions: []string{"*"}},
		},
	})

	overlay := store.Overlay()
	if overlay.RBAC == nil || len(overlay.RBAC.Roles) != 1 {
		t.Error("RBAC overlay should have 1 role")
	}
	if overlay.RBAC.Roles[0].Role != "Admin" {
		t.Errorf("role = %q, want Admin", overlay.RBAC.Roles[0].Role)
	}
}

func TestUpdateApplication(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	store, _ := NewSettingsStore(path)

	days := 30
	store.UpdateApplication(ApplicationOverlay{AuditRetentionDays: &days})

	overlay := store.Overlay()
	if overlay.Application == nil || overlay.Application.AuditRetentionDays == nil || *overlay.Application.AuditRetentionDays != 30 {
		t.Error("application overlay should have audit_retention_days=30")
	}
}

func TestSave_AtomicPermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	store, _ := NewSettingsStore(path)

	timeout := 8
	store.UpdateAuth(AuthOverlay{SessionTimeout: &timeout})
	if err := store.Save(); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	perm := info.Mode().Perm()
	if perm != 0640 {
		t.Errorf("file permissions = %o, want 0640", perm)
	}
}
