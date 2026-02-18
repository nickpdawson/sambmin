package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type createOURequest struct {
	Name        string `json:"name"`
	ParentDN    string `json:"parentDn"`
	Description string `json:"description"`
}

func handleCreateOU(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	var req createOURequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "OU name required")
		return
	}

	// Build the OU DN
	ouDN := "OU=" + req.Name
	if req.ParentDN != "" {
		ouDN += "," + req.ParentDN
	} else if handlerConfig != nil {
		ouDN += "," + handlerConfig.BaseDN
	}

	args := []string{"ou", "create", ouDN}
	if req.Description != "" {
		args = append(args, "--description", req.Description)
	}

	if _, err := runSambaTool(r.Context(), sess, args...); err != nil {
		slog.Error("OU create failed", "name", req.Name, "actor", sess.Username, "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	slog.Info("OU created", "name", req.Name, "dn", ouDN, "actor", sess.Username)
	respondJSON(w, http.StatusCreated, map[string]any{"success": true, "name": req.Name, "dn": ouDN})
}

func handleDeleteOU(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	dn := r.PathValue("dn")
	if dn == "" {
		respondError(w, http.StatusBadRequest, "OU DN required")
		return
	}

	if _, err := runSambaTool(r.Context(), sess, "ou", "delete", dn); err != nil {
		slog.Error("OU delete failed", "dn", dn, "actor", sess.Username, "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	slog.Info("OU deleted", "dn", dn, "actor", sess.Username)
	respondJSON(w, http.StatusOK, map[string]any{"success": true})
}
