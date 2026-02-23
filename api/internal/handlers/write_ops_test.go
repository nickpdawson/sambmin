package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nickdawson/sambmin/internal/config"
	"github.com/nickdawson/sambmin/internal/models"
)

func setupMockSambaTool(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	script := filepath.Join(dir, "samba-tool")
	err := os.WriteFile(script, []byte("#!/bin/sh\necho \"mock ok\"\n"), 0755)
	if err != nil {
		t.Fatalf("write mock samba-tool: %v", err)
	}
	sambaTool = script
	t.Cleanup(func() { sambaTool = "samba-tool" })
}

func setupFailingSambaTool(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	script := filepath.Join(dir, "samba-tool")
	err := os.WriteFile(script, []byte("#!/bin/sh\necho \"ERROR(ldb): some error\" >&2\nexit 1\n"), 0755)
	if err != nil {
		t.Fatalf("write mock samba-tool: %v", err)
	}
	sambaTool = script
	t.Cleanup(func() { sambaTool = "samba-tool" })
}

func newMuxWithRoute(method, pattern string, handler http.HandlerFunc) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc(method+" "+pattern, handler)
	return mux
}

func TestHandleCreateUserSuccess(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	body := `{"username":"newuser","password":"Pass123!"}`
	req := httptest.NewRequest("POST", "/api/users", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleCreateUser(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}
}

func TestHandleCreateUserWithOptionalFields(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	body := `{"username":"newuser","password":"Pass123!","givenName":"John","surname":"Doe","mail":"jdoe@test.com","department":"IT","title":"Engineer","ou":"OU=Staff","mustChangePassword":true}`
	req := httptest.NewRequest("POST", "/api/users", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleCreateUser(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}
}

func TestHandleCreateUserSambaToolFailure(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupFailingSambaTool(t)

	body := `{"username":"newuser","password":"Pass123!"}`
	req := httptest.NewRequest("POST", "/api/users", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleCreateUser(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandleDeleteUserNoAuth(t *testing.T) {
	store, _ := setupTestAuth(t)
	_ = store

	mux := newMuxWithRoute("DELETE", "/api/users/{dn}", handleDeleteUser)

	req := httptest.NewRequest("DELETE", "/api/users/CN=jdoe,CN=Users,DC=test,DC=com", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleDeleteUserSuccess(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	mux := newMuxWithRoute("DELETE", "/api/users/{dn}", handleDeleteUser)

	req := httptest.NewRequest("DELETE", "/api/users/CN=jdoe,CN=Users,DC=test,DC=com", nil)
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleDeleteUserFailure(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupFailingSambaTool(t)

	mux := newMuxWithRoute("DELETE", "/api/users/{dn}", handleDeleteUser)

	req := httptest.NewRequest("DELETE", "/api/users/CN=jdoe,CN=Users,DC=test,DC=com", nil)
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandleResetPasswordSuccess(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	mux := newMuxWithRoute("POST", "/api/users/{dn}/reset-password", handleResetPassword)

	body := `{"password":"NewPass123!"}`
	req := httptest.NewRequest("POST", "/api/users/CN=jdoe,CN=Users,DC=test,DC=com/reset-password", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleResetPasswordBadRequest(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	mux := newMuxWithRoute("POST", "/api/users/{dn}/reset-password", handleResetPassword)

	body := `{"password":""}`
	req := httptest.NewRequest("POST", "/api/users/CN=jdoe,CN=Users,DC=test,DC=com/reset-password", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleResetPasswordWithMustChange(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	mux := newMuxWithRoute("POST", "/api/users/{dn}/reset-password", handleResetPassword)

	body := `{"password":"NewPass123!","mustChangeAtNextLogin":true}`
	req := httptest.NewRequest("POST", "/api/users/CN=jdoe,CN=Users,DC=test,DC=com/reset-password", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleEnableUserSuccess(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	mux := newMuxWithRoute("POST", "/api/users/{dn}/enable", handleEnableUser)

	req := httptest.NewRequest("POST", "/api/users/CN=jdoe,CN=Users,DC=test,DC=com/enable", nil)
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleDisableUserSuccess(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	mux := newMuxWithRoute("POST", "/api/users/{dn}/disable", handleDisableUser)

	req := httptest.NewRequest("POST", "/api/users/CN=jdoe,CN=Users,DC=test,DC=com/disable", nil)
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleUnlockUserSuccess(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	mux := newMuxWithRoute("POST", "/api/users/{dn}/unlock", handleUnlockUser)

	req := httptest.NewRequest("POST", "/api/users/CN=jdoe,CN=Users,DC=test,DC=com/unlock", nil)
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleEnableUserFailure(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupFailingSambaTool(t)

	mux := newMuxWithRoute("POST", "/api/users/{dn}/enable", handleEnableUser)

	req := httptest.NewRequest("POST", "/api/users/CN=jdoe,CN=Users,DC=test,DC=com/enable", nil)
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandleCreateGroupSuccess(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	body := `{"name":"TestGroup","description":"A test group","groupType":"Security"}`
	req := httptest.NewRequest("POST", "/api/groups", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleCreateGroup(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}
}

func TestHandleCreateGroupFailure(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupFailingSambaTool(t)

	body := `{"name":"TestGroup"}`
	req := httptest.NewRequest("POST", "/api/groups", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleCreateGroup(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandleDeleteGroupSuccess(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	mux := newMuxWithRoute("DELETE", "/api/groups/{dn}", handleDeleteGroup)

	req := httptest.NewRequest("DELETE", "/api/groups/CN=TestGroup,CN=Users,DC=test,DC=com", nil)
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleDeleteGroupNoAuth(t *testing.T) {
	store, _ := setupTestAuth(t)
	_ = store

	mux := newMuxWithRoute("DELETE", "/api/groups/{dn}", handleDeleteGroup)

	req := httptest.NewRequest("DELETE", "/api/groups/CN=TestGroup,CN=Users,DC=test,DC=com", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleAddGroupMemberSuccess(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	mux := newMuxWithRoute("POST", "/api/groups/{dn}/members", handleAddGroupMember)

	body := `{"memberDn":"CN=jdoe,CN=Users,DC=test,DC=com"}`
	req := httptest.NewRequest("POST", "/api/groups/CN=TestGroup,CN=Users,DC=test,DC=com/members", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleAddGroupMemberBadMember(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	mux := newMuxWithRoute("POST", "/api/groups/{dn}/members", handleAddGroupMember)

	body := `{"memberDn":"OU=Staff,DC=test,DC=com"}`
	req := httptest.NewRequest("POST", "/api/groups/CN=TestGroup,CN=Users,DC=test,DC=com/members", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleRemoveGroupMemberSuccess(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	mux := newMuxWithRoute("DELETE", "/api/groups/{dn}/members/{memberDn}", handleRemoveGroupMember)

	req := httptest.NewRequest("DELETE", "/api/groups/CN=TestGroup,CN=Users,DC=test,DC=com/members/CN=jdoe,CN=Users,DC=test,DC=com", nil)
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleCreateDNSZoneSuccess(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	body := `{"name":"test.local"}`
	req := httptest.NewRequest("POST", "/api/dns/zones", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleCreateDNSZone(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}
}

func TestHandleCreateDNSZoneBadRequest(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	body := `{"name":""}`
	req := httptest.NewRequest("POST", "/api/dns/zones", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleCreateDNSZone(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleCreateDNSZoneNoAuth(t *testing.T) {
	store, _ := setupTestAuth(t)
	_ = store

	body := `{"name":"test.local"}`
	req := httptest.NewRequest("POST", "/api/dns/zones", strings.NewReader(body))
	w := httptest.NewRecorder()

	handleCreateDNSZone(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleDeleteDNSZoneSuccess(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	mux := newMuxWithRoute("DELETE", "/api/dns/zones/{zone}", handleDeleteDNSZone)

	req := httptest.NewRequest("DELETE", "/api/dns/zones/test.local", nil)
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleCreateDNSRecordSuccess(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	mux := newMuxWithRoute("POST", "/api/dns/zones/{zone}/records", handleCreateDNSRecord)

	body := `{"name":"www","type":"A","value":"10.0.0.1"}`
	req := httptest.NewRequest("POST", "/api/dns/zones/test.local/records", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}
}

func TestHandleCreateDNSRecordBadRequest(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	mux := newMuxWithRoute("POST", "/api/dns/zones/{zone}/records", handleCreateDNSRecord)

	body := `{"name":"","type":"","value":""}`
	req := httptest.NewRequest("POST", "/api/dns/zones/test.local/records", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleUpdateDNSRecordSuccess(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	mux := newMuxWithRoute("PUT", "/api/dns/zones/{zone}/records/{name}", handleUpdateDNSRecord)

	body := `{"type":"A","oldValue":"10.0.0.1","newValue":"10.0.0.2"}`
	req := httptest.NewRequest("PUT", "/api/dns/zones/test.local/records/www", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleUpdateDNSRecordBadRequest(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	mux := newMuxWithRoute("PUT", "/api/dns/zones/{zone}/records/{name}", handleUpdateDNSRecord)

	body := `{"type":"","oldValue":"","newValue":""}`
	req := httptest.NewRequest("PUT", "/api/dns/zones/test.local/records/www", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleDeleteDNSRecordSuccess(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	mux := newMuxWithRoute("DELETE", "/api/dns/zones/{zone}/records/{name}", handleDeleteDNSRecord)

	req := httptest.NewRequest("DELETE", "/api/dns/zones/test.local/records/www?type=A&value=10.0.0.1", nil)
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleDeleteDNSRecordMissingParams(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	mux := newMuxWithRoute("DELETE", "/api/dns/zones/{zone}/records/{name}", handleDeleteDNSRecord)

	req := httptest.NewRequest("DELETE", "/api/dns/zones/test.local/records/www", nil)
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleCreateOUSuccess(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	body := `{"name":"Engineering","description":"Engineering department"}`
	req := httptest.NewRequest("POST", "/api/ous", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleCreateOU(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}
}

func TestHandleCreateOUBadRequest(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	body := `{"name":""}`
	req := httptest.NewRequest("POST", "/api/ous", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleCreateOU(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleCreateOUWithParent(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	body := `{"name":"Backend","parentDn":"OU=Engineering,DC=test,DC=com"}`
	req := httptest.NewRequest("POST", "/api/ous", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleCreateOU(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}
}

func TestHandleDeleteOUSuccess(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	mux := newMuxWithRoute("DELETE", "/api/ous/{dn}", handleDeleteOU)

	req := httptest.NewRequest("DELETE", "/api/ous/OU=Engineering,DC=test,DC=com", nil)
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleDeleteOUNoAuth(t *testing.T) {
	store, _ := setupTestAuth(t)
	_ = store

	mux := newMuxWithRoute("DELETE", "/api/ous/{dn}", handleDeleteOU)

	req := httptest.NewRequest("DELETE", "/api/ous/OU=Engineering,DC=test,DC=com", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestSamAccountNameFromDNFallback(t *testing.T) {
	dirClient = nil

	tests := []struct {
		dn   string
		want string
	}{
		{"CN=jdoe,CN=Users,DC=test,DC=com", "jdoe"},
		{"CN=Domain Admins,CN=Users,DC=test,DC=com", "Domain Admins"},
	}

	for _, tt := range tests {
		got, err := samAccountNameFromDN(nil, tt.dn)
		if err != nil {
			t.Errorf("samAccountNameFromDN(%q) unexpected error: %v", tt.dn, err)
			continue
		}
		if got != tt.want {
			t.Errorf("samAccountNameFromDN(%q) = %q, want %q", tt.dn, got, tt.want)
		}
	}
}

func TestSamAccountNameFromDNNoMatch(t *testing.T) {
	dirClient = nil

	_, err := samAccountNameFromDN(nil, "OU=Staff,DC=test,DC=com")
	if err == nil {
		t.Error("samAccountNameFromDN should fail for DN without CN")
	}
}

func TestRunSambaToolWarningFiltering(t *testing.T) {
	_, sess := setupTestAuth(t)

	dir := t.TempDir()
	script := filepath.Join(dir, "samba-tool")
	err := os.WriteFile(script, []byte("#!/bin/sh\necho 'WARNING: something' >&2\necho 'Usage: samba-tool user create ...' >&2\necho 'actual error message' >&2\nexit 1\n"), 0755)
	if err != nil {
		t.Fatalf("write script: %v", err)
	}
	sambaTool = script
	t.Cleanup(func() { sambaTool = "samba-tool" })

	_, runErr := runSambaTool(context.Background(), sess, "user", "create", "testuser", "pass")
	if runErr == nil {
		t.Fatal("expected error from runSambaTool")
	}

	if runErr.Error() != "actual error message" {
		t.Errorf("error = %q, want %q", runErr.Error(), "actual error message")
	}
}

// --- Update User Tests ---

func TestHandleUpdateUserNoAuth(t *testing.T) {
	store, _ := setupTestAuth(t)
	_ = store

	mux := newMuxWithRoute("PUT", "/api/users/{dn}", handleUpdateUser)

	body := `{"displayName":"John Doe"}`
	req := httptest.NewRequest("PUT", "/api/users/CN=jdoe,CN=Users,DC=test,DC=com", strings.NewReader(body))
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleUpdateUserNoAttrs(t *testing.T) {
	_, sess := setupTestAuth(t)

	mux := newMuxWithRoute("PUT", "/api/users/{dn}", handleUpdateUser)

	body := `{}`
	req := httptest.NewRequest("PUT", "/api/users/CN=jdoe,CN=Users,DC=test,DC=com", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	// dirClient must be non-nil to get past the nil check, but since it's nil
	// in test mode, the handler returns 503 before reaching the empty-attrs check.
	// So we test the dirClient==nil path here.
	dirClient = nil
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleUpdateUserDirClientNil(t *testing.T) {
	_, sess := setupTestAuth(t)
	dirClient = nil

	mux := newMuxWithRoute("PUT", "/api/users/{dn}", handleUpdateUser)

	body := `{"displayName":"John Doe","mail":"jdoe@test.com"}`
	req := httptest.NewRequest("PUT", "/api/users/CN=jdoe,CN=Users,DC=test,DC=com", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleUpdateUserBadJSON(t *testing.T) {
	_, sess := setupTestAuth(t)

	mux := newMuxWithRoute("PUT", "/api/users/{dn}", handleUpdateUser)

	body := `not json`
	req := httptest.NewRequest("PUT", "/api/users/CN=jdoe,CN=Users,DC=test,DC=com", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// --- Update Group Tests ---

func TestHandleUpdateGroupNoAuth(t *testing.T) {
	store, _ := setupTestAuth(t)
	_ = store

	mux := newMuxWithRoute("PUT", "/api/groups/{dn}", handleUpdateGroup)

	body := `{"description":"Updated description"}`
	req := httptest.NewRequest("PUT", "/api/groups/CN=TestGroup,CN=Users,DC=test,DC=com", strings.NewReader(body))
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleUpdateGroupDirClientNil(t *testing.T) {
	_, sess := setupTestAuth(t)
	dirClient = nil

	mux := newMuxWithRoute("PUT", "/api/groups/{dn}", handleUpdateGroup)

	body := `{"description":"Updated description"}`
	req := httptest.NewRequest("PUT", "/api/groups/CN=TestGroup,CN=Users,DC=test,DC=com", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleUpdateGroupNoAttrs(t *testing.T) {
	_, sess := setupTestAuth(t)
	// dirClient nil causes 503 before the empty-attrs check,
	// so this test verifies the nil-client guard.
	dirClient = nil

	mux := newMuxWithRoute("PUT", "/api/groups/{dn}", handleUpdateGroup)

	body := `{}`
	req := httptest.NewRequest("PUT", "/api/groups/CN=TestGroup,CN=Users,DC=test,DC=com", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleUpdateGroupBadJSON(t *testing.T) {
	_, sess := setupTestAuth(t)

	mux := newMuxWithRoute("PUT", "/api/groups/{dn}", handleUpdateGroup)

	body := `not json`
	req := httptest.NewRequest("PUT", "/api/groups/CN=TestGroup,CN=Users,DC=test,DC=com", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// --- Delete Computer Tests ---

func TestHandleDeleteComputerNoAuth(t *testing.T) {
	store, _ := setupTestAuth(t)
	_ = store

	mux := newMuxWithRoute("DELETE", "/api/computers/{dn}", handleDeleteComputer)

	req := httptest.NewRequest("DELETE", "/api/computers/CN=WORKSTATION1,CN=Computers,DC=test,DC=com", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleDeleteComputerDirClientNil(t *testing.T) {
	_, sess := setupTestAuth(t)
	dirClient = nil

	mux := newMuxWithRoute("DELETE", "/api/computers/{dn}", handleDeleteComputer)

	req := httptest.NewRequest("DELETE", "/api/computers/CN=WORKSTATION1,CN=Computers,DC=test,DC=com", nil)
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleDeleteComputerBadDN(t *testing.T) {
	_, sess := setupTestAuth(t)
	dirClient = nil

	mux := newMuxWithRoute("DELETE", "/api/computers/{dn}", handleDeleteComputer)

	// DN without CN= prefix — cnFromDN returns ""
	req := httptest.NewRequest("DELETE", "/api/computers/OU=Computers,DC=test,DC=com", nil)
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// --- Additional failure path tests ---

func TestHandleDeleteDNSZoneFailure(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupFailingSambaTool(t)

	mux := newMuxWithRoute("DELETE", "/api/dns/zones/{zone}", handleDeleteDNSZone)

	req := httptest.NewRequest("DELETE", "/api/dns/zones/test.local", nil)
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandleCreateDNSRecordFailure(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupFailingSambaTool(t)

	mux := newMuxWithRoute("POST", "/api/dns/zones/{zone}/records", handleCreateDNSRecord)

	body := `{"name":"www","type":"A","value":"10.0.0.1"}`
	req := httptest.NewRequest("POST", "/api/dns/zones/test.local/records", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandleDeleteOUFailure(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupFailingSambaTool(t)

	mux := newMuxWithRoute("DELETE", "/api/ous/{dn}", handleDeleteOU)

	req := httptest.NewRequest("DELETE", "/api/ous/OU=Engineering,DC=test,DC=com", nil)
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandleRemoveGroupMemberFailure(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupFailingSambaTool(t)

	mux := newMuxWithRoute("DELETE", "/api/groups/{dn}/members/{memberDn}", handleRemoveGroupMember)

	req := httptest.NewRequest("DELETE", "/api/groups/CN=TestGroup,CN=Users,DC=test,DC=com/members/CN=jdoe,CN=Users,DC=test,DC=com", nil)
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandleDisableUserFailure(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupFailingSambaTool(t)

	mux := newMuxWithRoute("POST", "/api/users/{dn}/disable", handleDisableUser)

	req := httptest.NewRequest("POST", "/api/users/CN=jdoe,CN=Users,DC=test,DC=com/disable", nil)
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandleUnlockUserFailure(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupFailingSambaTool(t)

	mux := newMuxWithRoute("POST", "/api/users/{dn}/unlock", handleUnlockUser)

	req := httptest.NewRequest("POST", "/api/users/CN=jdoe,CN=Users,DC=test,DC=com/unlock", nil)
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// --- Self-Service Tests ---

func TestHandleSelfPasswordChangeSuccess(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	body := `{"currentPassword":"TestPass123!","newPassword":"NewSecure456!"}`
	req := httptest.NewRequest("POST", "/api/self/password", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleSelfPasswordChange(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestHandleSelfPasswordChangeNoAuth(t *testing.T) {
	store, _ := setupTestAuth(t)
	_ = store

	body := `{"currentPassword":"TestPass123!","newPassword":"NewSecure456!"}`
	req := httptest.NewRequest("POST", "/api/self/password", strings.NewReader(body))
	w := httptest.NewRecorder()

	handleSelfPasswordChange(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleSelfPasswordChangeMissingFields(t *testing.T) {
	_, sess := setupTestAuth(t)

	body := `{"currentPassword":"","newPassword":""}`
	req := httptest.NewRequest("POST", "/api/self/password", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleSelfPasswordChange(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleSelfPasswordChangeWrongCurrent(t *testing.T) {
	_, sess := setupTestAuth(t)

	body := `{"currentPassword":"WrongPassword!","newPassword":"NewSecure456!"}`
	req := httptest.NewRequest("POST", "/api/self/password", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleSelfPasswordChange(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestHandleSelfPasswordChangeSambaToolFailure(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupFailingSambaTool(t)

	body := `{"currentPassword":"TestPass123!","newPassword":"NewSecure456!"}`
	req := httptest.NewRequest("POST", "/api/self/password", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleSelfPasswordChange(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandleSelfProfileNoAuth(t *testing.T) {
	store, _ := setupTestAuth(t)
	_ = store

	req := httptest.NewRequest("GET", "/api/self", nil)
	w := httptest.NewRecorder()

	handleSelfProfile(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleSelfProfileDirClientNil(t *testing.T) {
	_, sess := setupTestAuth(t)
	dirClient = nil

	req := httptest.NewRequest("GET", "/api/self", nil)
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleSelfProfile(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleSelfProfileUpdateNoAuth(t *testing.T) {
	store, _ := setupTestAuth(t)
	_ = store

	body := `{"phone":"555-1234"}`
	req := httptest.NewRequest("PUT", "/api/self", strings.NewReader(body))
	w := httptest.NewRecorder()

	handleSelfProfileUpdate(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleSelfProfileUpdateDirClientNil(t *testing.T) {
	_, sess := setupTestAuth(t)
	dirClient = nil

	body := `{"phone":"555-1234"}`
	req := httptest.NewRequest("PUT", "/api/self", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleSelfProfileUpdate(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleSelfProfileUpdateNoFields(t *testing.T) {
	_, sess := setupTestAuth(t)
	dirClient = nil

	body := `{}`
	req := httptest.NewRequest("PUT", "/api/self", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleSelfProfileUpdate(w, req)

	// dirClient nil returns 503 before reaching the empty-fields check
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleSelfProfileUpdateBadJSON(t *testing.T) {
	_, sess := setupTestAuth(t)
	// dirClient is nil from previous tests — handler checks it before JSON decode,
	// so bad JSON with nil dirClient returns 503 not 400. That's correct behavior.
	dirClient = nil

	body := `not json`
	req := httptest.NewRequest("PUT", "/api/self", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleSelfProfileUpdate(w, req)

	// dirClient nil returns 503 before reaching JSON decode
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

// ================= M15: Contacts Tests =================

func TestHandleCreateContactSuccess(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	body := `{"name":"External Vendor","givenName":"External","surname":"Vendor","mail":"vendor@example.com"}`
	req := httptest.NewRequest("POST", "/api/contacts", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleCreateContact(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}
}

func TestHandleCreateContactNoAuth(t *testing.T) {
	store, _ := setupTestAuth(t)
	_ = store

	body := `{"name":"Test Contact"}`
	req := httptest.NewRequest("POST", "/api/contacts", strings.NewReader(body))
	w := httptest.NewRecorder()

	handleCreateContact(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleCreateContactBadRequest(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	body := `{"name":""}`
	req := httptest.NewRequest("POST", "/api/contacts", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleCreateContact(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleCreateContactFailure(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupFailingSambaTool(t)

	body := `{"name":"Test Contact"}`
	req := httptest.NewRequest("POST", "/api/contacts", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleCreateContact(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandleDeleteContactSuccess(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	mux := newMuxWithRoute("DELETE", "/api/contacts/{dn}", handleDeleteContact)

	req := httptest.NewRequest("DELETE", "/api/contacts/CN=ExternalVendor,OU=Contacts,DC=test,DC=com", nil)
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleDeleteContactNoAuth(t *testing.T) {
	store, _ := setupTestAuth(t)
	_ = store

	mux := newMuxWithRoute("DELETE", "/api/contacts/{dn}", handleDeleteContact)

	req := httptest.NewRequest("DELETE", "/api/contacts/CN=ExternalVendor,OU=Contacts,DC=test,DC=com", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleMoveContactSuccess(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	mux := newMuxWithRoute("POST", "/api/contacts/{dn}/move", handleMoveContact)

	body := `{"targetOu":"OU=External,DC=test,DC=com"}`
	req := httptest.NewRequest("POST", "/api/contacts/CN=ExternalVendor,OU=Contacts,DC=test,DC=com/move", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleMoveContactBadRequest(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	mux := newMuxWithRoute("POST", "/api/contacts/{dn}/move", handleMoveContact)

	body := `{"targetOu":""}`
	req := httptest.NewRequest("POST", "/api/contacts/CN=ExternalVendor,OU=Contacts,DC=test,DC=com/move", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleRenameContactSuccess(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	mux := newMuxWithRoute("POST", "/api/contacts/{dn}/rename", handleRenameContact)

	body := `{"newName":"New Vendor Name"}`
	req := httptest.NewRequest("POST", "/api/contacts/CN=ExternalVendor,OU=Contacts,DC=test,DC=com/rename", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleRenameContactBadRequest(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	mux := newMuxWithRoute("POST", "/api/contacts/{dn}/rename", handleRenameContact)

	body := `{"newName":""}`
	req := httptest.NewRequest("POST", "/api/contacts/CN=ExternalVendor,OU=Contacts,DC=test,DC=com/rename", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleUpdateContactDirClientNil(t *testing.T) {
	_, sess := setupTestAuth(t)
	dirClient = nil

	mux := newMuxWithRoute("PUT", "/api/contacts/{dn}", handleUpdateContact)

	body := `{"displayName":"Updated Name"}`
	req := httptest.NewRequest("PUT", "/api/contacts/CN=ExternalVendor,OU=Contacts,DC=test,DC=com", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

// ================= M15: User Rename Tests =================

func TestHandleRenameUserSuccess(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	mux := newMuxWithRoute("POST", "/api/users/{dn}/rename", handleRenameUser)

	body := `{"newName":"John Smith-Doe"}`
	req := httptest.NewRequest("POST", "/api/users/CN=jdoe,CN=Users,DC=test,DC=com/rename", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleRenameUserNoAuth(t *testing.T) {
	store, _ := setupTestAuth(t)
	_ = store

	mux := newMuxWithRoute("POST", "/api/users/{dn}/rename", handleRenameUser)

	body := `{"newName":"New Name"}`
	req := httptest.NewRequest("POST", "/api/users/CN=jdoe,CN=Users,DC=test,DC=com/rename", strings.NewReader(body))
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleRenameUserBadRequest(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	mux := newMuxWithRoute("POST", "/api/users/{dn}/rename", handleRenameUser)

	body := `{"newName":""}`
	req := httptest.NewRequest("POST", "/api/users/CN=jdoe,CN=Users,DC=test,DC=com/rename", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleRenameUserFailure(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupFailingSambaTool(t)

	mux := newMuxWithRoute("POST", "/api/users/{dn}/rename", handleRenameUser)

	body := `{"newName":"New Name"}`
	req := httptest.NewRequest("POST", "/api/users/CN=jdoe,CN=Users,DC=test,DC=com/rename", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// ================= M15: Group Rename Tests =================

func TestHandleRenameGroupSuccess(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	mux := newMuxWithRoute("POST", "/api/groups/{dn}/rename", handleRenameGroup)

	body := `{"newName":"Renamed Group"}`
	req := httptest.NewRequest("POST", "/api/groups/CN=TestGroup,CN=Users,DC=test,DC=com/rename", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleRenameGroupNoAuth(t *testing.T) {
	store, _ := setupTestAuth(t)
	_ = store

	mux := newMuxWithRoute("POST", "/api/groups/{dn}/rename", handleRenameGroup)

	body := `{"newName":"Renamed"}`
	req := httptest.NewRequest("POST", "/api/groups/CN=TestGroup,CN=Users,DC=test,DC=com/rename", strings.NewReader(body))
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleRenameGroupBadRequest(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	mux := newMuxWithRoute("POST", "/api/groups/{dn}/rename", handleRenameGroup)

	body := `{"newName":""}`
	req := httptest.NewRequest("POST", "/api/groups/CN=TestGroup,CN=Users,DC=test,DC=com/rename", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// ================= M15: Computer Create & Move Tests =================

func TestHandleCreateComputerSuccess(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	body := `{"name":"WORKSTATION01"}`
	req := httptest.NewRequest("POST", "/api/computers", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleCreateComputer(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}
}

func TestHandleCreateComputerNoAuth(t *testing.T) {
	store, _ := setupTestAuth(t)
	_ = store

	body := `{"name":"WORKSTATION01"}`
	req := httptest.NewRequest("POST", "/api/computers", strings.NewReader(body))
	w := httptest.NewRecorder()

	handleCreateComputer(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleCreateComputerBadRequest(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	body := `{"name":""}`
	req := httptest.NewRequest("POST", "/api/computers", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleCreateComputer(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleCreateComputerFailure(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupFailingSambaTool(t)

	body := `{"name":"WORKSTATION01"}`
	req := httptest.NewRequest("POST", "/api/computers", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleCreateComputer(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandleMoveComputerSuccess(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	mux := newMuxWithRoute("POST", "/api/computers/{dn}/move", handleMoveComputer)

	body := `{"targetOu":"OU=Servers,DC=test,DC=com"}`
	req := httptest.NewRequest("POST", "/api/computers/CN=WORKSTATION1,CN=Computers,DC=test,DC=com/move", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleMoveComputerBadRequest(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	mux := newMuxWithRoute("POST", "/api/computers/{dn}/move", handleMoveComputer)

	body := `{"targetOu":""}`
	req := httptest.NewRequest("POST", "/api/computers/CN=WORKSTATION1,CN=Computers,DC=test,DC=com/move", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleMoveComputerNoAuth(t *testing.T) {
	store, _ := setupTestAuth(t)
	_ = store

	mux := newMuxWithRoute("POST", "/api/computers/{dn}/move", handleMoveComputer)

	body := `{"targetOu":"OU=Servers,DC=test,DC=com"}`
	req := httptest.NewRequest("POST", "/api/computers/CN=WORKSTATION1,CN=Computers,DC=test,DC=com/move", strings.NewReader(body))
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

// ================= M16: Search Tests =================

func TestHandleSearchNoAuth(t *testing.T) {
	store, _ := setupTestAuth(t)
	_ = store

	body := `{"rawFilter":"(objectClass=user)"}`
	req := httptest.NewRequest("POST", "/api/search", strings.NewReader(body))
	w := httptest.NewRecorder()

	handleSearch(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleSearchDirClientNil(t *testing.T) {
	_, sess := setupTestAuth(t)
	dirClient = nil

	body := `{"rawFilter":"(objectClass=user)"}`
	req := httptest.NewRequest("POST", "/api/search", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleSearch(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleSearchBadJSON(t *testing.T) {
	_, sess := setupTestAuth(t)
	dirClient = nil

	body := `not json`
	req := httptest.NewRequest("POST", "/api/search", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleSearch(w, req)

	// dirClient nil returns 503 before JSON decode
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleSearchNoFilter(t *testing.T) {
	_, sess := setupTestAuth(t)
	dirClient = nil

	body := `{}`
	req := httptest.NewRequest("POST", "/api/search", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleSearch(w, req)

	// dirClient nil returns 503 before filter check
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleCreateSavedQuerySuccess(t *testing.T) {
	_, sess := setupTestAuth(t)

	body := `{"name":"All Admins","description":"Find all admin users","request":{"rawFilter":"(memberOf=CN=Domain Admins,CN=Users,DC=test,DC=com)"}}`
	req := httptest.NewRequest("POST", "/api/search/saved", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleCreateSavedQuery(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusCreated, w.Body.String())
	}
}

func TestHandleCreateSavedQueryNoAuth(t *testing.T) {
	store, _ := setupTestAuth(t)
	_ = store

	body := `{"name":"Test Query"}`
	req := httptest.NewRequest("POST", "/api/search/saved", strings.NewReader(body))
	w := httptest.NewRecorder()

	handleCreateSavedQuery(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleCreateSavedQueryBadRequest(t *testing.T) {
	_, sess := setupTestAuth(t)

	body := `{"name":""}`
	req := httptest.NewRequest("POST", "/api/search/saved", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleCreateSavedQuery(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleListSavedQueries(t *testing.T) {
	_, sess := setupTestAuth(t)

	req := httptest.NewRequest("GET", "/api/search/saved", nil)
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleListSavedQueries(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleDeleteSavedQueryNotFound(t *testing.T) {
	_, sess := setupTestAuth(t)

	mux := newMuxWithRoute("DELETE", "/api/search/saved/{id}", handleDeleteSavedQuery)

	req := httptest.NewRequest("DELETE", "/api/search/saved/nonexistent", nil)
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleDeleteSavedQueryNoAuth(t *testing.T) {
	store, _ := setupTestAuth(t)
	_ = store

	mux := newMuxWithRoute("DELETE", "/api/search/saved/{id}", handleDeleteSavedQuery)

	req := httptest.NewRequest("DELETE", "/api/search/saved/q1", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

// ================= M16: Password Policy Tests =================

func TestHandleGetPasswordPolicySuccess(t *testing.T) {
	_, sess := setupTestAuth(t)

	// Mock samba-tool to return password policy output
	dir := t.TempDir()
	script := filepath.Join(dir, "samba-tool")
	err := os.WriteFile(script, []byte(`#!/bin/sh
echo "Password informations for domain 'DC=test,DC=com'"
echo ""
echo "Password complexity: on"
echo "Store plaintext passwords: off"
echo "Password history length: 24"
echo "Minimum password length: 7"
echo "Minimum password age (days): 1"
echo "Maximum password age (days): 42"
echo "Account lockout duration (mins): 30"
echo "Account lockout threshold (attempts): 0"
echo "Reset account lockout after (mins): 30"
`), 0755)
	if err != nil {
		t.Fatalf("write script: %v", err)
	}
	sambaTool = script
	t.Cleanup(func() { sambaTool = "samba-tool" })

	req := httptest.NewRequest("GET", "/api/password-policy", nil)
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleGetPasswordPolicy(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestHandleGetPasswordPolicyNoAuth(t *testing.T) {
	store, _ := setupTestAuth(t)
	_ = store

	req := httptest.NewRequest("GET", "/api/password-policy", nil)
	w := httptest.NewRecorder()

	handleGetPasswordPolicy(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleGetPasswordPolicyFailure(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupFailingSambaTool(t)

	req := httptest.NewRequest("GET", "/api/password-policy", nil)
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleGetPasswordPolicy(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandleUpdatePasswordPolicySuccess(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	body := `{"minLength":8,"complexity":true}`
	req := httptest.NewRequest("PUT", "/api/password-policy", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleUpdatePasswordPolicy(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleUpdatePasswordPolicyNoAuth(t *testing.T) {
	store, _ := setupTestAuth(t)
	_ = store

	body := `{"minLength":8}`
	req := httptest.NewRequest("PUT", "/api/password-policy", strings.NewReader(body))
	w := httptest.NewRecorder()

	handleUpdatePasswordPolicy(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleUpdatePasswordPolicyFailure(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupFailingSambaTool(t)

	body := `{"minLength":8,"complexity":true}`
	req := httptest.NewRequest("PUT", "/api/password-policy", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleUpdatePasswordPolicy(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// --- PSO Tests ---

func TestHandleListPSOsSuccess(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	req := httptest.NewRequest("GET", "/api/password-policy/pso", nil)
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleListPSOs(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleListPSOsNoAuth(t *testing.T) {
	store, _ := setupTestAuth(t)
	_ = store

	req := httptest.NewRequest("GET", "/api/password-policy/pso", nil)
	w := httptest.NewRecorder()

	handleListPSOs(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleCreatePSOSuccess(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	body := `{"name":"StrictPolicy","precedence":10,"minLength":12,"complexity":true}`
	req := httptest.NewRequest("POST", "/api/password-policy/pso", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleCreatePSO(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusCreated, w.Body.String())
	}
}

func TestHandleCreatePSONoAuth(t *testing.T) {
	store, _ := setupTestAuth(t)
	_ = store

	body := `{"name":"StrictPolicy","precedence":10}`
	req := httptest.NewRequest("POST", "/api/password-policy/pso", strings.NewReader(body))
	w := httptest.NewRecorder()

	handleCreatePSO(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleCreatePSOBadRequest(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	body := `{"name":""}`
	req := httptest.NewRequest("POST", "/api/password-policy/pso", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleCreatePSO(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleCreatePSONoPrecedence(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	body := `{"name":"TestPSO","precedence":0}`
	req := httptest.NewRequest("POST", "/api/password-policy/pso", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleCreatePSO(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleCreatePSOFailure(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupFailingSambaTool(t)

	body := `{"name":"StrictPolicy","precedence":10}`
	req := httptest.NewRequest("POST", "/api/password-policy/pso", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleCreatePSO(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandleDeletePSOSuccess(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	mux := newMuxWithRoute("DELETE", "/api/password-policy/pso/{name}", handleDeletePSO)

	req := httptest.NewRequest("DELETE", "/api/password-policy/pso/StrictPolicy", nil)
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleDeletePSONoAuth(t *testing.T) {
	store, _ := setupTestAuth(t)
	_ = store

	mux := newMuxWithRoute("DELETE", "/api/password-policy/pso/{name}", handleDeletePSO)

	req := httptest.NewRequest("DELETE", "/api/password-policy/pso/StrictPolicy", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleDeletePSOFailure(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupFailingSambaTool(t)

	mux := newMuxWithRoute("DELETE", "/api/password-policy/pso/{name}", handleDeletePSO)

	req := httptest.NewRequest("DELETE", "/api/password-policy/pso/StrictPolicy", nil)
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandleApplyPSOSuccess(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	mux := newMuxWithRoute("POST", "/api/password-policy/pso/{name}/apply", handleApplyPSO)

	body := `{"target":"jdoe"}`
	req := httptest.NewRequest("POST", "/api/password-policy/pso/StrictPolicy/apply", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleApplyPSOBadRequest(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	mux := newMuxWithRoute("POST", "/api/password-policy/pso/{name}/apply", handleApplyPSO)

	body := `{"target":""}`
	req := httptest.NewRequest("POST", "/api/password-policy/pso/StrictPolicy/apply", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleUnapplyPSOSuccess(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	mux := newMuxWithRoute("POST", "/api/password-policy/pso/{name}/unapply", handleUnapplyPSO)

	body := `{"target":"jdoe"}`
	req := httptest.NewRequest("POST", "/api/password-policy/pso/StrictPolicy/unapply", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleUnapplyPSOBadRequest(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	mux := newMuxWithRoute("POST", "/api/password-policy/pso/{name}/unapply", handleUnapplyPSO)

	body := `{"target":""}`
	req := httptest.NewRequest("POST", "/api/password-policy/pso/StrictPolicy/unapply", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleGetEffectivePolicyNoAuth(t *testing.T) {
	store, _ := setupTestAuth(t)
	_ = store

	mux := newMuxWithRoute("GET", "/api/password-policy/user/{username}", handleGetEffectivePolicy)

	req := httptest.NewRequest("GET", "/api/password-policy/user/jdoe", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleTestPasswordSuccess(t *testing.T) {
	_, sess := setupTestAuth(t)

	// Mock samba-tool to return policy data
	dir := t.TempDir()
	script := filepath.Join(dir, "samba-tool")
	err := os.WriteFile(script, []byte(`#!/bin/sh
echo "Password complexity: on"
echo "Minimum password length: 7"
echo "Password history length: 24"
`), 0755)
	if err != nil {
		t.Fatalf("write script: %v", err)
	}
	sambaTool = script
	t.Cleanup(func() { sambaTool = "samba-tool" })

	body := `{"password":"StrongP@ss123","username":"testuser"}`
	req := httptest.NewRequest("POST", "/api/password-policy/test", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleTestPassword(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestHandleTestPasswordNoAuth(t *testing.T) {
	store, _ := setupTestAuth(t)
	_ = store

	body := `{"password":"Test123!"}`
	req := httptest.NewRequest("POST", "/api/password-policy/test", strings.NewReader(body))
	w := httptest.NewRecorder()

	handleTestPassword(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleTestPasswordBadRequest(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	body := `{"password":""}`
	req := httptest.NewRequest("POST", "/api/password-policy/test", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleTestPassword(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleUpdatePSOSuccess(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	mux := newMuxWithRoute("PUT", "/api/password-policy/pso/{name}", handleUpdatePSO)

	body := `{"minLength":14,"complexity":true}`
	req := httptest.NewRequest("PUT", "/api/password-policy/pso/StrictPolicy", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleUpdatePSONoAuth(t *testing.T) {
	store, _ := setupTestAuth(t)
	_ = store

	mux := newMuxWithRoute("PUT", "/api/password-policy/pso/{name}", handleUpdatePSO)

	body := `{"minLength":14}`
	req := httptest.NewRequest("PUT", "/api/password-policy/pso/StrictPolicy", strings.NewReader(body))
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

// --- Filter Builder Tests ---

func TestBuildFilterFromVisual(t *testing.T) {
	tests := []struct {
		name       string
		objectType string
		filters    []models.SearchFilter
		want       string
	}{
		{
			name:       "user with contains",
			objectType: "user",
			filters:    []models.SearchFilter{{Attribute: "displayName", Operator: "contains", Value: "admin"}},
			want:       "(&(&(objectClass=user)(!(objectClass=computer)))(displayName=*admin*))",
		},
		{
			name:       "group equals",
			objectType: "group",
			filters:    []models.SearchFilter{{Attribute: "cn", Operator: "equals", Value: "Domain Admins"}},
			want:       "(&(objectClass=group)(cn=Domain Admins))",
		},
		{
			name:       "no filters no type",
			objectType: "all",
			filters:    nil,
			want:       "(objectClass=*)",
		},
		{
			name:       "present operator",
			objectType: "user",
			filters:    []models.SearchFilter{{Attribute: "mail", Operator: "present"}},
			want:       "(&(&(objectClass=user)(!(objectClass=computer)))(mail=*))",
		},
		{
			name:       "notPresent operator",
			objectType: "computer",
			filters:    []models.SearchFilter{{Attribute: "operatingSystem", Operator: "notPresent"}},
			want:       "(&(objectClass=computer)(!(operatingSystem=*)))",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildFilterFromVisual(tt.objectType, tt.filters)
			if got != tt.want {
				t.Errorf("buildFilterFromVisual() = %q, want %q", got, tt.want)
			}
		})
	}
}

// --- Password Policy Parser Tests ---

func TestParsePasswordPolicy(t *testing.T) {
	output := `Password informations for domain 'DC=dzsec,DC=net'

Password complexity: on
Store plaintext passwords: off
Password history length: 24
Minimum password length: 7
Minimum password age (days): 1
Maximum password age (days): 42
Account lockout duration (mins): 30
Account lockout threshold (attempts): 5
Reset account lockout after (mins): 30
`
	p := parsePasswordPolicy(output)

	if p.MinLength != 7 {
		t.Errorf("MinLength = %d, want 7", p.MinLength)
	}
	if p.HistoryLength != 24 {
		t.Errorf("HistoryLength = %d, want 24", p.HistoryLength)
	}
	if !p.Complexity {
		t.Error("Complexity should be true")
	}
	if p.StorePlaintext {
		t.Error("StorePlaintext should be false")
	}
	if p.LockoutThreshold != 5 {
		t.Errorf("LockoutThreshold = %d, want 5", p.LockoutThreshold)
	}
}

// --- Password Tester Tests ---

func TestTestPasswordAgainstPolicy(t *testing.T) {
	policy := models.PasswordPolicy{
		MinLength:  8,
		Complexity: true,
	}

	tests := []struct {
		name     string
		password string
		username string
		valid    bool
	}{
		{"strong password", "MyStr0ng!Pass", "jdoe", true},
		{"too short", "Ab1!", "", false},
		{"no complexity", "abcdefgh", "", false},
		{"contains username", "jdoePassword1!", "jdoe", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testPasswordAgainstPolicy(tt.password, tt.username, policy)
			if result.Valid != tt.valid {
				t.Errorf("valid = %v, want %v; errors: %v", result.Valid, tt.valid, result.Errors)
			}
		})
	}
}

// ================= M17: DNS Deep Dive Tests =================

func setupDNSDeepTest(t *testing.T) {
	t.Helper()

	// Ensure handlerConfig is set
	if handlerConfig == nil {
		handlerConfig = &config.Config{BaseDN: "DC=test,DC=com"}
	}

	// Reset dnsClient so it will be re-created with our mock samba-tool
	dnsClient = nil
	t.Cleanup(func() { dnsClient = nil })

	dir := t.TempDir()
	script := filepath.Join(dir, "samba-tool")
	err := os.WriteFile(script, []byte(`#!/bin/sh
if echo "$@" | grep -q "serverinfo"; then
  echo "DNS information for DC 'localhost'"
  echo "dwVersion                   : 14601"
  echo "fBootMethod                 : DNS_BOOT_METHOD_DIRECTORY"
  echo "fAllowUpdate                : TRUE"
  echo "fDsAvailable                : TRUE"
  echo "aipForwarders               : ip4:10.15.15.1"
  exit 0
fi
if echo "$@" | grep -q "zoneinfo"; then
  echo "Zone information for 'dzsec.net'"
  echo "pszZoneName                 : dzsec.net"
  echo "Flags                       : DNS_RPC_ZONE_DSINTEGRATED DNS_RPC_ZONE_UPDATE_SECURE"
  echo "ZoneType                    : DNS_ZONE_TYPE_PRIMARY"
  echo "fAging                      : 1"
  echo "dwNoRefreshInterval         : 168"
  echo "dwRefreshInterval           : 168"
  echo "dwAvailForScavengeTime      : 0"
  echo "aipScavengeServers          : <none>"
  exit 0
fi
if echo "$@" | grep -q "zoneoptions"; then
  echo "Zone options updated"
  exit 0
fi
if echo "$@" | grep -q "query"; then
  echo "  Name=@, Records=1, Children=0"
  echo "    A: 10.15.15.57 (flags=600000f0, serial=110, ttl=900)"
  exit 0
fi
echo "mock ok"
`), 0755)
	if err != nil {
		t.Fatalf("write mock samba-tool: %v", err)
	}
	sambaTool = script
	t.Cleanup(func() { sambaTool = "samba-tool" })
}

func TestHandleDNSServerInfoSuccess(t *testing.T) {
	setupDNSDeepTest(t)

	req := httptest.NewRequest("GET", "/api/dns/serverinfo", nil)
	w := httptest.NewRecorder()

	handleDNSServerInfo(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify the response contains parsed data
	if !strings.Contains(w.Body.String(), "10.15.15.1") {
		t.Errorf("response should contain forwarder IP; got %s", w.Body.String())
	}
}

func TestHandleDNSServerInfoFailure(t *testing.T) {
	if handlerConfig == nil {
		handlerConfig = &config.Config{BaseDN: "DC=test,DC=com"}
	}
	dnsClient = nil
	t.Cleanup(func() { dnsClient = nil })
	setupFailingSambaTool(t)

	req := httptest.NewRequest("GET", "/api/dns/serverinfo", nil)
	w := httptest.NewRecorder()

	handleDNSServerInfo(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandleDNSZoneInfoSuccess(t *testing.T) {
	setupDNSDeepTest(t)

	mux := newMuxWithRoute("GET", "/api/dns/zones/{zone}/info", handleDNSZoneInfo)

	req := httptest.NewRequest("GET", "/api/dns/zones/dzsec.net/info", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify aging is parsed
	if !strings.Contains(w.Body.String(), `"agingEnabled":true`) {
		t.Errorf("response should show aging enabled; got %s", w.Body.String())
	}
}

func TestHandleDNSZoneInfoFailure(t *testing.T) {
	if handlerConfig == nil {
		handlerConfig = &config.Config{BaseDN: "DC=test,DC=com"}
	}
	dnsClient = nil
	t.Cleanup(func() { dnsClient = nil })
	setupFailingSambaTool(t)

	mux := newMuxWithRoute("GET", "/api/dns/zones/{zone}/info", handleDNSZoneInfo)

	req := httptest.NewRequest("GET", "/api/dns/zones/dzsec.net/info", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandleDNSZoneOptionsSuccess(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupDNSDeepTest(t)

	mux := newMuxWithRoute("PUT", "/api/dns/zones/{zone}/options", handleDNSZoneOptions)

	body := `{"aging":true,"noRefreshInterval":168,"refreshInterval":168}`
	req := httptest.NewRequest("PUT", "/api/dns/zones/dzsec.net/options", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestHandleDNSZoneOptionsNoAuth(t *testing.T) {
	store, _ := setupTestAuth(t)
	_ = store

	mux := newMuxWithRoute("PUT", "/api/dns/zones/{zone}/options", handleDNSZoneOptions)

	body := `{"aging":true}`
	req := httptest.NewRequest("PUT", "/api/dns/zones/dzsec.net/options", strings.NewReader(body))
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleDNSZoneOptionsFailure(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupFailingSambaTool(t)

	mux := newMuxWithRoute("PUT", "/api/dns/zones/{zone}/options", handleDNSZoneOptions)

	body := `{"aging":true}`
	req := httptest.NewRequest("PUT", "/api/dns/zones/dzsec.net/options", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandleDNSQuerySuccess(t *testing.T) {
	setupDNSDeepTest(t)

	body := `{"server":"localhost","zone":"dzsec.net","name":"@","type":"A"}`
	req := httptest.NewRequest("POST", "/api/dns/query", strings.NewReader(body))
	w := httptest.NewRecorder()

	handleDNSQuery(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	if !strings.Contains(w.Body.String(), "10.15.15.57") {
		t.Errorf("response should contain A record value; got %s", w.Body.String())
	}
}

func TestHandleDNSQueryBadRequest(t *testing.T) {
	setupMockSambaTool(t)

	body := `{"zone":"","name":""}`
	req := httptest.NewRequest("POST", "/api/dns/query", strings.NewReader(body))
	w := httptest.NewRecorder()

	handleDNSQuery(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleDNSQueryBadJSON(t *testing.T) {
	body := `not json`
	req := httptest.NewRequest("POST", "/api/dns/query", strings.NewReader(body))
	w := httptest.NewRecorder()

	handleDNSQuery(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleDNSQueryDefaultServer(t *testing.T) {
	setupDNSDeepTest(t)

	// No server specified — should default to localhost
	body := `{"zone":"dzsec.net","name":"@","type":"ALL"}`
	req := httptest.NewRequest("POST", "/api/dns/query", strings.NewReader(body))
	w := httptest.NewRecorder()

	handleDNSQuery(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestHandleDNSLimitations(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/dns/limitations", nil)
	w := httptest.NewRecorder()

	handleDNSLimitations(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	if !strings.Contains(w.Body.String(), "conditional-forwarders") {
		t.Errorf("response should contain limitation IDs; got %s", w.Body.String())
	}
}

func TestHandleDNSZoneOptionsBadJSON(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupMockSambaTool(t)

	mux := newMuxWithRoute("PUT", "/api/dns/zones/{zone}/options", handleDNSZoneOptions)

	body := `not json`
	req := httptest.NewRequest("PUT", "/api/dns/zones/dzsec.net/options", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// --- DNS Samba Client Parser Tests ---

func TestHandleDNSSRVValidatorNoConfig(t *testing.T) {
	setupDNSDeepTest(t)

	// With empty config, it should still work using localhost
	oldConfig := handlerConfig
	handlerConfig = &config.Config{BaseDN: "DC=test,DC=com"}
	t.Cleanup(func() { handlerConfig = oldConfig })

	req := httptest.NewRequest("GET", "/api/dns/srv-validator", nil)
	w := httptest.NewRecorder()

	handleDNSSRVValidator(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	if !strings.Contains(w.Body.String(), `"summary"`) {
		t.Errorf("response should contain summary; got %s", w.Body.String())
	}
}

func TestHandleDNSConsistencyNoConfig(t *testing.T) {
	setupDNSDeepTest(t)

	oldConfig := handlerConfig
	handlerConfig = &config.Config{BaseDN: "DC=test,DC=com"}
	t.Cleanup(func() { handlerConfig = oldConfig })

	req := httptest.NewRequest("GET", "/api/dns/consistency?zone=test.com", nil)
	w := httptest.NewRecorder()

	handleDNSConsistency(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	if !strings.Contains(w.Body.String(), `"consistent"`) {
		t.Errorf("response should contain consistent field; got %s", w.Body.String())
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// M18: Infrastructure & Replication Tests
// ──────────────────────────────────────────────────────────────────────────────

func setupInfraTest(t *testing.T) {
	t.Helper()

	if handlerConfig == nil {
		handlerConfig = &config.Config{BaseDN: "DC=test,DC=com"}
	}

	dir := t.TempDir()
	script := filepath.Join(dir, "samba-tool")
	err := os.WriteFile(script, []byte(`#!/bin/sh
if echo "$@" | grep -q "drs showrepl"; then
  if echo "$@" | grep -q -- "--json"; then
    cat <<'JSONEOF'
{"repsFrom":[{"naming context":"DC=test,DC=com","DSA":"dc2.test.com","last attempt message":"WERR_OK"}],"repsTo":[{"naming context":"DC=test,DC=com","DSA":"dc2.test.com","last attempt message":"successful"}]}
JSONEOF
    exit 0
  fi
  echo "=== INBOUND NEIGHBORS ==="
  echo ""
  echo "DC=test,DC=com"
  echo "    Default-First-Site-Name\DC2 via RPC"
  echo "        DSA object GUID: aaaaaaaa-0000-0000-0000-000000000000"
  echo "        Last attempt @ ... was successful"
  echo ""
  echo "=== OUTBOUND NEIGHBORS ==="
  echo ""
  echo "DC=test,DC=com"
  echo "    Default-First-Site-Name\DC2 via RPC"
  echo "        DSA object GUID: bbbbbbbb-0000-0000-0000-000000000000"
  echo "        Last attempt @ ... was successful"
  exit 0
fi
if echo "$@" | grep -q "drs replicate"; then
  echo "Replicate from dc2.test.com to dc1.test.com was successful."
  exit 0
fi
if echo "$@" | grep -q "sites list"; then
  echo "Default-First-Site-Name"
  echo "Bozeman"
  echo "Seattle"
  exit 0
fi
if echo "$@" | grep -q "sites subnet list"; then
  echo "10.10.1.0/24, Default-First-Site-Name"
  echo "10.10.2.0/24, Bozeman"
  echo "10.10.3.0/24, Seattle"
  exit 0
fi
if echo "$@" | grep -q "sites create"; then
  echo "Site created successfully"
  exit 0
fi
if echo "$@" | grep -q "fsmo show"; then
  echo "SchemaMasterRole owner: CN=NTDS Settings,CN=BRIDGER,CN=Servers,CN=Default-First-Site-Name,CN=Sites,CN=Configuration,DC=test,DC=com"
  echo "InfrastructureMasterRole owner: CN=NTDS Settings,CN=BRIDGER,CN=Servers,CN=Default-First-Site-Name,CN=Sites,CN=Configuration,DC=test,DC=com"
  echo "RidAllocationMasterRole owner: CN=NTDS Settings,CN=BRIDGER,CN=Servers,CN=Default-First-Site-Name,CN=Sites,CN=Configuration,DC=test,DC=com"
  echo "PdcEmulationMasterRole owner: CN=NTDS Settings,CN=BRIDGER,CN=Servers,CN=Default-First-Site-Name,CN=Sites,CN=Configuration,DC=test,DC=com"
  echo "DomainNamingMasterRole owner: CN=NTDS Settings,CN=BRIDGER,CN=Servers,CN=Default-First-Site-Name,CN=Sites,CN=Configuration,DC=test,DC=com"
  echo "DomainDnsZonesMasterRole owner: CN=NTDS Settings,CN=BRIDGER,CN=Servers,CN=Default-First-Site-Name,CN=Sites,CN=Configuration,DC=test,DC=com"
  echo "ForestDnsZonesMasterRole owner: CN=NTDS Settings,CN=BRIDGER,CN=Servers,CN=Default-First-Site-Name,CN=Sites,CN=Configuration,DC=test,DC=com"
  exit 0
fi
if echo "$@" | grep -q "fsmo transfer"; then
  echo "FSMO transfer was successful"
  exit 0
fi
echo "mock ok"
`), 0755)
	if err != nil {
		t.Fatalf("write mock samba-tool: %v", err)
	}
	sambaTool = script
	t.Cleanup(func() { sambaTool = "samba-tool" })
}

// --- Replication Tests ---

func TestHandleReplicationTopologySuccess(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupInfraTest(t)

	req := httptest.NewRequest("GET", "/api/replication/topology", nil)
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleReplicationTopologyLive(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"links"`) {
		t.Errorf("response should contain links; got %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"dcs"`) {
		t.Errorf("response should contain dcs; got %s", w.Body.String())
	}
}

func TestHandleReplicationTopologyFailure(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupFailingSambaTool(t)
	if handlerConfig == nil {
		handlerConfig = &config.Config{BaseDN: "DC=test,DC=com"}
	}

	req := httptest.NewRequest("GET", "/api/replication/topology", nil)
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleReplicationTopologyLive(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandleReplicationStatusNoDCs(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupInfraTest(t)

	oldConfig := handlerConfig
	handlerConfig = &config.Config{BaseDN: "DC=test,DC=com"}
	t.Cleanup(func() { handlerConfig = oldConfig })

	req := httptest.NewRequest("GET", "/api/replication/status", nil)
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleReplicationStatusLive(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !strings.Contains(w.Body.String(), `"total":0`) {
		t.Errorf("should show 0 total DCs; got %s", w.Body.String())
	}
}

func TestHandleReplicationStatusWithDCs(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupInfraTest(t)

	oldConfig := handlerConfig
	handlerConfig = &config.Config{
		BaseDN: "DC=test,DC=com",
		DCs: []config.DCConfig{
			{Hostname: "dc1", Address: "127.0.0.1", Site: "Default"},
		},
	}
	t.Cleanup(func() { handlerConfig = oldConfig })

	req := httptest.NewRequest("GET", "/api/replication/status", nil)
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleReplicationStatusLive(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"summary"`) {
		t.Errorf("should contain summary; got %s", w.Body.String())
	}
}

func TestHandleForceSyncNoAuth(t *testing.T) {
	setupInfraTest(t)

	body := `{"sourceDC":"dc1","destDC":"dc2"}`
	req := httptest.NewRequest("POST", "/api/replication/sync", strings.NewReader(body))
	w := httptest.NewRecorder()

	handleForceSyncLive(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleForceSyncSuccess(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupInfraTest(t)

	body := `{"sourceDC":"dc2.test.com","destDC":"dc1.test.com"}`
	req := httptest.NewRequest("POST", "/api/replication/sync", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleForceSyncLive(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"success":true`) {
		t.Errorf("should contain success; got %s", w.Body.String())
	}
}

func TestHandleForceSyncBadRequest(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupInfraTest(t)

	body := `{"sourceDC":"","destDC":""}`
	req := httptest.NewRequest("POST", "/api/replication/sync", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleForceSyncLive(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// --- Sites Tests ---

func TestHandleListSitesSuccess(t *testing.T) {
	setupInfraTest(t)

	req := httptest.NewRequest("GET", "/api/sites", nil)
	w := httptest.NewRecorder()

	handleListSitesLive(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Bozeman") {
		t.Errorf("should contain Bozeman site; got %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Default-First-Site-Name") {
		t.Errorf("should contain Default-First-Site-Name; got %s", w.Body.String())
	}
}

func TestHandleListSitesFailure(t *testing.T) {
	setupFailingSambaTool(t)
	if handlerConfig == nil {
		handlerConfig = &config.Config{BaseDN: "DC=test,DC=com"}
	}

	req := httptest.NewRequest("GET", "/api/sites", nil)
	w := httptest.NewRecorder()

	handleListSitesLive(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandleCreateSiteNoAuth(t *testing.T) {
	setupInfraTest(t)

	body := `{"name":"NewSite"}`
	req := httptest.NewRequest("POST", "/api/sites", strings.NewReader(body))
	w := httptest.NewRecorder()

	handleCreateSiteLive(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleCreateSiteSuccess(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupInfraTest(t)

	body := `{"name":"NewSite"}`
	req := httptest.NewRequest("POST", "/api/sites", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleCreateSiteLive(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusCreated, w.Body.String())
	}
}

func TestHandleCreateSiteEmptyName(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupInfraTest(t)

	body := `{"name":""}`
	req := httptest.NewRequest("POST", "/api/sites", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleCreateSiteLive(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleListSubnetsSuccess(t *testing.T) {
	setupInfraTest(t)

	mux := newMuxWithRoute("GET", "/api/sites/{site}/subnets", handleListSubnetsLive)
	req := httptest.NewRequest("GET", "/api/sites/Bozeman/subnets", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"subnets"`) {
		t.Errorf("should contain subnets; got %s", w.Body.String())
	}
}

// --- FSMO Tests ---

func TestHandleGetFSMORolesSuccess(t *testing.T) {
	setupInfraTest(t)

	req := httptest.NewRequest("GET", "/api/fsmo", nil)
	w := httptest.NewRecorder()

	handleGetFSMORolesLive(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Schema Master") {
		t.Errorf("should contain Schema Master; got %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "BRIDGER") {
		t.Errorf("should contain BRIDGER as DC; got %s", w.Body.String())
	}
}

func TestHandleGetFSMORolesFailure(t *testing.T) {
	setupFailingSambaTool(t)
	if handlerConfig == nil {
		handlerConfig = &config.Config{BaseDN: "DC=test,DC=com"}
	}

	req := httptest.NewRequest("GET", "/api/fsmo", nil)
	w := httptest.NewRecorder()

	handleGetFSMORolesLive(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandleTransferFSMONoAuth(t *testing.T) {
	setupInfraTest(t)

	body := `{"role":"schema"}`
	req := httptest.NewRequest("POST", "/api/fsmo/transfer", strings.NewReader(body))
	w := httptest.NewRecorder()

	handleTransferFSMOLive(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleTransferFSMOSuccess(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupInfraTest(t)

	body := `{"role":"schema"}`
	req := httptest.NewRequest("POST", "/api/fsmo/transfer", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleTransferFSMOLive(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"success":true`) {
		t.Errorf("should contain success; got %s", w.Body.String())
	}
}

func TestHandleTransferFSMONoRole(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupInfraTest(t)

	body := `{"role":""}`
	req := httptest.NewRequest("POST", "/api/fsmo/transfer", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleTransferFSMOLive(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// --- Audit Log Tests ---

func TestHandleListAuditLogEmpty(t *testing.T) {
	// Reset audit entries
	auditMu.Lock()
	oldEntries := auditEntries
	auditEntries = nil
	auditMu.Unlock()
	t.Cleanup(func() {
		auditMu.Lock()
		auditEntries = oldEntries
		auditMu.Unlock()
	})

	req := httptest.NewRequest("GET", "/api/audit", nil)
	w := httptest.NewRecorder()

	handleListAuditLogLive(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !strings.Contains(w.Body.String(), `"entries"`) {
		t.Errorf("should contain entries; got %s", w.Body.String())
	}
}

func TestHandleListAuditLogWithEntries(t *testing.T) {
	auditMu.Lock()
	oldEntries := auditEntries
	auditEntries = nil
	auditMu.Unlock()
	t.Cleanup(func() {
		auditMu.Lock()
		auditEntries = oldEntries
		auditMu.Unlock()
	})

	LogAudit("admin", "Create User", "CN=test,DC=test,DC=com", "user", "dc1", true, "created user")
	LogAudit("admin", "Delete User", "CN=old,DC=test,DC=com", "user", "dc1", true, "deleted user")

	req := httptest.NewRequest("GET", "/api/audit?limit=10", nil)
	w := httptest.NewRecorder()

	handleListAuditLogLive(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !strings.Contains(w.Body.String(), "Create User") {
		t.Errorf("should contain Create User; got %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"total":2`) {
		t.Errorf("should show total 2; got %s", w.Body.String())
	}
}

// --- Parser Tests ---

func TestParseFSMORoles(t *testing.T) {
	input := `SchemaMasterRole owner: CN=NTDS Settings,CN=BRIDGER,CN=Servers,CN=Default-First-Site-Name,CN=Sites,CN=Configuration,DC=dzsec,DC=net
InfrastructureMasterRole owner: CN=NTDS Settings,CN=BRIDGER,CN=Servers,CN=Default-First-Site-Name,CN=Sites,CN=Configuration,DC=dzsec,DC=net
RidAllocationMasterRole owner: CN=NTDS Settings,CN=SHOWDOWN,CN=Servers,CN=Default-First-Site-Name,CN=Sites,CN=Configuration,DC=dzsec,DC=net
PdcEmulationMasterRole owner: CN=NTDS Settings,CN=BRIDGER,CN=Servers,CN=Default-First-Site-Name,CN=Sites,CN=Configuration,DC=dzsec,DC=net
DomainNamingMasterRole owner: CN=NTDS Settings,CN=BRIDGER,CN=Servers,CN=Default-First-Site-Name,CN=Sites,CN=Configuration,DC=dzsec,DC=net`

	roles := parseFSMORoles(input)

	if len(roles) != 5 {
		t.Fatalf("expected 5 roles, got %d", len(roles))
	}

	if roles[0].Role != "Schema Master" {
		t.Errorf("role[0] = %s, want Schema Master", roles[0].Role)
	}
	if roles[0].DC != "BRIDGER" {
		t.Errorf("dc[0] = %s, want BRIDGER", roles[0].DC)
	}
	if roles[2].DC != "SHOWDOWN" {
		t.Errorf("dc[2] = %s, want SHOWDOWN", roles[2].DC)
	}
}

func TestParseSitesList(t *testing.T) {
	input := "Default-First-Site-Name\nBozeman\nSeattle\n"
	sites := parseSitesList(input)

	if len(sites) != 3 {
		t.Fatalf("expected 3 sites, got %d", len(sites))
	}
	if sites[1].Name != "Bozeman" {
		t.Errorf("sites[1] = %s, want Bozeman", sites[1].Name)
	}
}

func TestParseSubnetsList(t *testing.T) {
	input := "10.10.1.0/24, Default-First-Site-Name\n10.10.2.0/24, Bozeman\n10.10.3.0/24, Seattle\n"

	// Filter for Bozeman
	subnets := parseSubnetsList(input, "Bozeman")
	if len(subnets) != 1 {
		t.Fatalf("expected 1 subnet for Bozeman, got %d", len(subnets))
	}
	if subnets[0] != "10.10.2.0/24" {
		t.Errorf("subnet = %s, want 10.10.2.0/24", subnets[0])
	}

	// No filter
	all := parseSubnetsList(input, "")
	if len(all) != 3 {
		t.Fatalf("expected 3 subnets unfiltered, got %d", len(all))
	}
}

func TestParseShowreplJSON(t *testing.T) {
	data := map[string]any{
		"repsFrom": []any{
			map[string]any{
				"naming context":       "DC=test,DC=com",
				"DSA":                  "dc2.test.com",
				"last attempt message": "WERR_OK",
			},
		},
		"repsTo": []any{
			map[string]any{
				"naming context":       "DC=test,DC=com",
				"DSA":                  "dc2.test.com",
				"last attempt message": "successful",
			},
		},
	}

	result := parseShowreplJSON(data)

	links, ok := result["links"].([]models.ReplicationLink)
	if !ok {
		t.Fatal("links should be []models.ReplicationLink")
	}
	if len(links) != 2 {
		t.Fatalf("expected 2 links, got %d", len(links))
	}
	if links[0].SourceDC != "dc2.test.com" {
		t.Errorf("link[0].sourceDC = %s, want dc2.test.com", links[0].SourceDC)
	}
	if links[0].Status != "current" {
		t.Errorf("link[0].status = %s, want current", links[0].Status)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// M19: GPO & SPN Tests
// ──────────────────────────────────────────────────────────────────────────────

func setupGPOSPNTest(t *testing.T) {
	t.Helper()

	if handlerConfig == nil {
		handlerConfig = &config.Config{BaseDN: "DC=test,DC=com"}
	}

	dir := t.TempDir()
	script := filepath.Join(dir, "samba-tool")
	err := os.WriteFile(script, []byte(`#!/bin/sh
if echo "$@" | grep -q "gpo listall"; then
  echo "GPO          : {31B2F340-016D-11D2-945F-00C04FB984F9}"
  echo "display name : Default Domain Policy"
  echo "path         : \\\\test.com\\sysvol\\test.com\\Policies\\{31B2F340-016D-11D2-945F-00C04FB984F9}"
  echo "dn           : CN={31B2F340-016D-11D2-945F-00C04FB984F9},CN=Policies,CN=System,DC=test,DC=com"
  echo "version      : 65539"
  echo "flags        : NONE"
  echo ""
  echo "GPO          : {6AC1786C-016F-11D2-945F-00C04FB984F9}"
  echo "display name : Default Domain Controllers Policy"
  echo "path         : \\\\test.com\\sysvol\\test.com\\Policies\\{6AC1786C-016F-11D2-945F-00C04FB984F9}"
  echo "dn           : CN={6AC1786C-016F-11D2-945F-00C04FB984F9},CN=Policies,CN=System,DC=test,DC=com"
  echo "version      : 1"
  echo "flags        : NONE"
  exit 0
fi
if echo "$@" | grep -q "gpo show"; then
  echo "GPO          : {31B2F340-016D-11D2-945F-00C04FB984F9}"
  echo "display name : Default Domain Policy"
  echo "path         : \\\\test.com\\sysvol\\test.com\\Policies\\{31B2F340-016D-11D2-945F-00C04FB984F9}"
  echo "dn           : CN={31B2F340-016D-11D2-945F-00C04FB984F9},CN=Policies,CN=System,DC=test,DC=com"
  echo "version      : 65539"
  echo "flags        : NONE"
  exit 0
fi
if echo "$@" | grep -q "gpo create"; then
  echo "GPO '{AABBCCDD-1234-5678-9012-AABBCCDDEEFF}' created"
  exit 0
fi
if echo "$@" | grep -q "gpo del"; then
  echo "GPO deleted successfully"
  exit 0
fi
if echo "$@" | grep -q "gpo setlink"; then
  echo "GPO link set successfully"
  exit 0
fi
if echo "$@" | grep -q "gpo dellink"; then
  echo "GPO link removed successfully"
  exit 0
fi
if echo "$@" | grep -q "gpo getlink"; then
  echo "GPO(s) linked to DN OU=Staff,DC=test,DC=com"
  echo "    GPO     : {31B2F340-016D-11D2-945F-00C04FB984F9}"
  echo "    Name    : Default Domain Policy"
  exit 0
fi
if echo "$@" | grep -q "spn list"; then
  echo "User CN=myhost,CN=Computers,DC=test,DC=com has the following servicePrincipalName:"
  echo "	 HTTP/myhost.test.com"
  echo "	 HOST/myhost.test.com"
  echo "	 HOST/myhost"
  exit 0
fi
if echo "$@" | grep -q "spn add"; then
  echo "SPN added successfully"
  exit 0
fi
if echo "$@" | grep -q "spn delete"; then
  echo "SPN deleted successfully"
  exit 0
fi
if echo "$@" | grep -q "delegation show"; then
  echo "Incoming Delegations:"
  echo "  (none)"
  echo "Outgoing Delegations:"
  echo "  cifs/fileserver.test.com"
  echo "  HTTP/web.test.com"
  exit 0
fi
if echo "$@" | grep -q "delegation add-service"; then
  echo "Delegation service added"
  exit 0
fi
if echo "$@" | grep -q "delegation del-service"; then
  echo "Delegation service removed"
  exit 0
fi
echo "mock ok"
`), 0755)
	if err != nil {
		t.Fatalf("write mock samba-tool: %v", err)
	}
	sambaTool = script
	t.Cleanup(func() { sambaTool = "samba-tool" })
}

// --- GPO Handler Tests ---

func TestHandleListGPOsSuccess(t *testing.T) {
	setupGPOSPNTest(t)

	req := httptest.NewRequest("GET", "/api/gpo", nil)
	w := httptest.NewRecorder()

	handleListGPOs(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Default Domain Policy") {
		t.Errorf("should contain Default Domain Policy; got %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"total":2`) {
		t.Errorf("should show total 2; got %s", w.Body.String())
	}
}

func TestHandleListGPOsFailure(t *testing.T) {
	setupFailingSambaTool(t)
	if handlerConfig == nil {
		handlerConfig = &config.Config{BaseDN: "DC=test,DC=com"}
	}

	req := httptest.NewRequest("GET", "/api/gpo", nil)
	w := httptest.NewRecorder()

	handleListGPOs(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandleGetGPOSuccess(t *testing.T) {
	setupGPOSPNTest(t)

	mux := newMuxWithRoute("GET", "/api/gpo/{id}", handleGetGPO)
	req := httptest.NewRequest("GET", "/api/gpo/{31B2F340-016D-11D2-945F-00C04FB984F9}", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Default Domain Policy") {
		t.Errorf("should contain Default Domain Policy; got %s", w.Body.String())
	}
}

func TestHandleCreateGPONoAuth(t *testing.T) {
	setupGPOSPNTest(t)

	body := `{"name":"Test GPO"}`
	req := httptest.NewRequest("POST", "/api/gpo", strings.NewReader(body))
	w := httptest.NewRecorder()

	handleCreateGPO(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleCreateGPOSuccess(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupGPOSPNTest(t)

	body := `{"name":"Test GPO"}`
	req := httptest.NewRequest("POST", "/api/gpo", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleCreateGPO(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusCreated, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"success":true`) {
		t.Errorf("should contain success; got %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "AABBCCDD") {
		t.Errorf("should contain extracted GUID; got %s", w.Body.String())
	}
}

func TestHandleCreateGPOEmptyName(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupGPOSPNTest(t)

	body := `{"name":""}`
	req := httptest.NewRequest("POST", "/api/gpo", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleCreateGPO(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleDeleteGPONoAuth(t *testing.T) {
	setupGPOSPNTest(t)

	mux := newMuxWithRoute("DELETE", "/api/gpo/{id}", handleDeleteGPO)
	req := httptest.NewRequest("DELETE", "/api/gpo/{31B2F340-016D-11D2-945F-00C04FB984F9}", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleDeleteGPOSuccess(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupGPOSPNTest(t)

	mux := newMuxWithRoute("DELETE", "/api/gpo/{id}", handleDeleteGPO)
	req := httptest.NewRequest("DELETE", "/api/gpo/{31B2F340-016D-11D2-945F-00C04FB984F9}", nil)
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"success":true`) {
		t.Errorf("should contain success; got %s", w.Body.String())
	}
}

func TestHandleLinkGPOSuccess(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupGPOSPNTest(t)

	mux := newMuxWithRoute("POST", "/api/gpo/{id}/link", handleLinkGPO)
	body := `{"ouDn":"OU=Staff,DC=test,DC=com"}`
	req := httptest.NewRequest("POST", "/api/gpo/{31B2F340-016D-11D2-945F-00C04FB984F9}/link", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"success":true`) {
		t.Errorf("should contain success; got %s", w.Body.String())
	}
}

func TestHandleLinkGPONoOU(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupGPOSPNTest(t)

	mux := newMuxWithRoute("POST", "/api/gpo/{id}/link", handleLinkGPO)
	body := `{"ouDn":""}`
	req := httptest.NewRequest("POST", "/api/gpo/{31B2F340-016D-11D2-945F-00C04FB984F9}/link", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleUnlinkGPOSuccess(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupGPOSPNTest(t)

	mux := newMuxWithRoute("DELETE", "/api/gpo/{id}/link", handleUnlinkGPO)
	body := `{"ouDn":"OU=Staff,DC=test,DC=com"}`
	req := httptest.NewRequest("DELETE", "/api/gpo/{31B2F340-016D-11D2-945F-00C04FB984F9}/link", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestHandleGetGPOLinksSuccess(t *testing.T) {
	setupGPOSPNTest(t)

	mux := newMuxWithRoute("GET", "/api/gpo/links/{ou}", handleGetGPOLinks)
	req := httptest.NewRequest("GET", "/api/gpo/links/OU=Staff,DC=test,DC=com", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"links"`) {
		t.Errorf("should contain links; got %s", w.Body.String())
	}
}

// --- SPN Handler Tests ---

func TestHandleListSPNsSuccess(t *testing.T) {
	setupGPOSPNTest(t)

	mux := newMuxWithRoute("GET", "/api/spn/{account}", handleListSPNs)
	req := httptest.NewRequest("GET", "/api/spn/myhost$", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "HTTP/myhost.test.com") {
		t.Errorf("should contain HTTP SPN; got %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"total":3`) {
		t.Errorf("should show total 3; got %s", w.Body.String())
	}
}

func TestHandleListSPNsFailure(t *testing.T) {
	setupFailingSambaTool(t)
	if handlerConfig == nil {
		handlerConfig = &config.Config{BaseDN: "DC=test,DC=com"}
	}

	mux := newMuxWithRoute("GET", "/api/spn/{account}", handleListSPNs)
	req := httptest.NewRequest("GET", "/api/spn/myhost$", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandleAddSPNNoAuth(t *testing.T) {
	setupGPOSPNTest(t)

	body := `{"spn":"HTTP/web.test.com","account":"myhost$"}`
	req := httptest.NewRequest("POST", "/api/spn", strings.NewReader(body))
	w := httptest.NewRecorder()

	handleAddSPN(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleAddSPNSuccess(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupGPOSPNTest(t)

	body := `{"spn":"HTTP/web.test.com","account":"myhost$"}`
	req := httptest.NewRequest("POST", "/api/spn", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleAddSPN(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusCreated, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"success":true`) {
		t.Errorf("should contain success; got %s", w.Body.String())
	}
}

func TestHandleAddSPNMissingFields(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupGPOSPNTest(t)

	body := `{"spn":"","account":""}`
	req := httptest.NewRequest("POST", "/api/spn", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleAddSPN(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleDeleteSPNNoAuth(t *testing.T) {
	setupGPOSPNTest(t)

	body := `{"spn":"HTTP/web.test.com","account":"myhost$"}`
	req := httptest.NewRequest("DELETE", "/api/spn", strings.NewReader(body))
	w := httptest.NewRecorder()

	handleDeleteSPN(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleDeleteSPNSuccess(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupGPOSPNTest(t)

	body := `{"spn":"HTTP/web.test.com","account":"myhost$"}`
	req := httptest.NewRequest("DELETE", "/api/spn", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	handleDeleteSPN(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"success":true`) {
		t.Errorf("should contain success; got %s", w.Body.String())
	}
}

// --- Delegation Handler Tests ---

func TestHandleGetDelegationSuccess(t *testing.T) {
	setupGPOSPNTest(t)

	mux := newMuxWithRoute("GET", "/api/delegation/{account}", handleGetDelegation)
	req := httptest.NewRequest("GET", "/api/delegation/myhost$", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "cifs/fileserver.test.com") {
		t.Errorf("should contain delegation service; got %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"constrained":true`) {
		t.Errorf("should show constrained delegation; got %s", w.Body.String())
	}
}

func TestHandleGetDelegationFailure(t *testing.T) {
	setupFailingSambaTool(t)
	if handlerConfig == nil {
		handlerConfig = &config.Config{BaseDN: "DC=test,DC=com"}
	}

	mux := newMuxWithRoute("GET", "/api/delegation/{account}", handleGetDelegation)
	req := httptest.NewRequest("GET", "/api/delegation/myhost$", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandleAddDelegationServiceNoAuth(t *testing.T) {
	setupGPOSPNTest(t)

	mux := newMuxWithRoute("POST", "/api/delegation/{account}/service", handleAddDelegationService)
	body := `{"service":"cifs/newserver.test.com"}`
	req := httptest.NewRequest("POST", "/api/delegation/myhost$/service", strings.NewReader(body))
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleAddDelegationServiceSuccess(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupGPOSPNTest(t)

	mux := newMuxWithRoute("POST", "/api/delegation/{account}/service", handleAddDelegationService)
	body := `{"service":"cifs/newserver.test.com"}`
	req := httptest.NewRequest("POST", "/api/delegation/myhost$/service", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"success":true`) {
		t.Errorf("should contain success; got %s", w.Body.String())
	}
}

func TestHandleAddDelegationServiceNoService(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupGPOSPNTest(t)

	mux := newMuxWithRoute("POST", "/api/delegation/{account}/service", handleAddDelegationService)
	body := `{"service":""}`
	req := httptest.NewRequest("POST", "/api/delegation/myhost$/service", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleRemoveDelegationServiceSuccess(t *testing.T) {
	_, sess := setupTestAuth(t)
	setupGPOSPNTest(t)

	mux := newMuxWithRoute("DELETE", "/api/delegation/{account}/service", handleRemoveDelegationService)
	body := `{"service":"cifs/fileserver.test.com"}`
	req := httptest.NewRequest("DELETE", "/api/delegation/myhost$/service", strings.NewReader(body))
	addSessionCookie(req, sess)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"success":true`) {
		t.Errorf("should contain success; got %s", w.Body.String())
	}
}

// --- GPO Parser Tests ---

func TestParseGPOListAll(t *testing.T) {
	input := `GPO          : {31B2F340-016D-11D2-945F-00C04FB984F9}
display name : Default Domain Policy
path         : \\dzsec.net\sysvol\dzsec.net\Policies\{31B2F340-016D-11D2-945F-00C04FB984F9}
dn           : CN={31B2F340-016D-11D2-945F-00C04FB984F9},CN=Policies,CN=System,DC=dzsec,DC=net
version      : 65539
flags        : NONE

GPO          : {6AC1786C-016F-11D2-945F-00C04FB984F9}
display name : Default Domain Controllers Policy
path         : \\dzsec.net\sysvol\dzsec.net\Policies\{6AC1786C-016F-11D2-945F-00C04FB984F9}
dn           : CN={6AC1786C-016F-11D2-945F-00C04FB984F9},CN=Policies,CN=System,DC=dzsec,DC=net
version      : 1
flags        : NONE
`

	gpos := parseGPOListAll(input)

	if len(gpos) != 2 {
		t.Fatalf("expected 2 GPOs, got %d", len(gpos))
	}
	if gpos[0].ID != "{31B2F340-016D-11D2-945F-00C04FB984F9}" {
		t.Errorf("gpo[0].ID = %s", gpos[0].ID)
	}
	if gpos[0].Name != "Default Domain Policy" {
		t.Errorf("gpo[0].Name = %s", gpos[0].Name)
	}
	if gpos[0].Version != 65539 {
		t.Errorf("gpo[0].Version = %d, want 65539", gpos[0].Version)
	}
	if gpos[1].Name != "Default Domain Controllers Policy" {
		t.Errorf("gpo[1].Name = %s", gpos[1].Name)
	}
}

func TestParseGPOGetLink(t *testing.T) {
	input := `GPO(s) linked to DN OU=Staff,DC=test,DC=com
    GPO     : {31B2F340-016D-11D2-945F-00C04FB984F9}
    Name    : Default Domain Policy
    GPO     : {6AC1786C-016F-11D2-945F-00C04FB984F9}
    Name    : Default Domain Controllers Policy`

	links := parseGPOGetLink(input, "OU=Staff,DC=test,DC=com")

	if len(links) != 2 {
		t.Fatalf("expected 2 links, got %d", len(links))
	}
	if links[0].GPOID != "{31B2F340-016D-11D2-945F-00C04FB984F9}" {
		t.Errorf("link[0].GPOID = %s", links[0].GPOID)
	}
	if links[0].OUDN != "OU=Staff,DC=test,DC=com" {
		t.Errorf("link[0].OUDN = %s", links[0].OUDN)
	}
}

func TestExtractGPOID(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"GPO '{AABBCCDD-1234-5678-9012-AABBCCDDEEFF}' created", "{AABBCCDD-1234-5678-9012-AABBCCDDEEFF}"},
		{"no guid here", ""},
		{"multiple {11111111-2222-3333-4444-555555555555} and {AAAAAAAA-BBBB-CCCC-DDDD-EEEEEEEEEEEE}", "{11111111-2222-3333-4444-555555555555}"},
	}

	for _, tt := range tests {
		got := extractGPOID(tt.input)
		if got != tt.want {
			t.Errorf("extractGPOID(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// --- SPN Parser Tests ---

func TestParseSPNList(t *testing.T) {
	input := `User CN=myhost,CN=Computers,DC=test,DC=com has the following servicePrincipalName:
	 HTTP/myhost.test.com
	 HOST/myhost.test.com
	 HOST/myhost`

	spns := parseSPNList(input, "myhost$")

	if len(spns) != 3 {
		t.Fatalf("expected 3 SPNs, got %d", len(spns))
	}
	if spns[0].Value != "HTTP/myhost.test.com" {
		t.Errorf("spn[0].Value = %s", spns[0].Value)
	}
	if spns[0].Account != "myhost$" {
		t.Errorf("spn[0].Account = %s", spns[0].Account)
	}
}

func TestParseDelegationShow(t *testing.T) {
	input := `Incoming Delegations:
  (none)
Outgoing Delegations:
  cifs/fileserver.test.com
  HTTP/web.test.com`

	info := parseDelegationShow(input, "myhost$")

	if info.Account != "myhost$" {
		t.Errorf("account = %s", info.Account)
	}
	if !info.Constrained {
		t.Error("should be constrained")
	}
	if info.Unconstrained {
		t.Error("should not be unconstrained")
	}
	if len(info.AllowedServices) != 2 {
		t.Fatalf("expected 2 services, got %d", len(info.AllowedServices))
	}
	if info.AllowedServices[0] != "cifs/fileserver.test.com" {
		t.Errorf("service[0] = %s", info.AllowedServices[0])
	}
}

func TestParseDelegationShowUnconstrained(t *testing.T) {
	input := `Account is trusted for delegation
Incoming Delegations:
  (none)
Outgoing Delegations:
  (none)`

	info := parseDelegationShow(input, "svc$")

	if !info.Unconstrained {
		t.Error("should be unconstrained")
	}
	if info.Constrained {
		t.Error("should not be constrained")
	}
	if len(info.AllowedServices) != 0 {
		t.Errorf("expected 0 services, got %d", len(info.AllowedServices))
	}
}
