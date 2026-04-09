package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/nickdawson/sambmin/internal/auth"
	"github.com/nickdawson/sambmin/internal/config"
)

var (
	settingsStore     *config.SettingsStore
	settingsAuthStore *auth.Store
	appVersion        = "dev"
)

// InitSettings wires the settings store for the settings endpoints.
func InitSettings(store *config.SettingsStore, authStore *auth.Store) {
	settingsStore = store
	settingsAuthStore = authStore
}

// SetVersion sets the application version displayed in settings.
func SetVersion(v string) { appVersion = v }

// handleGetSettings returns the merged settings view.
// It applies the live overlay on top of the base config so that
// changes saved via PUT are immediately reflected without restart.
func handleGetSettings(w http.ResponseWriter, r *http.Request) {
	cfg := handlerConfig
	overlay := settingsStore.Overlay()

	// Merge overlay onto base config for the response
	merged := settingsStore.ApplyTo(cfg)

	// Build DC list from merged config
	dcs := make([]map[string]any, 0, len(merged.DCs))
	for _, dc := range merged.DCs {
		port := dc.Port
		if port == 0 {
			port = 636
		}
		dcs = append(dcs, map[string]any{
			"hostname": dc.Hostname,
			"address":  dc.Address,
			"port":     port,
			"site":     dc.Site,
			"primary":  dc.Primary,
			"status":   "configured",
		})
	}

	realm := merged.Kerberos.Realm
	if realm == "" {
		realm = domainFromBaseDN(merged.BaseDN)
	}

	protocol := "ldaps"
	if overlay.Connection != nil && overlay.Connection.Protocol != "" {
		protocol = overlay.Connection.Protocol
	}

	sessionTimeout := merged.SessionTimeout
	if sessionTimeout == 0 {
		sessionTimeout = 8
	}

	// Kerberos enabled: true if overlay explicitly set it, or if config has implementation
	kerberosEnabled := merged.Kerberos.Implementation != ""
	if overlay.Auth != nil && overlay.Auth.Kerberos != nil && overlay.Auth.Kerberos.Enabled != nil {
		kerberosEnabled = *overlay.Auth.Kerberos.Enabled
	}

	// Build RBAC roles from overlay if present, otherwise defaults
	roles := []map[string]any{
		{"role": "Full Admin", "groups": []string{"Domain Admins", "Enterprise Admins"}, "permissions": []string{"*"}},
		{"role": "User Admin", "groups": []string{"Account Operators"}, "permissions": []string{"users.*", "groups.read"}},
		{"role": "DNS Admin", "groups": []string{"DnsAdmins"}, "permissions": []string{"dns.*"}},
		{"role": "Read Only", "groups": []string{"Domain Users"}, "permissions": []string{"*.read"}},
	}
	if overlay.RBAC != nil && len(overlay.RBAC.Roles) > 0 {
		roles = make([]map[string]any, len(overlay.RBAC.Roles))
		for i, r := range overlay.RBAC.Roles {
			roles[i] = map[string]any{
				"role":        r.Role,
				"groups":      r.Groups,
				"permissions": r.Permissions,
			}
		}
	}

	auditRetentionDays := 90
	if overlay.Application != nil && overlay.Application.AuditRetentionDays != nil {
		auditRetentionDays = *overlay.Application.AuditRetentionDays
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"connection": map[string]any{
			"domainControllers": dcs,
			"baseDN":            merged.BaseDN,
			"realm":             realm,
			"protocol":          protocol,
		},
		"tls": map[string]any{
			"provider": "nginx-managed",
			"domain":   domainFromBaseDN(merged.BaseDN),
		},
		"auth": map[string]any{
			"kerberos": map[string]any{
				"enabled":        kerberosEnabled,
				"implementation": merged.Kerberos.Implementation,
				"keytab":         merged.Kerberos.KeytabPath,
			},
			"ldapBind": map[string]any{
				"enabled": merged.BindDN != "",
			},
			"sessionTimeout": sessionTimeout,
		},
		"rbac": map[string]any{
			"roles": roles,
		},
		"application": map[string]any{
			"version":            appVersion,
			"scriptsPath":        merged.ScriptsPath,
			"auditRetentionDays": auditRetentionDays,
		},
	})
}

// updateConnectionRequest is the expected body for PUT /api/settings/connection.
type updateConnectionRequest struct {
	DomainControllers []config.DCConfig `json:"domainControllers"`
	BaseDN            string            `json:"baseDN"`
	Protocol          string            `json:"protocol"`
}

