package auth

import (
	"testing"
	"time"
)

func TestNewStore(t *testing.T) {
	store, err := NewStore(8)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	if store == nil {
		t.Fatal("NewStore returned nil")
	}
}

func TestStoreCreateAndGet(t *testing.T) {
	store, err := NewStore(1)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	sess, err := store.Create("testuser", "CN=testuser,DC=test,DC=com", []string{"Domain Users"}, "secret123")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if sess.ID == "" {
		t.Error("session ID is empty")
	}
	if sess.Username != "testuser" {
		t.Errorf("username = %q, want %q", sess.Username, "testuser")
	}
	if sess.DN != "CN=testuser,DC=test,DC=com" {
		t.Errorf("DN = %q, want %q", sess.DN, "CN=testuser,DC=test,DC=com")
	}
	if len(sess.Groups) != 1 || sess.Groups[0] != "Domain Users" {
		t.Errorf("groups = %v, want [Domain Users]", sess.Groups)
	}

	// Get should return the session
	got := store.Get(sess.ID)
	if got == nil {
		t.Fatal("Get returned nil for valid session")
	}
	if got.Username != "testuser" {
		t.Errorf("Get username = %q, want %q", got.Username, "testuser")
	}
}

func TestStoreGetNonexistent(t *testing.T) {
	store, err := NewStore(1)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	got := store.Get("nonexistent-session-id")
	if got != nil {
		t.Error("Get should return nil for nonexistent session")
	}
}

func TestStoreDelete(t *testing.T) {
	store, err := NewStore(1)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	sess, err := store.Create("testuser", "CN=testuser,DC=test,DC=com", nil, "pass")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	store.Delete(sess.ID)

	got := store.Get(sess.ID)
	if got != nil {
		t.Error("Get should return nil after Delete")
	}
}

func TestStorePasswordEncryptDecrypt(t *testing.T) {
	store, err := NewStore(1)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	password := "MyS3cretP@ssw0rd!"
	sess, err := store.Create("testuser", "CN=testuser,DC=test,DC=com", nil, password)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Encrypted password should not be empty
	if len(sess.encPW) == 0 {
		t.Error("encrypted password is empty")
	}

	// Decrypt should return original password
	got, err := store.Password(sess)
	if err != nil {
		t.Fatalf("Password: %v", err)
	}
	if got != password {
		t.Errorf("Password = %q, want %q", got, password)
	}
}

func TestStorePasswordNilSession(t *testing.T) {
	store, err := NewStore(1)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	_, err = store.Password(nil)
	if err == nil {
		t.Error("Password(nil) should return error")
	}
}

func TestStoreExpiredSession(t *testing.T) {
	store, err := NewStore(1)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	sess, err := store.Create("testuser", "CN=testuser,DC=test,DC=com", nil, "pass")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Manually expire the session
	store.mu.Lock()
	sess.Expires = time.Now().Add(-1 * time.Hour)
	store.mu.Unlock()

	got := store.Get(sess.ID)
	if got != nil {
		t.Error("Get should return nil for expired session")
	}
}

func TestStoreDefaultTimeout(t *testing.T) {
	// 0 timeout should default to 8 hours
	store, err := NewStore(0)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	if store.timeout != 8*time.Hour {
		t.Errorf("default timeout = %v, want %v", store.timeout, 8*time.Hour)
	}
}

func TestGenerateSessionID(t *testing.T) {
	id1, err := generateSessionID()
	if err != nil {
		t.Fatalf("generateSessionID: %v", err)
	}
	id2, err := generateSessionID()
	if err != nil {
		t.Fatalf("generateSessionID: %v", err)
	}

	if id1 == "" {
		t.Error("session ID is empty")
	}
	if len(id1) != 64 { // 32 bytes hex-encoded
		t.Errorf("session ID length = %d, want 64", len(id1))
	}
	if id1 == id2 {
		t.Error("two generated session IDs should not be identical")
	}
}

func TestDnToDomain(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"DC=example,DC=com", "example.com"},
		{"DC=example,DC=com", "example.com"},
		{"DC=sub,DC=example,DC=com", "sub.example.com"},
		{"", ""},
	}

	for _, tt := range tests {
		got := dnToDomain(tt.input)
		if got != tt.want {
			t.Errorf("dnToDomain(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
