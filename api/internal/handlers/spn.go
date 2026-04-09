package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/nickdawson/sambmin/internal/models"
	"github.com/nickdawson/sambmin/internal/validate"
)

// ──────────────────────────────────────────────────────────────────────────────
// SPN Management
// ──────────────────────────────────────────────────────────────────────────────

// handleListSPNs returns SPNs for a given account.
// GET /api/spn/{account}
func handleListSPNs(w http.ResponseWriter, r *http.Request) {
	account := r.PathValue("account")
	if account == "" {
		respondError(w, http.StatusBadRequest, "account name required")
		return
	}

	output, err := runDNSCommand("spn", "list", account)
	if err != nil {
		slog.Error("spn: list failed", "account", account, "error", err)
		respondError(w, http.StatusInternalServerError, "failed to list SPNs: "+err.Error())
		return
	}

	spns := parseSPNList(output, account)
	respondJSON(w, http.StatusOK, map[string]any{
		"spns":    spns,
		"account": account,
		"total":   len(spns),
	})
}

// handleAddSPN adds an SPN to an account.
// POST /api/spn
func handleAddSPN(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	var req struct {
		SPN     string `json:"spn"`
		Account string `json:"account"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.SPN == "" || req.Account == "" {
		respondError(w, http.StatusBadRequest, "spn and account required")
		return
	}
	if err := validate.SPN(req.SPN); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validate.SAMAccountName(req.Account); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if _, err := runSambaTool(r.Context(), sess, "spn", "add", req.SPN, req.Account); err != nil {
		slog.Error("spn add failed", "spn", req.SPN, "account", req.Account, "actor", sess.Username, "error", err)
		respondSafeError(w, http.StatusInternalServerError, "SPN registration failed", err)
		return
	}

	slog.Info("spn added", "spn", req.SPN, "account", req.Account, "actor", sess.Username)
	LogAudit(sess.Username, "spn.add", "", "spn", "", true, "spn="+req.SPN+" account="+req.Account)
	respondJSON(w, http.StatusCreated, map[string]any{
		"success": true,
		"spn":     req.SPN,
		"account": req.Account,
	})
}

// handleDeleteSPN removes an SPN from an account.
// DELETE /api/spn
func handleDeleteSPN(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	var req struct {
		SPN     string `json:"spn"`
		Account string `json:"account"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.SPN == "" || req.Account == "" {
		respondError(w, http.StatusBadRequest, "spn and account required")
		return
	}
	if err := validate.SPN(req.SPN); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validate.SAMAccountName(req.Account); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if _, err := runSambaTool(r.Context(), sess, "spn", "delete", req.SPN, req.Account); err != nil {
		slog.Error("spn delete failed", "spn", req.SPN, "account", req.Account, "actor", sess.Username, "error", err)
		respondSafeError(w, http.StatusInternalServerError, "SPN removal failed", err)
		return
	}

	slog.Info("spn deleted", "spn", req.SPN, "account", req.Account, "actor", sess.Username)
	LogAudit(sess.Username, "spn.delete", "", "spn", "", true, "spn="+req.SPN+" account="+req.Account)
	respondJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"spn":     req.SPN,
		"account": req.Account,
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Delegation Management
// ──────────────────────────────────────────────────────────────────────────────

// handleGetDelegation returns delegation configuration for an account.
// GET /api/delegation/{account}
func handleGetDelegation(w http.ResponseWriter, r *http.Request) {
	account := r.PathValue("account")
	if account == "" {
		respondError(w, http.StatusBadRequest, "account name required")
		return
	}

	output, err := runDNSCommand("delegation", "show", account)
	if err != nil {
		// "no delegation" is a valid state
		if strings.Contains(err.Error(), "no msDS-AllowedToDelegateTo") ||
			strings.Contains(err.Error(), "not set") {
			respondJSON(w, http.StatusOK, models.DelegationInfo{
				Account:         account,
				AllowedServices: []string{},
			})
			return
		}
		slog.Error("delegation: show failed", "account", account, "error", err)
		respondError(w, http.StatusInternalServerError, "failed to get delegation info: "+err.Error())
		return
	}

	info := parseDelegationShow(output, account)
	respondJSON(w, http.StatusOK, info)
}

// handleAddDelegationService adds a service to constrained delegation.
// POST /api/delegation/{account}/service
func handleAddDelegationService(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	account := r.PathValue("account")
	if account == "" {
		respondError(w, http.StatusBadRequest, "account name required")
		return
	}

	var req struct {
		Service string `json:"service"` // e.g., "cifs/fileserver.example.com"
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Service == "" {
		respondError(w, http.StatusBadRequest, "service SPN required")
		return
	}
	if err := validate.SPN(req.Service); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if _, err := runSambaTool(r.Context(), sess, "delegation", "add-service", account, req.Service); err != nil {
		slog.Error("delegation add-service failed", "account", account, "service", req.Service, "actor", sess.Username, "error", err)
		respondSafeError(w, http.StatusInternalServerError, "delegation service addition failed", err)
		return
	}

	slog.Info("delegation service added", "account", account, "service", req.Service, "actor", sess.Username)
	LogAudit(sess.Username, "delegation.add", "", "delegation", "", true, "account="+account+" service="+req.Service)
	respondJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"account": account,
		"service": req.Service,
	})
}

// handleRemoveDelegationService removes a service from constrained delegation.
// DELETE /api/delegation/{account}/service
func handleRemoveDelegationService(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	account := r.PathValue("account")
	if account == "" {
		respondError(w, http.StatusBadRequest, "account name required")
		return
	}

	var req struct {
		Service string `json:"service"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Service == "" {
		respondError(w, http.StatusBadRequest, "service SPN required")
		return
	}
	if err := validate.SPN(req.Service); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if _, err := runSambaTool(r.Context(), sess, "delegation", "del-service", account, req.Service); err != nil {
		slog.Error("delegation del-service failed", "account", account, "service", req.Service, "actor", sess.Username, "error", err)
		respondSafeError(w, http.StatusInternalServerError, "delegation service removal failed", err)
		return
	}

	slog.Info("delegation service removed", "account", account, "service", req.Service, "actor", sess.Username)
	LogAudit(sess.Username, "delegation.remove", "", "delegation", "", true, "account="+account+" service="+req.Service)
	respondJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"account": account,
		"service": req.Service,
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// SPN & Delegation Parsers
// ──────────────────────────────────────────────────────────────────────────────

// parseSPNList parses `samba-tool spn list <account>` output.
// Output format (one SPN per line, after a header):
//
//	User CN=myhost,CN=Computers,DC=example,DC=com has the following servicePrincipalName:
//	   HTTP/myhost.example.com
//	   HOST/myhost.example.com
//	   HOST/myhost
func parseSPNList(output, account string) []models.SPN {
	var spns []models.SPN
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Skip the header line
		if strings.Contains(line, "servicePrincipalName") || strings.HasPrefix(line, "User ") {
			continue
		}
		spns = append(spns, models.SPN{
			Value:   line,
			Account: account,
		})
	}
	return spns
}

// parseDelegationShow parses `samba-tool delegation show <account>` output.
// Output format:
//
//	Incoming Delegations:
//	  (none)
//	Outgoing Delegations:
//	  cifs/fileserver.example.com
//	  HTTP/web.example.com
//
// Or for unconstrained:
//
//	Account is trusted for delegation
func parseDelegationShow(output, account string) models.DelegationInfo {
	info := models.DelegationInfo{
		Account:         account,
		AllowedServices: []string{},
	}

	if strings.Contains(output, "trusted for delegation") {
		info.Unconstrained = true
	}

	inOutgoing := false
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Outgoing Delegations") {
			inOutgoing = true
			continue
		}
		if strings.HasPrefix(line, "Incoming Delegations") {
			inOutgoing = false
			continue
		}
		if inOutgoing && line != "" && line != "(none)" {
			info.Constrained = true
			info.AllowedServices = append(info.AllowedServices, line)
		}
	}

	return info
}
