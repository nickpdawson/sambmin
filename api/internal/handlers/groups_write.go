package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// --- Group Create ---

type createGroupRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	GroupType   string `json:"groupType"` // "Security" or "Distribution"
	GroupScope  string `json:"groupScope"` // "Global", "DomainLocal", "Universal"
	OU          string `json:"ou"`
}

func handleCreateGroup(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	var req createGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "group name required")
		return
	}

	args := []string{"group", "add", req.Name}
	if req.Description != "" {
		args = append(args, "--description", req.Description)
	}
	if req.GroupType != "" {
		args = append(args, "--group-type", req.GroupType)
	}
	if req.OU != "" {
		args = append(args, "--groupou", req.OU)
	}

	if _, err := runSambaTool(r.Context(), sess, args...); err != nil {
		slog.Error("group create failed", "name", req.Name, "actor", sess.Username, "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	slog.Info("group created", "name", req.Name, "actor", sess.Username)
	respondJSON(w, http.StatusCreated, map[string]any{"success": true, "name": req.Name})
}

// --- Group Update ---

type updateGroupRequest struct {
	Description string `json:"description"`
}

func handleUpdateGroup(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	dn := r.PathValue("dn")
	if dn == "" {
		respondError(w, http.StatusBadRequest, "group DN required")
		return
	}

	var req updateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if dirClient == nil {
		respondError(w, http.StatusServiceUnavailable, "directory not available")
		return
	}

	attrs := make(map[string]string)
	if req.Description != "" {
		attrs["description"] = req.Description
	}

	if len(attrs) == 0 {
		respondError(w, http.StatusBadRequest, "no attributes to update")
		return
	}

	password, err := sessionStore.Password(sess)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "session credentials unavailable")
		return
	}

	if err := dirClient.ModifyAttributes(r.Context(), dn, attrs, sess.DN, password); err != nil {
		slog.Error("group update failed", "dn", dn, "actor", sess.Username, "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	slog.Info("group updated", "dn", dn, "actor", sess.Username)
	respondJSON(w, http.StatusOK, map[string]any{"success": true})
}

// --- Group Delete ---

func handleDeleteGroup(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	dn := r.PathValue("dn")
	groupName := cnFromDN(dn)
	if groupName == "" {
		respondError(w, http.StatusBadRequest, "could not extract group name from DN")
		return
	}

	if _, err := runSambaTool(r.Context(), sess, "group", "delete", groupName); err != nil {
		slog.Error("group delete failed", "name", groupName, "actor", sess.Username, "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	slog.Info("group deleted", "name", groupName, "actor", sess.Username)
	respondJSON(w, http.StatusOK, map[string]any{"success": true, "name": groupName})
}

// --- Group Membership ---

type memberRequest struct {
	MemberDN string `json:"memberDn"`
}

func handleAddGroupMember(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	dn := r.PathValue("dn")
	groupName := cnFromDN(dn)
	if groupName == "" {
		respondError(w, http.StatusBadRequest, "could not extract group name from DN")
		return
	}

	var req memberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	memberName := cnFromDN(req.MemberDN)
	if memberName == "" {
		respondError(w, http.StatusBadRequest, "could not extract member name from DN")
		return
	}

	if _, err := runSambaTool(r.Context(), sess, "group", "addmembers", groupName, memberName); err != nil {
		slog.Error("add group member failed", "group", groupName, "member", memberName, "actor", sess.Username, "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	slog.Info("group member added", "group", groupName, "member", memberName, "actor", sess.Username)
	respondJSON(w, http.StatusOK, map[string]any{"success": true})
}

func handleRemoveGroupMember(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	dn := r.PathValue("dn")
	groupName := cnFromDN(dn)
	if groupName == "" {
		respondError(w, http.StatusBadRequest, "could not extract group name from DN")
		return
	}

	memberDN := r.PathValue("memberDn")
	memberName := cnFromDN(memberDN)
	if memberName == "" {
		respondError(w, http.StatusBadRequest, "could not extract member name from DN")
		return
	}

	if _, err := runSambaTool(r.Context(), sess, "group", "removemembers", groupName, memberName); err != nil {
		slog.Error("remove group member failed", "group", groupName, "member", memberName, "actor", sess.Username, "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	slog.Info("group member removed", "group", groupName, "member", memberName, "actor", sess.Username)
	respondJSON(w, http.StatusOK, map[string]any{"success": true})
}
