package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/nickdawson/sambmin/internal/config"
	"github.com/nickdawson/sambmin/internal/directory"
)

// dirClient is the shared directory client, nil when running in mock mode.
var dirClient *directory.Client

// Register wires all API routes. If dir is nil, mock handlers are used for reads.
func Register(mux *http.ServeMux, cfg *config.Config, dir *directory.Client) {
	dirClient = dir
	handlerConfig = cfg

	if dir != nil {
		slog.Info("routes: using live LDAP handlers")
	} else {
		slog.Info("routes: using mock data handlers")
	}

	// Health check
	mux.HandleFunc("GET /api/health", handleHealth)

	// Auth
	if sessionStore != nil {
		mux.HandleFunc("POST /api/auth/login", handleLoginImpl)
		mux.HandleFunc("POST /api/auth/logout", handleLogoutImpl)
		mux.HandleFunc("GET /api/auth/me", handleMeImpl)
	} else {
		mux.HandleFunc("POST /api/auth/login", handleLogin)
		mux.HandleFunc("POST /api/auth/logout", handleLogout)
		mux.HandleFunc("GET /api/auth/me", handleMe)
	}

	// Dashboard
	if dir != nil {
		mux.HandleFunc("GET /api/dashboard/metrics", handleDashboardMetrics)
	} else {
		mux.HandleFunc("GET /api/dashboard/metrics", handleDashboardMetricsMock)
	}
	if dir != nil {
		mux.HandleFunc("GET /api/dashboard/health", handleDashboardHealthLive)
		mux.HandleFunc("GET /api/dashboard/activity", handleRecentActivityLive)
	} else {
		mux.HandleFunc("GET /api/dashboard/health", handleDashboardHealthMock)
		mux.HandleFunc("GET /api/dashboard/activity", handleRecentActivityMock)
	}

	// Self-service
	mux.HandleFunc("GET /api/self", handleSelfProfile)
	mux.HandleFunc("PUT /api/self", handleSelfProfileUpdate)
	mux.HandleFunc("POST /api/self/password", handleSelfPasswordChange)

	// Users
	if dir != nil {
		mux.HandleFunc("GET /api/users", handleListUsers)
		mux.HandleFunc("GET /api/users/{dn}", handleGetUser)
	} else {
		mux.HandleFunc("GET /api/users", handleListUsersMock)
		mux.HandleFunc("GET /api/users/{dn}", handleGetUserStub)
	}
	mux.HandleFunc("POST /api/users", handleCreateUser)
	mux.HandleFunc("PUT /api/users/{dn}", handleUpdateUser)
	mux.HandleFunc("DELETE /api/users/{dn}", handleDeleteUser)
	mux.HandleFunc("POST /api/users/{dn}/reset-password", handleResetPassword)
	mux.HandleFunc("POST /api/users/{dn}/enable", handleEnableUser)
	mux.HandleFunc("POST /api/users/{dn}/disable", handleDisableUser)
	mux.HandleFunc("POST /api/users/{dn}/unlock", handleUnlockUser)

	// Groups
	if dir != nil {
		mux.HandleFunc("GET /api/groups", handleListGroupsLive)
		mux.HandleFunc("GET /api/groups/{dn}", handleGetGroupLive)
	} else {
		mux.HandleFunc("GET /api/groups", handleListGroups)
		mux.HandleFunc("GET /api/groups/{dn}", handleGetGroup)
	}
	mux.HandleFunc("POST /api/groups", handleCreateGroup)
	mux.HandleFunc("PUT /api/groups/{dn}", handleUpdateGroup)
	mux.HandleFunc("DELETE /api/groups/{dn}", handleDeleteGroup)
	mux.HandleFunc("POST /api/groups/{dn}/members", handleAddGroupMember)
	mux.HandleFunc("DELETE /api/groups/{dn}/members/{memberDn}", handleRemoveGroupMember)

	// Computers
	if dir != nil {
		mux.HandleFunc("GET /api/computers", handleListComputersLive)
		mux.HandleFunc("GET /api/computers/{dn}", handleGetComputerLive)
	} else {
		mux.HandleFunc("GET /api/computers", handleListComputers)
		mux.HandleFunc("GET /api/computers/{dn}", handleGetComputer)
	}
	mux.HandleFunc("DELETE /api/computers/{dn}", handleDeleteComputer)

	// OUs
	if dir != nil {
		mux.HandleFunc("GET /api/ous", handleListOUsLive)
		mux.HandleFunc("GET /api/ous/tree", handleOUTreeLive)
	} else {
		mux.HandleFunc("GET /api/ous", handleListOUs)
		mux.HandleFunc("GET /api/ous/tree", handleOUTree)
	}
	mux.HandleFunc("POST /api/ous", handleCreateOU)
	mux.HandleFunc("DELETE /api/ous/{dn}", handleDeleteOU)

	// DNS
	if dir != nil {
		mux.HandleFunc("GET /api/dns/zones", handleListDNSZonesLive)
		mux.HandleFunc("GET /api/dns/zones/{zone}/records", handleListDNSRecordsLive)
		mux.HandleFunc("GET /api/dns/diagnostics", handleDNSDiagnosticsLive)
	} else {
		mux.HandleFunc("GET /api/dns/zones", handleListDNSZonesMock)
		mux.HandleFunc("GET /api/dns/zones/{zone}/records", handleListDNSRecordsMock)
		mux.HandleFunc("GET /api/dns/diagnostics", handleDNSDiagnosticsMock)
	}
	mux.HandleFunc("POST /api/dns/zones", handleCreateDNSZone)
	mux.HandleFunc("DELETE /api/dns/zones/{zone}", handleDeleteDNSZone)
	mux.HandleFunc("POST /api/dns/zones/{zone}/records", handleCreateDNSRecord)
	mux.HandleFunc("PUT /api/dns/zones/{zone}/records/{name}", handleUpdateDNSRecord)
	mux.HandleFunc("DELETE /api/dns/zones/{zone}/records/{name}", handleDeleteDNSRecord)

	// Settings (mock data for dev)
	mux.HandleFunc("GET /api/settings", handleGetSettingsMock)

	// Replication
	mux.HandleFunc("GET /api/replication/topology", handleReplicationTopology)
	mux.HandleFunc("GET /api/replication/status", handleReplicationStatus)
	mux.HandleFunc("POST /api/replication/sync", handleForceSync)

	// Sites
	mux.HandleFunc("GET /api/sites", handleListSites)
	mux.HandleFunc("POST /api/sites", handleCreateSite)
	mux.HandleFunc("GET /api/sites/{site}/subnets", handleListSubnets)

	// FSMO Roles
	mux.HandleFunc("GET /api/fsmo", handleGetFSMORoles)
	mux.HandleFunc("POST /api/fsmo/transfer", handleTransferFSMO)

	// Audit Log
	mux.HandleFunc("GET /api/audit", handleListAuditLog)
}

// respondJSON writes a JSON response with status code
func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// respondError writes a JSON error response
func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
