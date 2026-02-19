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

// Replication stubs — mock mode (no LDAP)
func handleReplicationTopology(w http.ResponseWriter, r *http.Request) {
	respondError(w, 501, "not implemented")
}
func handleReplicationStatus(w http.ResponseWriter, r *http.Request) {
	respondError(w, 501, "not implemented")
}

// Sites stubs — mock mode
func handleListSites(w http.ResponseWriter, r *http.Request)   { respondError(w, 501, "not implemented") }
func handleListSubnets(w http.ResponseWriter, r *http.Request) { respondError(w, 501, "not implemented") }

// FSMO stubs — mock mode
func handleGetFSMORoles(w http.ResponseWriter, r *http.Request) {
	respondError(w, 501, "not implemented")
}
