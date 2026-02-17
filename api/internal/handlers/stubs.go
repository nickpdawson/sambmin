package handlers

import "net/http"

// Auth handlers — Phase 1
func handleLogin(w http.ResponseWriter, r *http.Request)  { respondError(w, 501, "not implemented") }
func handleLogout(w http.ResponseWriter, r *http.Request) { respondError(w, 501, "not implemented") }
func handleMe(w http.ResponseWriter, r *http.Request)     { respondError(w, 501, "not implemented") }

// User handlers — Phase 2
func handleCreateUser(w http.ResponseWriter, r *http.Request)   { respondError(w, 501, "not implemented") }
func handleGetUser(w http.ResponseWriter, r *http.Request)      { respondError(w, 501, "not implemented") }
func handleUpdateUser(w http.ResponseWriter, r *http.Request)   { respondError(w, 501, "not implemented") }
func handleDeleteUser(w http.ResponseWriter, r *http.Request)   { respondError(w, 501, "not implemented") }
func handleResetPassword(w http.ResponseWriter, r *http.Request) {
	respondError(w, 501, "not implemented")
}
func handleEnableUser(w http.ResponseWriter, r *http.Request)  { respondError(w, 501, "not implemented") }
func handleDisableUser(w http.ResponseWriter, r *http.Request) { respondError(w, 501, "not implemented") }
func handleUnlockUser(w http.ResponseWriter, r *http.Request)  { respondError(w, 501, "not implemented") }

// Group handlers — Phase 2
func handleListGroups(w http.ResponseWriter, r *http.Request)       { respondError(w, 501, "not implemented") }
func handleCreateGroup(w http.ResponseWriter, r *http.Request)      { respondError(w, 501, "not implemented") }
func handleGetGroup(w http.ResponseWriter, r *http.Request)         { respondError(w, 501, "not implemented") }
func handleUpdateGroup(w http.ResponseWriter, r *http.Request)      { respondError(w, 501, "not implemented") }
func handleDeleteGroup(w http.ResponseWriter, r *http.Request)      { respondError(w, 501, "not implemented") }
func handleAddGroupMember(w http.ResponseWriter, r *http.Request)   { respondError(w, 501, "not implemented") }
func handleRemoveGroupMember(w http.ResponseWriter, r *http.Request) {
	respondError(w, 501, "not implemented")
}

// Computer handlers — Phase 2
func handleListComputers(w http.ResponseWriter, r *http.Request)  { respondError(w, 501, "not implemented") }
func handleGetComputer(w http.ResponseWriter, r *http.Request)    { respondError(w, 501, "not implemented") }
func handleDeleteComputer(w http.ResponseWriter, r *http.Request) { respondError(w, 501, "not implemented") }

// OU handlers — Phase 2
func handleListOUs(w http.ResponseWriter, r *http.Request)  { respondError(w, 501, "not implemented") }
func handleOUTree(w http.ResponseWriter, r *http.Request)   { respondError(w, 501, "not implemented") }
func handleCreateOU(w http.ResponseWriter, r *http.Request) { respondError(w, 501, "not implemented") }
func handleDeleteOU(w http.ResponseWriter, r *http.Request) { respondError(w, 501, "not implemented") }

// DNS handlers — Phase 3
func handleListDNSZones(w http.ResponseWriter, r *http.Request)    { respondError(w, 501, "not implemented") }
func handleCreateDNSZone(w http.ResponseWriter, r *http.Request)   { respondError(w, 501, "not implemented") }
func handleDeleteDNSZone(w http.ResponseWriter, r *http.Request)   { respondError(w, 501, "not implemented") }
func handleListDNSRecords(w http.ResponseWriter, r *http.Request)  { respondError(w, 501, "not implemented") }
func handleCreateDNSRecord(w http.ResponseWriter, r *http.Request) { respondError(w, 501, "not implemented") }
func handleUpdateDNSRecord(w http.ResponseWriter, r *http.Request) { respondError(w, 501, "not implemented") }
func handleDeleteDNSRecord(w http.ResponseWriter, r *http.Request) { respondError(w, 501, "not implemented") }
func handleDNSDiagnostics(w http.ResponseWriter, r *http.Request)  { respondError(w, 501, "not implemented") }

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
