package handlers

import (
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
