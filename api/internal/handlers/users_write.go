package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/nickdawson/sambmin/internal/auth"
)

// sambaTool is the path to the samba-tool binary.
var sambaTool = "samba-tool"

// sambaToolTimeout is the maximum duration for any samba-tool invocation.
const sambaToolTimeout = 15 * time.Second

// runSambaTool executes a samba-tool command with the given user's credentials.
func runSambaTool(ctx context.Context, sess *auth.Session, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, sambaToolTimeout)
	defer cancel()

	// Get the user's password from the session store
	password, err := sessionStore.Password(sess)
	if err != nil {
		return "", fmt.Errorf("get session credentials: %w", err)
	}

	// Force remote LDAP connection (avoids local LDB permission issues) and append credentials
	args = append(args, "-H", "ldap://localhost", "-U", fmt.Sprintf("%s%%%s", sess.Username, password))

	cmd := exec.CommandContext(ctx, sambaTool, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Log without credentials
	slog.Debug("samba-tool", "args", args[:len(args)-2])

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = err.Error()
		}
		// Extract the meaningful error line (last non-empty line, skip warnings)
		lines := strings.Split(errMsg, "\n")
		for i := len(lines) - 1; i >= 0; i-- {
			line := strings.TrimSpace(lines[i])
			if line != "" && !strings.HasPrefix(line, "WARNING:") && !strings.HasPrefix(line, "Usage:") {
				errMsg = line
				break
			}
		}
		return "", fmt.Errorf("%s", errMsg)
	}

	return stdout.String(), nil
}

// requireSession extracts the session from the request. Returns nil and sends
// a 401 response if no valid session is found.
func requireSession(w http.ResponseWriter, r *http.Request) *auth.Session {
	sess := SessionFromRequest(r)
	if sess == nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return nil
	}
	return sess
}

// --- User Create ---

type createUserRequest struct {
	Username           string `json:"username"`
	Password           string `json:"password"`
	GivenName          string `json:"givenName"`
	Surname            string `json:"surname"`
	Mail               string `json:"mail"`
	Department         string `json:"department"`
	Title              string `json:"title"`
	OU                 string `json:"ou"`
	MustChangePassword bool   `json:"mustChangePassword"`
}

