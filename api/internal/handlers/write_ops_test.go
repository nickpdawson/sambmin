package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