func handleUpdateConnection(w http.ResponseWriter, r *http.Request) {
	if settingsStore == nil {
		respondError(w, http.StatusServiceUnavailable, "settings not available")
		return
	}

	var req updateConnectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.DomainControllers) == 0 {
		respondError(w, http.StatusBadRequest, "at least one domain controller is required")
		return
	}
	if req.BaseDN == "" {
		respondError(w, http.StatusBadRequest, "base_dn is required")
		return
	}

	overlay := config.ConnectionOverlay{
		DCs:      req.DomainControllers,
		BaseDN:   req.BaseDN,
		Protocol: req.Protocol,
	}

	restartFields := settingsStore.UpdateConnection(overlay)
	if err := settingsStore.Save(); err != nil {
		slog.Error("settings save failed", "section", "connection", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to save settings")
		return
	}

	sess, _ := r.Context().Value(auth.SessionKey).(*auth.Session)
	actor := "unknown"
	if sess != nil {
		actor = sess.Username
	}
	slog.Info("settings updated", "section", "connection", "actor", actor, "restartFields", restartFields)

	respondJSON(w, http.StatusOK, map[string]any{
		"status":         "saved",
		"restartRequired": len(restartFields) > 0,
		"restartFields":  restartFields,
	})
}

// updateAuthRequest is the expected body for PUT /api/settings/auth.
type updateAuthRequest struct {
	Kerberos       *kerberosRequest `json:"kerberos,omitempty"`
	SessionTimeout *int             `json:"sessionTimeout,omitempty"`
}

type kerberosRequest struct {
	Enabled    *bool  `json:"enabled,omitempty"`
	KeytabPath string `json:"keytab,omitempty"`
}

func handleUpdateAuth(w http.ResponseWriter, r *http.Request) {
	if settingsStore == nil {
		respondError(w, http.StatusServiceUnavailable, "settings not available")
		return
	}

	var req updateAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	overlay := config.AuthOverlay{
		SessionTimeout: req.SessionTimeout,
	}

	if req.Kerberos != nil {
		overlay.Kerberos = &config.KerberosOverlay{
			Enabled:    req.Kerberos.Enabled,
			KeytabPath: req.Kerberos.KeytabPath,
		}
	}

	restartFields := settingsStore.UpdateAuth(overlay)
	if err := settingsStore.Save(); err != nil {
		slog.Error("settings save failed", "section", "auth", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to save settings")
		return
	}

	// Apply session timeout immediately if changed
	if req.SessionTimeout != nil && settingsAuthStore != nil {
		settingsAuthStore.SetTimeout(*req.SessionTimeout)
		slog.Info("session timeout updated live", "hours", *req.SessionTimeout)
	}

	sess, _ := r.Context().Value(auth.SessionKey).(*auth.Session)
	actor := "unknown"
	if sess != nil {
		actor = sess.Username
	}
	slog.Info("settings updated", "section", "auth", "actor", actor, "restartFields", restartFields)

	respondJSON(w, http.StatusOK, map[string]any{
		"status":         "saved",
		"restartRequired": len(restartFields) > 0,
		"restartFields":  restartFields,
	})
}

// updateRBACRequest is the expected body for PUT /api/settings/rbac.
type updateRBACRequest struct {
	Roles []config.RoleMapping `json:"roles"`
}

func handleUpdateRBAC(w http.ResponseWriter, r *http.Request) {
	if settingsStore == nil {
		respondError(w, http.StatusServiceUnavailable, "settings not available")
		return
	}

	var req updateRBACRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.Roles) == 0 {
		respondError(w, http.StatusBadRequest, "at least one role is required")
		return
	}

	// Validate each role has a name
	for _, role := range req.Roles {
		if strings.TrimSpace(role.Role) == "" {
			respondError(w, http.StatusBadRequest, "role name is required")
			return
		}
	}

	settingsStore.UpdateRBAC(config.RBACOverlay{Roles: req.Roles})
	if err := settingsStore.Save(); err != nil {
		slog.Error("settings save failed", "section", "rbac", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to save settings")
		return
	}

	sess, _ := r.Context().Value(auth.SessionKey).(*auth.Session)
	actor := "unknown"
	if sess != nil {
		actor = sess.Username
	}
	slog.Info("settings updated", "section", "rbac", "actor", actor, "roles", len(req.Roles))

	respondJSON(w, http.StatusOK, map[string]any{
		"status":         "saved",
		"restartRequired": false,
	})
}

// updateApplicationRequest is the expected body for PUT /api/settings/application.
type updateApplicationRequest struct {
	AuditRetentionDays *int `json:"auditRetentionDays,omitempty"`
}

func handleUpdateApplication(w http.ResponseWriter, r *http.Request) {
	if settingsStore == nil {
		respondError(w, http.StatusServiceUnavailable, "settings not available")
		return
	}

	var req updateApplicationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.AuditRetentionDays != nil && *req.AuditRetentionDays < 1 {
		respondError(w, http.StatusBadRequest, "audit_retention_days must be at least 1")
		return
	}

	settingsStore.UpdateApplication(config.ApplicationOverlay{
		AuditRetentionDays: req.AuditRetentionDays,
	})
	if err := settingsStore.Save(); err != nil {
		slog.Error("settings save failed", "section", "application", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to save settings")
		return
	}

	sess, _ := r.Context().Value(auth.SessionKey).(*auth.Session)
	actor := "unknown"
	if sess != nil {
		actor = sess.Username
	}
	slog.Info("settings updated", "section", "application", "actor", actor)

	respondJSON(w, http.StatusOK, map[string]any{
		"status":         "saved",
		"restartRequired": false,
	})
}
