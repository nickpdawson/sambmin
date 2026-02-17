package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/nickdawson/sambmin/internal/config"
)

func Register(mux *http.ServeMux, cfg *config.Config) {
	// Health check
	mux.HandleFunc("GET /api/health", handleHealth)

	// Auth
	mux.HandleFunc("POST /api/auth/login", handleLogin)
	mux.HandleFunc("POST /api/auth/logout", handleLogout)
	mux.HandleFunc("GET /api/auth/me", handleMe)

	// Dashboard (mock data for dev, real data when connected to DC)
	mux.HandleFunc("GET /api/dashboard/health", handleDashboardHealthMock)
	mux.HandleFunc("GET /api/dashboard/metrics", handleDashboardMetricsMock)
	mux.HandleFunc("GET /api/dashboard/activity", handleRecentActivityMock)

	// Users (mock data for dev)
	mux.HandleFunc("GET /api/users", handleListUsersMock)
	mux.HandleFunc("POST /api/users", handleCreateUser)
	mux.HandleFunc("GET /api/users/{dn}", handleGetUser)
	mux.HandleFunc("PUT /api/users/{dn}", handleUpdateUser)
	mux.HandleFunc("DELETE /api/users/{dn}", handleDeleteUser)
	mux.HandleFunc("POST /api/users/{dn}/reset-password", handleResetPassword)
	mux.HandleFunc("POST /api/users/{dn}/enable", handleEnableUser)
	mux.HandleFunc("POST /api/users/{dn}/disable", handleDisableUser)
	mux.HandleFunc("POST /api/users/{dn}/unlock", handleUnlockUser)

	// Groups
	mux.HandleFunc("GET /api/groups", handleListGroups)
	mux.HandleFunc("POST /api/groups", handleCreateGroup)
	mux.HandleFunc("GET /api/groups/{dn}", handleGetGroup)
	mux.HandleFunc("PUT /api/groups/{dn}", handleUpdateGroup)
	mux.HandleFunc("DELETE /api/groups/{dn}", handleDeleteGroup)
	mux.HandleFunc("POST /api/groups/{dn}/members", handleAddGroupMember)
	mux.HandleFunc("DELETE /api/groups/{dn}/members/{memberDn}", handleRemoveGroupMember)

	// Computers
	mux.HandleFunc("GET /api/computers", handleListComputers)
	mux.HandleFunc("GET /api/computers/{dn}", handleGetComputer)
	mux.HandleFunc("DELETE /api/computers/{dn}", handleDeleteComputer)

	// OUs
	mux.HandleFunc("GET /api/ous", handleListOUs)
	mux.HandleFunc("GET /api/ous/tree", handleOUTree)
	mux.HandleFunc("POST /api/ous", handleCreateOU)
	mux.HandleFunc("DELETE /api/ous/{dn}", handleDeleteOU)

	// DNS
	mux.HandleFunc("GET /api/dns/zones", handleListDNSZones)
	mux.HandleFunc("POST /api/dns/zones", handleCreateDNSZone)
	mux.HandleFunc("DELETE /api/dns/zones/{zone}", handleDeleteDNSZone)
	mux.HandleFunc("GET /api/dns/zones/{zone}/records", handleListDNSRecords)
	mux.HandleFunc("POST /api/dns/zones/{zone}/records", handleCreateDNSRecord)
	mux.HandleFunc("PUT /api/dns/zones/{zone}/records/{name}", handleUpdateDNSRecord)
	mux.HandleFunc("DELETE /api/dns/zones/{zone}/records/{name}", handleDeleteDNSRecord)
	mux.HandleFunc("GET /api/dns/diagnostics", handleDNSDiagnostics)

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
