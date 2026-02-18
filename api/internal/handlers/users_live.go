package handlers

import (
	"net/http"
	"strconv"

	"github.com/nickdawson/sambmin/internal/directory"
)

// handleListUsers queries the live LDAP directory for users.
func handleListUsers(w http.ResponseWriter, r *http.Request) {
	opts := directory.ListUsersOptions{
		Filter: r.URL.Query().Get("filter"),
		Search: r.URL.Query().Get("q"),
	}

	if limit := r.URL.Query().Get("limit"); limit != "" {
		opts.Limit, _ = strconv.Atoi(limit)
	}
	if offset := r.URL.Query().Get("offset"); offset != "" {
		opts.Offset, _ = strconv.Atoi(offset)
	}

	users, total, err := dirClient.ListUsers(r.Context(), opts)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list users: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"users": users,
		"total": total,
	})
}

// handleGetUser queries a single user by DN from the live directory.
func handleGetUser(w http.ResponseWriter, r *http.Request) {
	dn := r.PathValue("dn")
	if dn == "" {
		respondError(w, http.StatusBadRequest, "missing dn parameter")
		return
	}

	user, err := dirClient.GetUser(r.Context(), dn)
	if err != nil {
		respondError(w, http.StatusNotFound, "user not found: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, user)
}

// handleDashboardMetrics returns real counts from the directory.
func handleDashboardMetrics(w http.ResponseWriter, r *http.Request) {
	metrics, err := dirClient.Metrics(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get metrics: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, metrics)
}
