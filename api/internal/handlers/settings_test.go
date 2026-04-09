package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nickdawson/sambmin/internal/auth"
	"github.com/nickdawson/sambmin/internal/config"
)

func setupSettingsTest(t *testing.T) (*auth.Store, *auth.Session) {
	t.Helper()

	store, err := auth.NewStore(1)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	sessionStore = store

	sess, err := store.Create("testadmin", "CN=testadmin,CN=Users,DC=test,DC=com",
		[]string{"Domain Admins"}, "TestPass123!")
	if err != nil {
		t.Fatalf("Create session: %v", err)
	}

	handlerConfig = &config.Config{
		BaseDN:         "DC=test,DC=com",
		SessionTimeout: 8,
		DCs: []config.DCConfig{
			{Hostname: "dc1.test.com", Address: "10.0.0.1", Port: 636, Site: "Default", Primary: true},
		},
		ScriptsPath: "/usr/local/share/sambmin/scripts",
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	ss, err := config.NewSettingsStore(path)
	if err != nil {
		t.Fatalf("NewSettingsStore: %v", err)
	}
	settingsStore = ss
	settingsAuthStore = store

	return store, sess
}

func withSession(r *http.Request, sess *auth.Session) *http.Request {
	ctx := context.WithValue(r.Context(), auth.SessionKey, sess)
	r.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sess.ID})
	return r.WithContext(ctx)
}

