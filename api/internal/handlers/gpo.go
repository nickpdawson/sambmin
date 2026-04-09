package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/nickdawson/sambmin/internal/models"
)

// ──────────────────────────────────────────────────────────────────────────────
// GPO Management
// ──────────────────────────────────────────────────────────────────────────────

// handleListGPOs returns all Group Policy Objects in the domain.
// GET /api/gpo
func handleListGPOs(w http.ResponseWriter, _ *http.Request) {
	output, err := runDNSCommand("gpo", "listall")
	if err != nil {
		slog.Error("gpo: listall failed", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to list GPOs: "+err.Error())
		return
	}

	gpos := parseGPOListAll(output)
	respondJSON(w, http.StatusOK, map[string]any{
		"gpos":  gpos,
		"total": len(gpos),
	})
}

// handleGetGPO returns details for a single GPO.
// GET /api/gpo/{id}
func handleGetGPO(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "GPO ID required")
		return
	}

	output, err := runDNSCommand("gpo", "show", id)
	if err != nil {
		slog.Error("gpo: show failed", "id", id, "error", err)
		respondError(w, http.StatusInternalServerError, "failed to get GPO: "+err.Error())
		return
	}

	gpo := parseGPOShow(output)
	if gpo.ID == "" {
		gpo.ID = id
	}
	respondJSON(w, http.StatusOK, gpo)
}

// handleCreateGPO creates a new GPO.
// POST /api/gpo
func handleCreateGPO(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "GPO name required")
		return
	}

	output, err := runSambaTool(r.Context(), sess, "gpo", "create", req.Name)
	if err != nil {
		slog.Error("gpo create failed", "name", req.Name, "actor", sess.Username, "error", err)
		respondSafeError(w, http.StatusInternalServerError, "GPO creation failed", err)
		return
	}

	// Try to extract GPO GUID from output
	gpoID := extractGPOID(output)

	slog.Info("gpo created", "name", req.Name, "id", gpoID, "actor", sess.Username)
	LogAudit(sess.Username, "gpo.create", "", "gpo", "", true, "name="+req.Name)
	respondJSON(w, http.StatusCreated, map[string]any{
		"success": true,
		"name":    req.Name,
		"id":      gpoID,
	})
}

// handleDeleteGPO deletes a GPO.
// DELETE /api/gpo/{id}
func handleDeleteGPO(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	id := r.PathValue("id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "GPO ID required")
		return
	}

	if _, err := runSambaTool(r.Context(), sess, "gpo", "del", id); err != nil {
		slog.Error("gpo delete failed", "id", id, "actor", sess.Username, "error", err)
		respondSafeError(w, http.StatusInternalServerError, "GPO deletion failed", err)
		return
	}

	slog.Info("gpo deleted", "id", id, "actor", sess.Username)
	LogAudit(sess.Username, "gpo.delete", "", "gpo", "", true, "id="+id)
	respondJSON(w, http.StatusOK, map[string]any{"success": true, "id": id})
}

