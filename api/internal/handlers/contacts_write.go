package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// --- Contact Create ---

type createContactRequest struct {
	Name        string `json:"name"`
	GivenName   string `json:"givenName"`
	Surname     string `json:"surname"`
	Mail        string `json:"mail"`
	Description string `json:"description"`
	OU          string `json:"ou"`
}

func handleCreateContact(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	var req createContactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "contact name required")
		return
	}

	args := []string{"contact", "create", req.Name}
	if req.GivenName != "" {
		args = append(args, "--given-name", req.GivenName)
	}
	if req.Surname != "" {
		args = append(args, "--surname", req.Surname)
	}
	if req.Mail != "" {
		args = append(args, "--mail-address", req.Mail)
	}
	if req.Description != "" {
		args = append(args, "--description", req.Description)
	}
	if req.OU != "" {
		args = append(args, "--ou", req.OU)
	}

	if _, err := runSambaTool(r.Context(), sess, args...); err != nil {
		slog.Error("contact create failed", "name", req.Name, "actor", sess.Username, "error", err)
		respondSafeError(w, http.StatusInternalServerError, "contact creation failed", err)
		return
	}

	slog.Info("contact created", "name", req.Name, "actor", sess.Username)
	respondJSON(w, http.StatusCreated, map[string]any{"success": true, "name": req.Name})
}

// --- Contact Update ---

type updateContactRequest struct {
	DisplayName string `json:"displayName"`
	GivenName   string `json:"givenName"`
	Surname     string `json:"surname"`
	Mail        string `json:"mail"`
	Description string `json:"description"`
	Department  string `json:"department"`
	Title       string `json:"title"`
	Company     string `json:"company"`
	Phone       string `json:"phone"`
	Mobile      string `json:"mobile"`
	Office      string `json:"office"`
	Street      string `json:"streetAddress"`
	City        string `json:"city"`
	State       string `json:"state"`
	PostalCode  string `json:"postalCode"`
	Country     string `json:"country"`
}

func handleUpdateContact(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	dn := r.PathValue("dn")
	if dn == "" {
		respondError(w, http.StatusBadRequest, "contact DN required")
		return
	}

	var req updateContactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if dirClient == nil {
		respondError(w, http.StatusServiceUnavailable, "directory not available")
		return
	}

	attrs := make(map[string]string)
	if req.DisplayName != "" {
		attrs["displayName"] = req.DisplayName
	}
	if req.GivenName != "" {
		attrs["givenName"] = req.GivenName
	}
	if req.Surname != "" {
		attrs["sn"] = req.Surname
	}
	if req.Mail != "" {
		attrs["mail"] = req.Mail
	}
	if req.Description != "" {
		attrs["description"] = req.Description
	}
	if req.Department != "" {
		attrs["department"] = req.Department
	}
	if req.Title != "" {
		attrs["title"] = req.Title
	}
	if req.Company != "" {
		attrs["company"] = req.Company
	}
	if req.Phone != "" {
		attrs["telephoneNumber"] = req.Phone
	}
	if req.Mobile != "" {
		attrs["mobile"] = req.Mobile
	}
	if req.Office != "" {
		attrs["physicalDeliveryOfficeName"] = req.Office
	}
	if req.Street != "" {
		attrs["streetAddress"] = req.Street
	}
	if req.City != "" {
		attrs["l"] = req.City
	}
	if req.State != "" {
		attrs["st"] = req.State
	}
	if req.PostalCode != "" {
		attrs["postalCode"] = req.PostalCode
	}
	if req.Country != "" {
		attrs["co"] = req.Country
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
		slog.Error("contact update failed", "dn", dn, "actor", sess.Username, "error", err)
		respondSafeError(w, http.StatusInternalServerError, "contact update failed", err)
		return
	}

	slog.Info("contact updated", "dn", dn, "actor", sess.Username, "attrs", len(attrs))
	respondJSON(w, http.StatusOK, map[string]any{"success": true})
}

// --- Contact Delete ---

func handleDeleteContact(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	dn := r.PathValue("dn")
	contactName := cnFromDN(dn)
	if contactName == "" {
		respondError(w, http.StatusBadRequest, "could not extract contact name from DN")
		return
	}

	if _, err := runSambaTool(r.Context(), sess, "contact", "delete", contactName); err != nil {
		slog.Error("contact delete failed", "name", contactName, "actor", sess.Username, "error", err)
		respondSafeError(w, http.StatusInternalServerError, "contact deletion failed", err)
		return
	}

	slog.Info("contact deleted", "name", contactName, "actor", sess.Username)
	respondJSON(w, http.StatusOK, map[string]any{"success": true, "name": contactName})
}

// --- Contact Move ---

type moveRequest struct {
	TargetOU string `json:"targetOu"`
}

func handleMoveContact(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	dn := r.PathValue("dn")
	contactName := cnFromDN(dn)
	if contactName == "" {
		respondError(w, http.StatusBadRequest, "could not extract contact name from DN")
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

	if _, err := runSambaTool(r.Context(), sess, "contact", "move", contactName, req.TargetOU); err != nil {
		slog.Error("contact move failed", "name", contactName, "targetOU", req.TargetOU, "actor", sess.Username, "error", err)
		respondSafeError(w, http.StatusInternalServerError, "contact move failed", err)
		return
	}

	slog.Info("contact moved", "name", contactName, "targetOU", req.TargetOU, "actor", sess.Username)
	respondJSON(w, http.StatusOK, map[string]any{"success": true, "name": contactName})
}

// --- Contact Rename ---

type renameContactRequest struct {
	NewName string `json:"newName"`
}

func handleRenameContact(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	dn := r.PathValue("dn")
	contactName := cnFromDN(dn)
	if contactName == "" {
		respondError(w, http.StatusBadRequest, "could not extract contact name from DN")
		return
	}

	var req renameContactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.NewName == "" {
		respondError(w, http.StatusBadRequest, "new name required")
		return
	}

	if _, err := runSambaTool(r.Context(), sess, "contact", "rename", contactName, "--new-cn="+req.NewName); err != nil {
		slog.Error("contact rename failed", "name", contactName, "newName", req.NewName, "actor", sess.Username, "error", err)
		respondSafeError(w, http.StatusInternalServerError, "contact rename failed", err)
		return
	}

	slog.Info("contact renamed", "oldName", contactName, "newName", req.NewName, "actor", sess.Username)
	respondJSON(w, http.StatusOK, map[string]any{"success": true, "oldName": contactName, "newName": req.NewName})
}
