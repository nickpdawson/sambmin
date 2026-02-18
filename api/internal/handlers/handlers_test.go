package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nickdawson/sambmin/internal/auth"
	"github.com/nickdawson/sambmin/internal/config"
)

func setupTestAuth(t *testing.T) (*auth.Store, *auth.Session) {
	t.Helper()
	store, err := auth.NewStore(1)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	sessionStore = store

	sess, err := store.Create("testadmin", "CN=testadmin,CN=Users,DC=test,DC=com", []string{"Domain Admins"}, "TestPass123!")
	if err != nil {
		t.Fatalf("Create session: %v", err)
	}

	handlerConfig = &config.Config{
		BaseDN: "DC=test,DC=com",
	}

	return store, sess
}

func addSessionCookie(r *http.Request, sess *auth.Session) {
	r.AddCookie(&http.Cookie{
		Name:  sessionCookieName,
		Value: sess.ID,
	})
}

func TestHandleHealth(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()

	handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("status = %q, want %q", resp["status"], "ok")
	}
}

func TestRequireSessionNoAuth(t *testing.T) {
	store, _ := auth.NewStore(1)
	sessionStore = store

	req := httptest.NewRequest("POST", "/api/users", nil)
	w := httptest.NewRecorder()

	sess := requireSession(w, req)
	if sess != nil {
		t.Error("requireSession should return nil without auth")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestRequireSessionWithAuth(t *testing.T) {
	_, sess := setupTestAuth(t)

	req := httptest.NewRequest("POST", "/api/users", nil)
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	got := requireSession(w, req)
	if got == nil {
		t.Fatal("requireSession should return session with valid cookie")
	}
	if got.Username != "testadmin" {
		t.Errorf("username = %q, want %q", got.Username, "testadmin")
	}
}

func TestHandleCreateUserNoAuth(t *testing.T) {
	store, _ := auth.NewStore(1)
	sessionStore = store

	body := `{"username":"newuser","password":"Pass123!"}`
	req := httptest.NewRequest("POST", "/api/users", strings.NewReader(body))
	w := httptest.NewRecorder()

	handleCreateUser(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleCreateUserBadRequest(t *testing.T) {
	_, sess := setupTestAuth(t)

	// Missing required fields
	body := `{"username":"","password":""}`
	req := httptest.NewRequest("POST", "/api/users", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleCreateUser(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleResetPasswordNoAuth(t *testing.T) {
	store, _ := auth.NewStore(1)
	sessionStore = store

	body := `{"password":"NewPass123!"}`
	req := httptest.NewRequest("POST", "/api/users/CN=jdoe,CN=Users,DC=test,DC=com/reset-password", strings.NewReader(body))
	w := httptest.NewRecorder()

	handleResetPassword(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleCreateGroupNoAuth(t *testing.T) {
	store, _ := auth.NewStore(1)
	sessionStore = store

	body := `{"name":"TestGroup"}`
	req := httptest.NewRequest("POST", "/api/groups", strings.NewReader(body))
	w := httptest.NewRecorder()

	handleCreateGroup(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleCreateGroupBadRequest(t *testing.T) {
	_, sess := setupTestAuth(t)

	body := `{"name":""}`
	req := httptest.NewRequest("POST", "/api/groups", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleCreateGroup(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestCnFromDN(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"CN=jdoe,CN=Users,DC=dzsec,DC=net", "jdoe"},
		{"CN=Domain Admins,CN=Users,DC=dzsec,DC=net", "Domain Admins"},
		{"OU=Sales,DC=dzsec,DC=net", ""},
		{"", ""},
	}

	for _, tt := range tests {
		got := cnFromDN(tt.input)
		if got != tt.want {
			t.Errorf("cnFromDN(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestRespondJSON(t *testing.T) {
	w := httptest.NewRecorder()
	respondJSON(w, http.StatusOK, map[string]string{"key": "value"})

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["key"] != "value" {
		t.Errorf("key = %q, want %q", resp["key"], "value")
	}
}

func TestRespondError(t *testing.T) {
	w := httptest.NewRecorder()
	respondError(w, http.StatusBadRequest, "test error")

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] != "test error" {
		t.Errorf("error = %q, want %q", resp["error"], "test error")
	}
}