// handleLinkGPO links a GPO to an OU.
// POST /api/gpo/{id}/link
func handleLinkGPO(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	id := r.PathValue("id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "GPO ID required")
		return
	}

	var req struct {
		OUDN string `json:"ouDn"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.OUDN == "" {
		respondError(w, http.StatusBadRequest, "OU DN required")
		return
	}

	if _, err := runSambaTool(r.Context(), sess, "gpo", "setlink", req.OUDN, id); err != nil {
		slog.Error("gpo link failed", "id", id, "ou", req.OUDN, "actor", sess.Username, "error", err)
		respondSafeError(w, http.StatusInternalServerError, "GPO link failed", err)
		return
	}

	slog.Info("gpo linked", "id", id, "ou", req.OUDN, "actor", sess.Username)
	LogAudit(sess.Username, "gpo.link", req.OUDN, "gpo", "", true, "gpo="+id)
	respondJSON(w, http.StatusOK, map[string]any{"success": true, "id": id, "ouDn": req.OUDN})
}

// handleUnlinkGPO removes a GPO link from an OU.
// DELETE /api/gpo/{id}/link
func handleUnlinkGPO(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	id := r.PathValue("id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "GPO ID required")
		return
	}

	var req struct {
		OUDN string `json:"ouDn"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.OUDN == "" {
		respondError(w, http.StatusBadRequest, "OU DN required")
		return
	}

	if _, err := runSambaTool(r.Context(), sess, "gpo", "dellink", req.OUDN, id); err != nil {
		slog.Error("gpo unlink failed", "id", id, "ou", req.OUDN, "actor", sess.Username, "error", err)
		respondSafeError(w, http.StatusInternalServerError, "GPO unlink failed", err)
		return
	}

	slog.Info("gpo unlinked", "id", id, "ou", req.OUDN, "actor", sess.Username)
	LogAudit(sess.Username, "gpo.unlink", req.OUDN, "gpo", "", true, "gpo="+id)
	respondJSON(w, http.StatusOK, map[string]any{"success": true, "id": id, "ouDn": req.OUDN})
}

// handleGetGPOLinks returns the GPO links for a given OU.
// GET /api/gpo/links/{ou}
func handleGetGPOLinks(w http.ResponseWriter, r *http.Request) {
	ou := r.PathValue("ou")
	if ou == "" {
		respondError(w, http.StatusBadRequest, "OU DN required")
		return
	}

	output, err := runDNSCommand("gpo", "getlink", ou)
	if err != nil {
		if strings.Contains(err.Error(), "No GPO(s) linked") {
			respondJSON(w, http.StatusOK, map[string]any{"links": []models.GPOLink{}, "ou": ou})
			return
		}
		slog.Error("gpo: getlink failed", "ou", ou, "error", err)
		respondError(w, http.StatusInternalServerError, "failed to get GPO links: "+err.Error())
		return
	}

	links := parseGPOGetLink(output, ou)
	respondJSON(w, http.StatusOK, map[string]any{
		"links": links,
		"ou":    ou,
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// GPO Output Parsers
// ──────────────────────────────────────────────────────────────────────────────

// samba-tool gpo listall output format:
//
//	GPO          : {31B2F340-016D-11D2-945F-00C04FB984F9}
//	display name : Default Domain Policy
//	path         : \\example.com\sysvol\example.com\Policies\{31B2F340-016D-11D2-945F-00C04FB984F9}
//	dn           : CN={31B2F340-016D-11D2-945F-00C04FB984F9},CN=Policies,CN=System,DC=example,DC=com
//	version      : 65539
//	flags        : NONE
func parseGPOListAll(output string) []models.GPO {
	var gpos []models.GPO
	var current *models.GPO

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			if current != nil {
				gpos = append(gpos, *current)
				current = nil
			}
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch strings.ToLower(key) {
		case "gpo":
			current = &models.GPO{ID: val}
		case "display name":
			if current != nil {
				current.Name = val
			}
		case "path":
			if current != nil {
				current.Path = val
			}
		case "dn":
			if current != nil {
				current.DN = val
			}
		case "version":
			if current != nil {
				v, _ := strconv.Atoi(val)
				current.Version = v
			}
		case "flags":
			if current != nil {
				current.Flags = parseGPOFlags(val)
			}
		}
	}
	if current != nil {
		gpos = append(gpos, *current)
	}
	return gpos
}

// parseGPOShow parses `samba-tool gpo show` output (same format as listall for one GPO).
func parseGPOShow(output string) models.GPO {
	gpos := parseGPOListAll(output)
	if len(gpos) > 0 {
		return gpos[0]
	}
	return models.GPO{}
}

// parseGPOFlags converts flag string to int.
func parseGPOFlags(s string) int {
	s = strings.ToUpper(strings.TrimSpace(s))
	if s == "NONE" || s == "" {
		return 0
	}
	v, _ := strconv.Atoi(s)
	return v
}

// parseGPOGetLink parses `samba-tool gpo getlink` output.
// Output format:
//
//	GPO(s) linked to DN OU=Staff,DC=example,DC=com
//	    GPO     : {31B2F340-016D-11D2-945F-00C04FB984F9}
//	    Name    : Default Domain Policy
func parseGPOGetLink(output, ouDN string) []models.GPOLink {
	var links []models.GPOLink
	var currentID string

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch strings.ToLower(key) {
		case "gpo":
			currentID = val
			links = append(links, models.GPOLink{
				GPOID:   currentID,
				OUDN:    ouDN,
				Enabled: true,
			})
		}
	}
	return links
}

// extractGPOID extracts a GPO GUID from samba-tool output.
var gpoGUIDRe = regexp.MustCompile(`\{[0-9A-Fa-f-]{36}\}`)

func extractGPOID(output string) string {
	m := gpoGUIDRe.FindString(output)
	return m
}
