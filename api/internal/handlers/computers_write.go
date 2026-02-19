package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

func handleDeleteComputer(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	dn := r.PathValue("dn")
	computerName := cnFromDN(dn)
	if computerName == "" {
		respondError(w, http.StatusBadRequest, "could not extract computer name from DN")
		return
	}

	if dirClient == nil {
		respondError(w, http.StatusServiceUnavailable, "directory not available")
		return
	}

	password, err := sessionStore.Password(sess)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "session credentials unavailable")
		return
	}

	if err := dirClient.DeleteObject(r.Context(), dn, sess.DN, password); err != nil {
		slog.Error("computer delete failed", "name", computerName, "actor", sess.Username, "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	slog.Info("computer deleted", "name", computerName, "actor", sess.Username)
	respondJSON(w, http.StatusOK, map[string]any{"success": true, "name": computerName})
}

// --- Computer Create ---

type createComputerRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	OU          string `json:"ou"`
}

func handleCreateComputer(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	var req createComputerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "computer name required")
		return
	}

	args := []string{"computer", "create", req.Name}
	if req.Description != "" {
		args = append(args, "--description", req.Description)
	}
	if req.OU != "" {
		args = append(args, "--computerou", req.OU)
	}

	if _, err := runSambaTool(r.Context(), sess, args...); err != nil {
		slog.Error("computer create failed", "name", req.Name, "actor", sess.Username, "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	slog.Info("computer created", "name", req.Name, "actor", sess.Username)
	respondJSON(w, http.StatusCreated, map[string]any{"success": true, "name": req.Name})
}

// --- Computer Move ---

func handleMoveComputer(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	dn := r.PathValue("dn")
	computerName := cnFromDN(dn)
	if computerName == "" {
		respondError(w, http.StatusBadRequest, "could not extract computer name from DN")
		return
	}

	var req moveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.TargetOU == "" {
		respondError(w, http.StatusBadRequest, "target OU required")
		return
	}

	if _, err := runSambaTool(r.Context(), sess, "computer", "move", computerName, req.TargetOU); err != nil {
		slog.Error("computer move failed", "name", computerName, "targetOU", req.TargetOU, "actor", sess.Username, "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	slog.Info("computer moved", "name", computerName, "targetOU", req.TargetOU, "actor", sess.Username)
	respondJSON(w, http.StatusOK, map[string]any{"success": true, "name": computerName})
}