func handleCreateUser(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Username == "" || req.Password == "" {
		respondError(w, http.StatusBadRequest, "username and password required")
		return
	}

	args := []string{"user", "create", req.Username, req.Password}
	if req.GivenName != "" {
		args = append(args, "--given-name", req.GivenName)
	}
	if req.Surname != "" {
		args = append(args, "--surname", req.Surname)
	}
	if req.Mail != "" {
		args = append(args, "--mail-address", req.Mail)
	}
	if req.Department != "" {
		args = append(args, "--department", req.Department)
	}
	if req.Title != "" {
		args = append(args, "--job-title", req.Title)
	}
	if req.OU != "" {
		args = append(args, "--userou", req.OU)
	}
	if req.MustChangePassword {
		args = append(args, "--must-change-at-next-login")
	}

	if _, err := runSambaTool(r.Context(), sess, args...); err != nil {
		slog.Error("user create failed", "username", req.Username, "actor", sess.Username, "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	slog.Info("user created", "username", req.Username, "actor", sess.Username)
	respondJSON(w, http.StatusCreated, map[string]any{
		"success":  true,
		"username": req.Username,
	})
}

// --- User Update ---

type updateUserRequest struct {
	DisplayName string `json:"displayName"`
	GivenName   string `json:"givenName"`
	Surname     string `json:"surname"`
	Mail        string `json:"mail"`
	Department  string `json:"department"`
	Title       string `json:"title"`
	Company     string `json:"company"`
	Description string `json:"description"`
	Phone       string `json:"phone"`
}

func handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	dn := r.PathValue("dn")
	if dn == "" {
		respondError(w, http.StatusBadRequest, "user DN required")
		return
	}

	var req updateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Use LDAP modify for attribute updates (faster than samba-tool)
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
	if req.Department != "" {
		attrs["department"] = req.Department
	}
	if req.Title != "" {
		attrs["title"] = req.Title
	}
	if req.Company != "" {
		attrs["company"] = req.Company
	}
	if req.Description != "" {
		attrs["description"] = req.Description
	}
	if req.Phone != "" {
		attrs["telephoneNumber"] = req.Phone
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
		slog.Error("user update failed", "dn", dn, "actor", sess.Username, "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	slog.Info("user updated", "dn", dn, "actor", sess.Username, "attrs", len(attrs))
	respondJSON(w, http.StatusOK, map[string]any{"success": true})
}

// --- User Delete ---

func handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	dn := r.PathValue("dn")
	if dn == "" {
		respondError(w, http.StatusBadRequest, "user DN required")
		return
	}

	username, err := samAccountNameFromDN(r.Context(), dn)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if _, err := runSambaTool(r.Context(), sess, "user", "delete", username); err != nil {
		slog.Error("user delete failed", "username", username, "actor", sess.Username, "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	slog.Info("user deleted", "username", username, "actor", sess.Username)
	respondJSON(w, http.StatusOK, map[string]any{"success": true, "username": username})
}

// --- Password Reset ---

type resetPasswordRequest struct {
	Password              string `json:"password"`
	MustChangeAtNextLogin bool   `json:"mustChangeAtNextLogin"`
}

func handleResetPassword(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	dn := r.PathValue("dn")
	username, err := samAccountNameFromDN(r.Context(), dn)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	var req resetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Password == "" {
		respondError(w, http.StatusBadRequest, "password required")
		return
	}

	args := []string{"user", "setpassword", username, "--newpassword=" + req.Password}
	if req.MustChangeAtNextLogin {
		args = append(args, "--must-change-at-next-login")
	}

	if _, err := runSambaTool(r.Context(), sess, args...); err != nil {
		slog.Error("password reset failed", "username", username, "actor", sess.Username, "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	slog.Info("password reset", "username", username, "actor", sess.Username)
	respondJSON(w, http.StatusOK, map[string]any{"success": true, "username": username})
}

// --- Enable / Disable / Unlock ---

func handleEnableUser(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	dn := r.PathValue("dn")
	username, err := samAccountNameFromDN(r.Context(), dn)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if _, err := runSambaTool(r.Context(), sess, "user", "enable", username); err != nil {
		slog.Error("user enable failed", "username", username, "actor", sess.Username, "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	slog.Info("user enabled", "username", username, "actor", sess.Username)
	respondJSON(w, http.StatusOK, map[string]any{"success": true, "username": username})
}

func handleDisableUser(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	dn := r.PathValue("dn")
	username, err := samAccountNameFromDN(r.Context(), dn)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if _, err := runSambaTool(r.Context(), sess, "user", "disable", username); err != nil {
		slog.Error("user disable failed", "username", username, "actor", sess.Username, "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	slog.Info("user disabled", "username", username, "actor", sess.Username)
	respondJSON(w, http.StatusOK, map[string]any{"success": true, "username": username})
}

func handleUnlockUser(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	dn := r.PathValue("dn")
	username, err := samAccountNameFromDN(r.Context(), dn)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if _, err := runSambaTool(r.Context(), sess, "user", "unlock", username); err != nil {
		slog.Error("user unlock failed", "username", username, "actor", sess.Username, "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	slog.Info("user unlocked", "username", username, "actor", sess.Username)
	respondJSON(w, http.StatusOK, map[string]any{"success": true, "username": username})
}

// cnFromDN extracts the CN value from a distinguished name.
// e.g., "CN=jdoe,CN=Users,DC=dzsec,DC=net" → "jdoe"
func cnFromDN(dn string) string {
	parts := strings.Split(dn, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(strings.ToUpper(part), "CN=") {
			return part[3:]
		}
	}
	return ""
}

// samAccountNameFromDN looks up the sAMAccountName for a DN via LDAP.
// Falls back to extracting CN from DN if LDAP is unavailable.
func samAccountNameFromDN(ctx context.Context, dn string) (string, error) {
	if dirClient != nil {
		sam, err := dirClient.GetSamAccountName(ctx, dn)
		if err == nil && sam != "" {
			return sam, nil
		}
		slog.Warn("LDAP lookup for sAMAccountName failed, falling back to CN", "dn", dn, "error", err)
	}
	// Fallback: extract CN (works when CN == sAMAccountName)
	parts := strings.Split(dn, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(strings.ToUpper(part), "CN=") {
			return part[3:], nil
		}
	}
	return "", fmt.Errorf("could not determine username from DN: %s", dn)
}
