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
	"github.com/nickdawson/sambmin/internal/validate"
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

	// DRS and domain commands use DCE/RPC, not LDAP — skip -H flag for them
	if len(args) > 0 && args[0] != "drs" && args[0] != "domain" {
		args = append(args, "-H", "ldap://localhost")
	}
	args = append(args, "-U", fmt.Sprintf("%s%%%s", sess.Username, password))

	cmd := exec.CommandContext(ctx, sambaTool, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Log without credentials
	slog.Debug("samba-tool", "args", filterSensitiveArgs(args))

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = err.Error()
		}
		// Extract the meaningful error line (last non-empty line, skip noise)
		lines := strings.Split(errMsg, "\n")
		for i := len(lines) - 1; i >= 0; i-- {
			line := strings.TrimSpace(lines[i])
			if line == "" {
				continue
			}
			// Skip Python traceback noise, warnings, usage hints
			if strings.HasPrefix(line, "WARNING:") || strings.HasPrefix(line, "Usage:") {
				continue
			}
			if strings.HasPrefix(line, "File ") || strings.HasPrefix(line, "^^^^") || strings.Trim(line, "^ ") == "" {
				continue
			}
			errMsg = line
			break
		}
		return "", fmt.Errorf("%s", errMsg)
	}

	return stdout.String(), nil
}

// filterSensitiveArgs returns a copy of args with sensitive values redacted.
// Strips: -U user%password, --newpassword=..., --password=..., and
// the positional password in "user create <username> <password>".
func filterSensitiveArgs(args []string) []string {
	filtered := make([]string, 0, len(args))
	skipNext := false
	for i, a := range args {
		if skipNext {
			skipNext = false
			continue
		}
		// Skip -U and its value
		if a == "-U" {
			skipNext = true
			continue
		}
		// Redact password flags
		if strings.HasPrefix(a, "--newpassword=") || strings.HasPrefix(a, "--password=") {
			filtered = append(filtered, strings.SplitN(a, "=", 2)[0]+"=***")
			continue
		}
		// Redact positional password in "user create <username> <password>"
		if i == 3 && len(args) > 1 && args[0] == "user" && args[1] == "create" {
			filtered = append(filtered, "***")
			continue
		}
		// Skip args containing % (likely user%password)
		if strings.Contains(a, "%") && i > 0 && args[i-1] == "-U" {
			continue
		}
		filtered = append(filtered, a)
	}
	return filtered
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
	if err := validate.SAMAccountName(req.Username); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validate.Password(req.Password); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
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
		respondSafeError(w, http.StatusInternalServerError, "user creation failed", err)
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
	Mobile      string `json:"mobile"`
	Office      string `json:"office"`
	Street      string `json:"streetAddress"`
	City        string `json:"city"`
	State       string `json:"state"`
	PostalCode  string `json:"postalCode"`
	Country     string `json:"country"`
	// Windows profile
	ProfilePath   string `json:"profilePath"`
	ScriptPath    string `json:"scriptPath"`
	HomeDrive     string `json:"homeDrive"`
	HomeDirectory string `json:"homeDirectory"`
	// Unix/POSIX
	LoginShell    string `json:"loginShell"`
	UnixHomeDir   string `json:"unixHomeDirectory"`
	UidNumber     string `json:"uidNumber"`
	GidNumber     string `json:"gidNumber"`
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
	if req.ProfilePath != "" {
		attrs["profilePath"] = req.ProfilePath
	}
	if req.ScriptPath != "" {
		attrs["scriptPath"] = req.ScriptPath
	}
	if req.HomeDrive != "" {
		attrs["homeDrive"] = req.HomeDrive
	}
	if req.HomeDirectory != "" {
		attrs["homeDirectory"] = req.HomeDirectory
	}
	if req.LoginShell != "" {
		attrs["loginShell"] = req.LoginShell
	}
	if req.UnixHomeDir != "" {
		attrs["unixHomeDirectory"] = req.UnixHomeDir
	}
	if req.UidNumber != "" {
		attrs["uidNumber"] = req.UidNumber
	}
	if req.GidNumber != "" {
		attrs["gidNumber"] = req.GidNumber
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
		respondSafeError(w, http.StatusInternalServerError, "user update failed", err)
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
		respondSafeError(w, http.StatusInternalServerError, "user deletion failed", err)
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
	if err := validate.Password(req.Password); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	args := []string{"user", "setpassword", username, "--newpassword=" + req.Password}
	if req.MustChangeAtNextLogin {
		args = append(args, "--must-change-at-next-login")
	}

	if _, err := runSambaTool(r.Context(), sess, args...); err != nil {
		slog.Error("password reset failed", "username", username, "actor", sess.Username, "error", err)
		respondSafeError(w, http.StatusInternalServerError, "password reset failed", err)
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
		respondSafeError(w, http.StatusInternalServerError, "user enable failed", err)
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
		respondSafeError(w, http.StatusInternalServerError, "user disable failed", err)
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
		respondSafeError(w, http.StatusInternalServerError, "user unlock failed", err)
		return
	}

	slog.Info("user unlocked", "username", username, "actor", sess.Username)
	respondJSON(w, http.StatusOK, map[string]any{"success": true, "username": username})
}

// --- User Rename ---

type renameUserRequest struct {
	NewName      string `json:"newName"`
	NewSurname   string `json:"newSurname"`
	NewGivenName string `json:"newGivenName"`
}

func handleRenameUser(w http.ResponseWriter, r *http.Request) {
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

	var req renameUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.NewName == "" {
		respondError(w, http.StatusBadRequest, "new name required")
		return
	}
	if err := validate.NoFlagInjection(req.NewName, "new name"); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	args := []string{"user", "rename", username, "--new-cn=" + req.NewName}
	if req.NewSurname != "" {
		args = append(args, "--surname=" + req.NewSurname)
	}
	if req.NewGivenName != "" {
		args = append(args, "--given-name=" + req.NewGivenName)
	}

	if _, err := runSambaTool(r.Context(), sess, args...); err != nil {
		slog.Error("user rename failed", "username", username, "newName", req.NewName, "actor", sess.Username, "error", err)
		respondSafeError(w, http.StatusInternalServerError, "user rename failed", err)
		return
	}

	slog.Info("user renamed", "username", username, "newName", req.NewName, "actor", sess.Username)
	respondJSON(w, http.StatusOK, map[string]any{"success": true, "oldName": username, "newName": req.NewName})
}

// cnFromDN extracts the CN value from a distinguished name.
// e.g., "CN=jdoe,CN=Users,DC=example,DC=com" → "jdoe"
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
