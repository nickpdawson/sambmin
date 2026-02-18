package handlers

import "net/http"

// Auth handlers — mock mode (no LDAP available)
func handleLogin(w http.ResponseWriter, r *http.Request)  { respondError(w, 501, "not implemented") }
func handleLogout(w http.ResponseWriter, r *http.Request) { respondError(w, 501, "not implemented") }
func handleMe(w http.ResponseWriter, r *http.Request)     { respondError(w, 501, "not implemented") }

// User read stub for mock mode
func handleGetUserStub(w http.ResponseWriter, r *http.Request) { respondError(w, 501, "not implemented") }

// Group stub handlers — mock mode reads
func handleListGroups(w http.ResponseWriter, r *http.Request) { respondError(w, 501, "not implemented") }
func handleGetGroup(w http.ResponseWriter, r *http.Request)   { respondError(w, 501, "not implemented") }

// Computer stub handlers — mock mode reads
func handleListComputers(w http.ResponseWriter, r *http.Request) { respondError(w, 501, "not implemented") }
func handleGetComputer(w http.ResponseWriter, r *http.Request)   { respondError(w, 501, "not implemented") }

// OU stub handlers — mock mode reads
func handleListOUs(w http.ResponseWriter, r *http.Request) { respondError(w, 501, "not implemented") }
func handleOUTree(w http.ResponseWriter, r *http.Request)  { respondError(w, 501, "not implemented") }

// Replication handlers — Phase 4
func handleReplicationTopology(w http.ResponseWriter, r *http.Request) {
	respondError(w, 501, "not implemented")
}
func handleReplicationStatus(w http.ResponseWriter, r *http.Request) {
	respondError(w, 501, "not implemented")
}
func handleForceSync(w http.ResponseWriter, r *http.Request) { respondError(w, 501, "not implemented") }

// Sites handlers — Phase 4
func handleListSites(w http.ResponseWriter, r *http.Request)   { respondError(w, 501, "not implemented") }
func handleCreateSite(w http.ResponseWriter, r *http.Request)  { respondError(w, 501, "not implemented") }
func handleListSubnets(w http.ResponseWriter, r *http.Request) { respondError(w, 501, "not implemented") }

// FSMO handlers — Phase 4
func handleGetFSMORoles(w http.ResponseWriter, r *http.Request) {
	respondError(w, 501, "not implemented")
}
func handleTransferFSMO(w http.ResponseWriter, r *http.Request) {
	respondError(w, 501, "not implemented")
}

// Audit handlers — Phase 6
func handleListAuditLog(w http.ResponseWriter, r *http.Request) {
	respondError(w, 501, "not implemented")
}
