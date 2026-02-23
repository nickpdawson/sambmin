package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/nickdawson/sambmin/internal/auth"
	"github.com/nickdawson/sambmin/internal/config"
	"github.com/nickdawson/sambmin/internal/directory"
)

// dirClient is the shared directory client, nil when running in mock mode.
var dirClient *directory.Client

// Register wires all API routes. If dir is nil, mock handlers are used for reads.
// If store is non-nil, all routes except health and login require authentication.
func Register(mux *http.ServeMux, cfg *config.Config, dir *directory.Client, store *auth.Store) {
	dirClient = dir
	handlerConfig = cfg

	if dir != nil {
		slog.Info("routes: using live LDAP handlers")
	} else {
		slog.Info("routes: using mock data handlers")
	}

	// --- Public routes (no auth required) ---
	mux.HandleFunc("GET /api/health", handleHealth)

	// Auth endpoints: login is public, logout/me need auth but are registered
	// on the public mux so the middleware wrapper doesn't double-check them.
	if sessionStore != nil {
		mux.HandleFunc("POST /api/auth/login", handleLoginImpl)
	} else {
		mux.HandleFunc("POST /api/auth/login", handleLogin)
	}

	// --- Protected routes (auth required) ---
	// Build a secondary mux for protected routes, then wrap with RequireAuth.
	protected := http.NewServeMux()

	// Auth (protected)
	if sessionStore != nil {
		protected.HandleFunc("POST /api/auth/logout", handleLogoutImpl)
		protected.HandleFunc("GET /api/auth/me", handleMeImpl)
	} else {
		protected.HandleFunc("POST /api/auth/logout", handleLogout)
		protected.HandleFunc("GET /api/auth/me", handleMe)
	}

	// Dashboard (read-only, authenticated)
	if dir != nil {
		protected.HandleFunc("GET /api/dashboard/metrics", handleDashboardMetrics)
		protected.HandleFunc("GET /api/dashboard/health", handleDashboardHealthLive)
		protected.HandleFunc("GET /api/dashboard/activity", handleRecentActivityLive)
	} else {
		protected.HandleFunc("GET /api/dashboard/metrics", handleDashboardMetricsMock)
		protected.HandleFunc("GET /api/dashboard/health", handleDashboardHealthMock)
		protected.HandleFunc("GET /api/dashboard/activity", handleRecentActivityMock)
	}

	// Self-service (any authenticated user)
	protected.HandleFunc("GET /api/self", handleSelfProfile)
	protected.HandleFunc("PUT /api/self", handleSelfProfileUpdate)
	protected.HandleFunc("POST /api/self/password", handleSelfPasswordChange)

	// Users (read = authenticated, write = operator)
	if dir != nil {
		protected.HandleFunc("GET /api/users", handleListUsers)
		protected.HandleFunc("GET /api/users/{dn}", handleGetUser)
	} else {
		protected.HandleFunc("GET /api/users", handleListUsersMock)
		protected.HandleFunc("GET /api/users/{dn}", handleGetUserStub)
	}
	protected.Handle("POST /api/users", requireRole(auth.RoleOperator, handleCreateUser))
	protected.Handle("PUT /api/users/{dn}", requireRole(auth.RoleOperator, handleUpdateUser))
	protected.Handle("DELETE /api/users/{dn}", requireRole(auth.RoleOperator, handleDeleteUser))
	protected.Handle("POST /api/users/{dn}/reset-password", requireRole(auth.RoleOperator, handleResetPassword))
	protected.Handle("POST /api/users/{dn}/enable", requireRole(auth.RoleOperator, handleEnableUser))
	protected.Handle("POST /api/users/{dn}/disable", requireRole(auth.RoleOperator, handleDisableUser))
	protected.Handle("POST /api/users/{dn}/unlock", requireRole(auth.RoleOperator, handleUnlockUser))
	protected.Handle("POST /api/users/{dn}/rename", requireRole(auth.RoleOperator, handleRenameUser))

	// Groups (read = authenticated, write = operator)
	if dir != nil {
		protected.HandleFunc("GET /api/groups", handleListGroupsLive)
		protected.HandleFunc("GET /api/groups/{dn}", handleGetGroupLive)
	} else {
		protected.HandleFunc("GET /api/groups", handleListGroups)
		protected.HandleFunc("GET /api/groups/{dn}", handleGetGroup)
	}
	protected.Handle("POST /api/groups", requireRole(auth.RoleOperator, handleCreateGroup))
	protected.Handle("PUT /api/groups/{dn}", requireRole(auth.RoleOperator, handleUpdateGroup))
	protected.Handle("DELETE /api/groups/{dn}", requireRole(auth.RoleOperator, handleDeleteGroup))
	protected.Handle("POST /api/groups/{dn}/members", requireRole(auth.RoleOperator, handleAddGroupMember))
	protected.Handle("DELETE /api/groups/{dn}/members/{memberDn}", requireRole(auth.RoleOperator, handleRemoveGroupMember))
	protected.Handle("POST /api/groups/{dn}/rename", requireRole(auth.RoleOperator, handleRenameGroup))

	// Computers (read = authenticated, write = operator)
	if dir != nil {
		protected.HandleFunc("GET /api/computers", handleListComputersLive)
		protected.HandleFunc("GET /api/computers/{dn}", handleGetComputerLive)
	} else {
		protected.HandleFunc("GET /api/computers", handleListComputers)
		protected.HandleFunc("GET /api/computers/{dn}", handleGetComputer)
	}
	protected.Handle("POST /api/computers", requireRole(auth.RoleOperator, handleCreateComputer))
	protected.Handle("DELETE /api/computers/{dn}", requireRole(auth.RoleOperator, handleDeleteComputer))
	protected.Handle("POST /api/computers/{dn}/move", requireRole(auth.RoleOperator, handleMoveComputer))

	// Contacts (read = authenticated, write = operator)
	if dir != nil {
		protected.HandleFunc("GET /api/contacts", handleListContactsLive)
		protected.HandleFunc("GET /api/contacts/{dn}", handleGetContactLive)
	} else {
		protected.HandleFunc("GET /api/contacts", handleListContactsMock)
		protected.HandleFunc("GET /api/contacts/{dn}", handleGetContactMock)
	}
	protected.Handle("POST /api/contacts", requireRole(auth.RoleOperator, handleCreateContact))
	protected.Handle("PUT /api/contacts/{dn}", requireRole(auth.RoleOperator, handleUpdateContact))
	protected.Handle("DELETE /api/contacts/{dn}", requireRole(auth.RoleOperator, handleDeleteContact))
	protected.Handle("POST /api/contacts/{dn}/move", requireRole(auth.RoleOperator, handleMoveContact))
	protected.Handle("POST /api/contacts/{dn}/rename", requireRole(auth.RoleOperator, handleRenameContact))

	// OUs (read = authenticated, write = operator)
	if dir != nil {
		protected.HandleFunc("GET /api/ous", handleListOUsLive)
		protected.HandleFunc("GET /api/ous/tree", handleOUTreeLive)
		protected.HandleFunc("GET /api/ous/tree/full", handleOUTreeFullLive)
		protected.HandleFunc("GET /api/ous/{dn}/contents", handleOUContentsLive)
	} else {
		protected.HandleFunc("GET /api/ous", handleListOUs)
		protected.HandleFunc("GET /api/ous/tree", handleOUTree)
	}
	protected.Handle("POST /api/ous", requireRole(auth.RoleOperator, handleCreateOU))
	protected.Handle("DELETE /api/ous/{dn}", requireRole(auth.RoleOperator, handleDeleteOU))

	// DNS (read = authenticated, write = dns admin)
	if dir != nil {
		protected.HandleFunc("GET /api/dns/zones", handleListDNSZonesLive)
		protected.HandleFunc("GET /api/dns/zones/{zone}/records", handleListDNSRecordsLive)
		protected.HandleFunc("GET /api/dns/diagnostics", handleDNSDiagnosticsLive)
	} else {
		protected.HandleFunc("GET /api/dns/zones", handleListDNSZonesMock)
		protected.HandleFunc("GET /api/dns/zones/{zone}/records", handleListDNSRecordsMock)
		protected.HandleFunc("GET /api/dns/diagnostics", handleDNSDiagnosticsMock)
	}
	protected.Handle("POST /api/dns/zones", requireRole(auth.RoleDNSAdmin, handleCreateDNSZone))
	protected.Handle("DELETE /api/dns/zones/{zone}", requireRole(auth.RoleDNSAdmin, handleDeleteDNSZone))
	protected.Handle("POST /api/dns/zones/{zone}/records", requireRole(auth.RoleDNSAdmin, handleCreateDNSRecord))
	protected.Handle("PUT /api/dns/zones/{zone}/records/{name}", requireRole(auth.RoleDNSAdmin, handleUpdateDNSRecord))
	protected.Handle("DELETE /api/dns/zones/{zone}/records/{name}", requireRole(auth.RoleDNSAdmin, handleDeleteDNSRecord))

	// DNS Deep Dive (read = authenticated, zone options = dns admin)
	protected.HandleFunc("GET /api/dns/serverinfo", handleDNSServerInfo)
	protected.HandleFunc("GET /api/dns/zones/{zone}/info", handleDNSZoneInfo)
	protected.Handle("PUT /api/dns/zones/{zone}/options", requireRole(auth.RoleDNSAdmin, handleDNSZoneOptions))
	protected.HandleFunc("POST /api/dns/query", handleDNSQuery)
	protected.HandleFunc("GET /api/dns/srv-validator", handleDNSSRVValidator)
	protected.HandleFunc("GET /api/dns/consistency", handleDNSConsistency)
	protected.HandleFunc("GET /api/dns/limitations", handleDNSLimitations)

	// Search (authenticated)
	protected.HandleFunc("POST /api/search", handleSearch)
	protected.HandleFunc("GET /api/search/saved", handleListSavedQueries)
	protected.HandleFunc("POST /api/search/saved", handleCreateSavedQuery)
	protected.HandleFunc("DELETE /api/search/saved/{id}", handleDeleteSavedQuery)

	// Password Policy (read = authenticated, write = admin)
	protected.HandleFunc("GET /api/password-policy", handleGetPasswordPolicy)
	protected.Handle("PUT /api/password-policy", requireRole(auth.RoleAdmin, handleUpdatePasswordPolicy))
	protected.HandleFunc("GET /api/password-policy/pso", handleListPSOs)
	protected.Handle("POST /api/password-policy/pso", requireRole(auth.RoleAdmin, handleCreatePSO))
	protected.Handle("PUT /api/password-policy/pso/{name}", requireRole(auth.RoleAdmin, handleUpdatePSO))
	protected.Handle("DELETE /api/password-policy/pso/{name}", requireRole(auth.RoleAdmin, handleDeletePSO))
	protected.Handle("POST /api/password-policy/pso/{name}/apply", requireRole(auth.RoleAdmin, handleApplyPSO))
	protected.Handle("POST /api/password-policy/pso/{name}/unapply", requireRole(auth.RoleAdmin, handleUnapplyPSO))
	protected.HandleFunc("GET /api/password-policy/user/{username}", handleGetEffectivePolicy)
	protected.HandleFunc("POST /api/password-policy/test", handleTestPassword)

	// Settings (mock data for dev)
	protected.HandleFunc("GET /api/settings", handleGetSettingsMock)

	// Replication (read = authenticated, sync = admin)
	if dir != nil {
		protected.HandleFunc("GET /api/replication/topology", handleReplicationTopologyLive)
		protected.HandleFunc("GET /api/replication/status", handleReplicationStatusLive)
	} else {
		protected.HandleFunc("GET /api/replication/topology", handleReplicationTopology)
		protected.HandleFunc("GET /api/replication/status", handleReplicationStatus)
	}
	protected.Handle("POST /api/replication/sync", requireRole(auth.RoleAdmin, handleForceSyncLive))

	// Sites (read = authenticated, create = admin)
	if dir != nil {
		protected.HandleFunc("GET /api/sites", handleListSitesLive)
		protected.HandleFunc("GET /api/sites/{site}/subnets", handleListSubnetsLive)
	} else {
		protected.HandleFunc("GET /api/sites", handleListSites)
		protected.HandleFunc("GET /api/sites/{site}/subnets", handleListSubnets)
	}
	protected.Handle("POST /api/sites", requireRole(auth.RoleAdmin, handleCreateSiteLive))

	// FSMO Roles (read = authenticated, transfer = admin)
	if dir != nil {
		protected.HandleFunc("GET /api/fsmo", handleGetFSMORolesLive)
	} else {
		protected.HandleFunc("GET /api/fsmo", handleGetFSMORoles)
	}
	protected.Handle("POST /api/fsmo/transfer", requireRole(auth.RoleAdmin, handleTransferFSMOLive))

	// Audit Log (authenticated)
	protected.HandleFunc("GET /api/audit", handleListAuditLogLive)

	// GPO Management (read = authenticated, write = admin)
	protected.HandleFunc("GET /api/gpo", handleListGPOs)
	protected.HandleFunc("GET /api/gpo/{id}", handleGetGPO)
	protected.Handle("POST /api/gpo", requireRole(auth.RoleAdmin, handleCreateGPO))
	protected.Handle("DELETE /api/gpo/{id}", requireRole(auth.RoleAdmin, handleDeleteGPO))
	protected.Handle("POST /api/gpo/{id}/link", requireRole(auth.RoleAdmin, handleLinkGPO))
	protected.Handle("DELETE /api/gpo/{id}/link", requireRole(auth.RoleAdmin, handleUnlinkGPO))
	protected.HandleFunc("GET /api/gpo/links/{ou}", handleGetGPOLinks)

	// SPN Management (read = authenticated, write = admin)
	protected.HandleFunc("GET /api/spn/{account}", handleListSPNs)
	protected.Handle("POST /api/spn", requireRole(auth.RoleAdmin, handleAddSPN))
	protected.Handle("DELETE /api/spn", requireRole(auth.RoleAdmin, handleDeleteSPN))

	// Delegation Management (read = authenticated, write = admin)
	protected.HandleFunc("GET /api/delegation/{account}", handleGetDelegation)
	protected.Handle("POST /api/delegation/{account}/service", requireRole(auth.RoleAdmin, handleAddDelegationService))
	protected.Handle("DELETE /api/delegation/{account}/service", requireRole(auth.RoleAdmin, handleRemoveDelegationService))

	// Kerberos (read = authenticated, keytab export = admin)
	if dir != nil {
		protected.HandleFunc("GET /api/kerberos/policy", handleKerberosPolicy)
		protected.HandleFunc("GET /api/kerberos/accounts", handleKerberosAccounts)
	}
	protected.Handle("POST /api/kerberos/keytab", requireRole(auth.RoleAdmin, handleExportKeytab))

	// Schema Browser (authenticated)
	if dir != nil {
		protected.HandleFunc("GET /api/schema/classes", handleListSchemaClasses)
		protected.HandleFunc("GET /api/schema/attributes", handleListSchemaAttributes)
	}

	// Wrap all protected routes with RequireAuth middleware
	if store != nil {
		mux.Handle("/api/", auth.RequireAuth(store)(protected))
	} else {
		// No auth store (mock mode) — register protected routes directly
		mux.Handle("/api/", protected)
	}
}

// requireRole wraps a handler function with RBAC role checking.
func requireRole(role auth.Role, handler http.HandlerFunc) http.Handler {
	return auth.RequireRole(role)(handler)
}

// respondJSON writes a JSON response with status code
func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("respondJSON: failed to encode response", "error", err)
	}
}

// respondError writes a JSON error response
func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

// respondSafeError logs the real error but sends a sanitized message to the client.
func respondSafeError(w http.ResponseWriter, status int, publicMsg string, err error) {
	slog.Error(publicMsg, "error", err)
	respondError(w, status, publicMsg)
}