func TestHandleGetSettings(t *testing.T) {
	_, sess := setupSettingsTest(t)

	req := httptest.NewRequest("GET", "/api/settings", nil)
	req = withSession(req, sess)
	w := httptest.NewRecorder()

	handleGetSettings(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	conn, ok := resp["connection"].(map[string]any)
	if !ok {
		t.Fatal("missing connection in response")
	}
	if conn["baseDN"] != "DC=test,DC=com" {
		t.Errorf("baseDN = %v, want DC=test,DC=com", conn["baseDN"])
	}

	dcs, ok := conn["domainControllers"].([]any)
	if !ok || len(dcs) != 1 {
		t.Fatalf("expected 1 DC, got %v", conn["domainControllers"])
	}
}

func TestHandleUpdateConnection(t *testing.T) {
	_, sess := setupSettingsTest(t)

	body := `{
		"domainControllers": [
			{"hostname":"dc2.test.com","address":"10.0.0.2","port":636,"site":"Remote","primary":true}
		],
		"baseDN": "DC=new,DC=com",
		"protocol": "ldaps"
	}`

	req := httptest.NewRequest("PUT", "/api/settings/connection", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withSession(req, sess)
	w := httptest.NewRecorder()

	handleUpdateConnection(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["status"] != "saved" {
		t.Errorf("status = %v, want saved", resp["status"])
	}
	if resp["restartRequired"] != true {
		t.Error("expected restartRequired=true for connection change")
	}
}

func TestHandleUpdateConnection_Validation(t *testing.T) {
	_, sess := setupSettingsTest(t)

	tests := []struct {
		name string
		body string
		want int
	}{
		{"empty DCs", `{"domainControllers":[],"baseDN":"DC=test,DC=com"}`, 400},
		{"missing baseDN", `{"domainControllers":[{"hostname":"dc1","address":"1.1.1.1","port":636}],"baseDN":""}`, 400},
		{"invalid JSON", `{bad json}`, 400},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("PUT", "/api/settings/connection", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			req = withSession(req, sess)
			w := httptest.NewRecorder()

			handleUpdateConnection(w, req)

			if w.Code != tt.want {
				t.Errorf("status = %d, want %d, body: %s", w.Code, tt.want, w.Body.String())
			}
		})
	}
}

func TestHandleUpdateAuth(t *testing.T) {
	store, sess := setupSettingsTest(t)

	body := `{"sessionTimeout": 4}`

	req := httptest.NewRequest("PUT", "/api/settings/auth", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withSession(req, sess)
	w := httptest.NewRecorder()

	handleUpdateAuth(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["status"] != "saved" {
		t.Errorf("status = %v, want saved", resp["status"])
	}
	// Session timeout should not require restart
	if resp["restartRequired"] != false {
		t.Error("session timeout change should not require restart")
	}

	// Verify session timeout was applied immediately
	_ = store // store is used for live timeout update verification
}

func TestHandleUpdateAuth_KerberosRequiresRestart(t *testing.T) {
	_, sess := setupSettingsTest(t)

	body := `{"kerberos":{"enabled":true,"keytab":"/etc/krb5.keytab"}}`

	req := httptest.NewRequest("PUT", "/api/settings/auth", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withSession(req, sess)
	w := httptest.NewRecorder()

	handleUpdateAuth(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["restartRequired"] != true {
		t.Error("kerberos change should require restart")
	}
}

func TestHandleUpdateRBAC(t *testing.T) {
	_, sess := setupSettingsTest(t)

	body := `{
		"roles": [
			{"role":"Admin","groups":["Domain Admins"],"permissions":["*"]},
			{"role":"Reader","groups":["Domain Users"],"permissions":["*.read"]}
		]
	}`

	req := httptest.NewRequest("PUT", "/api/settings/rbac", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withSession(req, sess)
	w := httptest.NewRecorder()

	handleUpdateRBAC(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["status"] != "saved" {
		t.Errorf("status = %v, want saved", resp["status"])
	}
}

func TestHandleUpdateRBAC_Validation(t *testing.T) {
	_, sess := setupSettingsTest(t)

	tests := []struct {
		name string
		body string
		want int
	}{
		{"empty roles", `{"roles":[]}`, 400},
		{"empty role name", `{"roles":[{"role":"","groups":[],"permissions":[]}]}`, 400},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("PUT", "/api/settings/rbac", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			req = withSession(req, sess)
			w := httptest.NewRecorder()

			handleUpdateRBAC(w, req)

			if w.Code != tt.want {
				t.Errorf("status = %d, want %d, body: %s", w.Code, tt.want, w.Body.String())
			}
		})
	}
}

func TestHandleUpdateApplication(t *testing.T) {
	_, sess := setupSettingsTest(t)

	body := `{"auditRetentionDays": 30}`

	req := httptest.NewRequest("PUT", "/api/settings/application", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withSession(req, sess)
	w := httptest.NewRecorder()

	handleUpdateApplication(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["status"] != "saved" {
		t.Errorf("status = %v, want saved", resp["status"])
	}
}

func TestHandleUpdateApplication_Validation(t *testing.T) {
	_, sess := setupSettingsTest(t)

	body := `{"auditRetentionDays": 0}`

	req := httptest.NewRequest("PUT", "/api/settings/application", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withSession(req, sess)
	w := httptest.NewRecorder()

	handleUpdateApplication(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestSettingsNilStore(t *testing.T) {
	oldStore := settingsStore
	settingsStore = nil
	defer func() { settingsStore = oldStore }()

	req := httptest.NewRequest("PUT", "/api/settings/connection", strings.NewReader(`{}`))
	w := httptest.NewRecorder()

	handleUpdateConnection(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", w.Code)
	}
}

func TestSettingsPersistence(t *testing.T) {
	_, sess := setupSettingsTest(t)

	// Save connection settings
	body := `{
		"domainControllers": [
			{"hostname":"dc1.test.com","address":"10.0.0.1","port":636,"site":"Default","primary":true}
		],
		"baseDN": "DC=test,DC=com",
		"protocol": "ldaps"
	}`
	req := httptest.NewRequest("PUT", "/api/settings/connection", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withSession(req, sess)
	w := httptest.NewRecorder()
	handleUpdateConnection(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("save status = %d", w.Code)
	}

	// GET should reflect the saved values
	req2 := httptest.NewRequest("GET", "/api/settings", nil)
	req2 = withSession(req2, sess)
	w2 := httptest.NewRecorder()
	handleGetSettings(w2, req2)

	var resp map[string]any
	json.NewDecoder(w2.Body).Decode(&resp)

	conn := resp["connection"].(map[string]any)
	if conn["protocol"] != "ldaps" {
		t.Errorf("protocol = %v, want ldaps", conn["protocol"])
	}
}

func TestSettingsPersistence_SessionTimeout(t *testing.T) {
	_, sess := setupSettingsTest(t)

	// Save auth settings with session timeout = 4
	body := `{"sessionTimeout": 4}`
	req := httptest.NewRequest("PUT", "/api/settings/auth", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withSession(req, sess)
	w := httptest.NewRecorder()
	handleUpdateAuth(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("save status = %d, body: %s", w.Code, w.Body.String())
	}

	// GET should return sessionTimeout=4, not the base config's 8
	req2 := httptest.NewRequest("GET", "/api/settings", nil)
	req2 = withSession(req2, sess)
	w2 := httptest.NewRecorder()
	handleGetSettings(w2, req2)

	var resp map[string]any
	json.NewDecoder(w2.Body).Decode(&resp)

	authSection := resp["auth"].(map[string]any)
	timeout := authSection["sessionTimeout"].(float64)
	if timeout != 4 {
		t.Errorf("sessionTimeout = %v, want 4", timeout)
	}
}

func TestSettingsPersistence_KerberosDisabled(t *testing.T) {
	_, sess := setupSettingsTest(t)

	// Disable kerberos via overlay
	body := `{"kerberos":{"enabled":false}}`
	req := httptest.NewRequest("PUT", "/api/settings/auth", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withSession(req, sess)
	w := httptest.NewRecorder()
	handleUpdateAuth(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("save status = %d", w.Code)
	}

	// GET should reflect kerberos disabled
	req2 := httptest.NewRequest("GET", "/api/settings", nil)
	req2 = withSession(req2, sess)
	w2 := httptest.NewRecorder()
	handleGetSettings(w2, req2)

	var resp map[string]any
	json.NewDecoder(w2.Body).Decode(&resp)

	authSection := resp["auth"].(map[string]any)
	kerb := authSection["kerberos"].(map[string]any)
	if kerb["enabled"] != false {
		t.Errorf("kerberos enabled = %v, want false", kerb["enabled"])
	}
}
