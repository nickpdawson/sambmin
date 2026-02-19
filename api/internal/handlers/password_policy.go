package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/nickdawson/sambmin/internal/models"
)

// --- Domain Password Policy ---

// handleGetPasswordPolicy retrieves the domain default password policy via samba-tool.
func handleGetPasswordPolicy(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	output, err := runSambaTool(r.Context(), sess, "domain", "passwordsettings", "show")
	if err != nil {
		slog.Error("get password policy failed", "actor", sess.Username, "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	policy := parsePasswordPolicy(output)
	respondJSON(w, http.StatusOK, policy)
}

// handleUpdatePasswordPolicy updates the domain default password policy.
func handleUpdatePasswordPolicy(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	var req models.PasswordPolicy
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	args := []string{"domain", "passwordsettings", "set"}

	// Only set fields that were provided (non-zero values)
	if req.MinLength > 0 {
		args = append(args, fmt.Sprintf("--min-pwd-length=%d", req.MinLength))
	}
	if req.HistoryLength >= 0 {
		args = append(args, fmt.Sprintf("--history-length=%d", req.HistoryLength))
	}
	if req.MinAge != "" {
		args = append(args, fmt.Sprintf("--min-pwd-age=%s", req.MinAge))
	}
	if req.MaxAge != "" {
		args = append(args, fmt.Sprintf("--max-pwd-age=%s", req.MaxAge))
	}
	if req.Complexity {
		args = append(args, "--complexity=on")
	} else {
		args = append(args, "--complexity=off")
	}
	if req.StorePlaintext {
		args = append(args, "--store-plaintext=on")
	} else {
		args = append(args, "--store-plaintext=off")
	}
	if req.LockoutThreshold >= 0 {
		args = append(args, fmt.Sprintf("--account-lockout-threshold=%d", req.LockoutThreshold))
	}
	if req.LockoutDuration != "" {
		args = append(args, fmt.Sprintf("--account-lockout-duration=%s", parseDurationToMinutes(req.LockoutDuration)))
	}
	if req.LockoutWindow != "" {
		args = append(args, fmt.Sprintf("--reset-account-lockout-after=%s", parseDurationToMinutes(req.LockoutWindow)))
	}

	if _, err := runSambaTool(r.Context(), sess, args...); err != nil {
		slog.Error("update password policy failed", "actor", sess.Username, "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	slog.Info("password policy updated", "actor", sess.Username)
	respondJSON(w, http.StatusOK, map[string]any{"success": true})
}

// --- PSO (Fine-Grained Password Policy) ---

// handleListPSOs lists all Password Settings Objects.
func handleListPSOs(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	output, err := runSambaTool(r.Context(), sess, "domain", "passwordsettings", "pso", "list")
	if err != nil {
		slog.Error("list PSOs failed", "actor", sess.Username, "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	psos := parsePSOList(output)
	respondJSON(w, http.StatusOK, psos)
}

// handleCreatePSO creates a new PSO.
func handleCreatePSO(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	var req models.PSO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "PSO name required")
		return
	}
	if req.Precedence <= 0 {
		respondError(w, http.StatusBadRequest, "precedence must be > 0")
		return
	}

	args := []string{
		"domain", "passwordsettings", "pso", "create",
		req.Name,
		strconv.Itoa(req.Precedence),
	}
	if req.MinLength > 0 {
		args = append(args, fmt.Sprintf("--min-pwd-length=%d", req.MinLength))
	}
	if req.HistoryLength > 0 {
		args = append(args, fmt.Sprintf("--history-length=%d", req.HistoryLength))
	}
	if req.MinAge != "" {
		args = append(args, fmt.Sprintf("--min-pwd-age=%s", req.MinAge))
	}
	if req.MaxAge != "" {
		args = append(args, fmt.Sprintf("--max-pwd-age=%s", req.MaxAge))
	}
	if req.Complexity {
		args = append(args, "--complexity=on")
	}
	if req.LockoutThreshold > 0 {
		args = append(args, fmt.Sprintf("--account-lockout-threshold=%d", req.LockoutThreshold))
	}
	if req.LockoutDuration != "" {
		args = append(args, fmt.Sprintf("--account-lockout-duration=%s", parseDurationToMinutes(req.LockoutDuration)))
	}
	if req.LockoutWindow != "" {
		args = append(args, fmt.Sprintf("--reset-account-lockout-after=%s", parseDurationToMinutes(req.LockoutWindow)))
	}

	if _, err := runSambaTool(r.Context(), sess, args...); err != nil {
		slog.Error("create PSO failed", "name", req.Name, "actor", sess.Username, "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	slog.Info("PSO created", "name", req.Name, "actor", sess.Username)
	respondJSON(w, http.StatusCreated, map[string]any{"success": true, "name": req.Name})
}

// handleUpdatePSO modifies a PSO.
func handleUpdatePSO(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	name := r.PathValue("name")
	if name == "" {
		respondError(w, http.StatusBadRequest, "PSO name required")
		return
	}

	var req models.PSO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	args := []string{"domain", "passwordsettings", "pso", "set", name}
	if req.MinLength > 0 {
		args = append(args, fmt.Sprintf("--min-pwd-length=%d", req.MinLength))
	}
	if req.HistoryLength > 0 {
		args = append(args, fmt.Sprintf("--history-length=%d", req.HistoryLength))
	}
	if req.Complexity {
		args = append(args, "--complexity=on")
	} else {
		args = append(args, "--complexity=off")
	}
	if req.MinAge != "" {
		args = append(args, fmt.Sprintf("--min-pwd-age=%s", req.MinAge))
	}
	if req.MaxAge != "" {
		args = append(args, fmt.Sprintf("--max-pwd-age=%s", req.MaxAge))
	}

	if _, err := runSambaTool(r.Context(), sess, args...); err != nil {
		slog.Error("update PSO failed", "name", name, "actor", sess.Username, "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	slog.Info("PSO updated", "name", name, "actor", sess.Username)
	respondJSON(w, http.StatusOK, map[string]any{"success": true})
}

// handleDeletePSO deletes a PSO.
func handleDeletePSO(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	name := r.PathValue("name")
	if name == "" {
		respondError(w, http.StatusBadRequest, "PSO name required")
		return
	}

	if _, err := runSambaTool(r.Context(), sess, "domain", "passwordsettings", "pso", "delete", name); err != nil {
		slog.Error("delete PSO failed", "name", name, "actor", sess.Username, "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	slog.Info("PSO deleted", "name", name, "actor", sess.Username)
	respondJSON(w, http.StatusOK, map[string]any{"success": true})
}

// handleApplyPSO applies a PSO to a user or group.
func handleApplyPSO(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	name := r.PathValue("name")
	if name == "" {
		respondError(w, http.StatusBadRequest, "PSO name required")
		return
	}

	var req struct {
		Target string `json:"target"` // sAMAccountName of user or group
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Target == "" {
		respondError(w, http.StatusBadRequest, "target user or group required")
		return
	}

	if _, err := runSambaTool(r.Context(), sess, "domain", "passwordsettings", "pso", "apply", name, req.Target); err != nil {
		slog.Error("apply PSO failed", "pso", name, "target", req.Target, "actor", sess.Username, "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	slog.Info("PSO applied", "pso", name, "target", req.Target, "actor", sess.Username)
	respondJSON(w, http.StatusOK, map[string]any{"success": true})
}

// handleUnapplyPSO removes a PSO from a user or group.
func handleUnapplyPSO(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	name := r.PathValue("name")
	if name == "" {
		respondError(w, http.StatusBadRequest, "PSO name required")
		return
	}

	var req struct {
		Target string `json:"target"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Target == "" {
		respondError(w, http.StatusBadRequest, "target user or group required")
		return
	}

	if _, err := runSambaTool(r.Context(), sess, "domain", "passwordsettings", "pso", "unapply", name, req.Target); err != nil {
		slog.Error("unapply PSO failed", "pso", name, "target", req.Target, "actor", sess.Username, "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	slog.Info("PSO unapplied", "pso", name, "target", req.Target, "actor", sess.Username)
	respondJSON(w, http.StatusOK, map[string]any{"success": true})
}

// handleGetEffectivePolicy gets the effective password policy for a user.
func handleGetEffectivePolicy(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	username := r.PathValue("username")
	if username == "" {
		respondError(w, http.StatusBadRequest, "username required")
		return
	}

	// Try PSO-specific lookup first
	output, err := runSambaTool(r.Context(), sess, "domain", "passwordsettings", "pso", "show-user", username)
	if err != nil {
		// No PSO applies, fall back to domain default
		output, err = runSambaTool(r.Context(), sess, "domain", "passwordsettings", "show")
		if err != nil {
			slog.Error("get effective policy failed", "username", username, "actor", sess.Username, "error", err)
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		policy := parsePasswordPolicy(output)
		respondJSON(w, http.StatusOK, map[string]any{
			"policy": policy,
			"source": "default",
		})
		return
	}

	policy := parsePasswordPolicy(output)
	respondJSON(w, http.StatusOK, map[string]any{
		"policy": policy,
		"source": "pso",
	})
}

// handleTestPassword tests a password against policy rules client-side.
func handleTestPassword(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	var req models.PasswordTestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Password == "" {
		respondError(w, http.StatusBadRequest, "password required")
		return
	}

	// Get the effective policy
	var policyOutput string
	var policySource string
	var err error

	if req.Username != "" {
		policyOutput, err = runSambaTool(r.Context(), sess, "domain", "passwordsettings", "pso", "show-user", req.Username)
		if err != nil {
			policyOutput, err = runSambaTool(r.Context(), sess, "domain", "passwordsettings", "show")
			policySource = "default"
		} else {
			policySource = "pso"
		}
	} else {
		policyOutput, err = runSambaTool(r.Context(), sess, "domain", "passwordsettings", "show")
		policySource = "default"
	}

	if err != nil {
		slog.Error("get policy for test failed", "actor", sess.Username, "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	policy := parsePasswordPolicy(policyOutput)
	result := testPasswordAgainstPolicy(req.Password, req.Username, policy)
	result.Policy = policySource

	respondJSON(w, http.StatusOK, result)
}

// --- Parser helpers ---

// parsePasswordPolicy parses the output of `samba-tool domain passwordsettings show`.
func parsePasswordPolicy(output string) models.PasswordPolicy {
	var p models.PasswordPolicy
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !strings.Contains(line, ":") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch {
		case strings.Contains(strings.ToLower(key), "minimum password length"):
			p.MinLength, _ = strconv.Atoi(val)
		case strings.Contains(strings.ToLower(key), "password history"):
			p.HistoryLength, _ = strconv.Atoi(val)
		case strings.Contains(strings.ToLower(key), "minimum password age"):
			p.MinAge = val
		case strings.Contains(strings.ToLower(key), "maximum password age"):
			p.MaxAge = val
		case strings.Contains(strings.ToLower(key), "password complexity"):
			p.Complexity = strings.ToLower(val) == "on"
		case strings.Contains(strings.ToLower(key), "store plaintext"):
			p.StorePlaintext = strings.ToLower(val) == "on"
		case strings.Contains(strings.ToLower(key), "lockout threshold"):
			p.LockoutThreshold, _ = strconv.Atoi(val)
		case strings.Contains(strings.ToLower(key), "lockout duration"):
			p.LockoutDuration = val
		case strings.Contains(strings.ToLower(key), "reset account lockout"):
			p.LockoutWindow = val
		}
	}
	return p
}

// parsePSOList parses the output of `samba-tool domain passwordsettings pso list`.
func parsePSOList(output string) []models.PSO {
	var psos []models.PSO
	// Each PSO is listed as a line with the name
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Password") {
			continue
		}
		psos = append(psos, models.PSO{Name: line})
	}
	return psos
}

// parseDurationToMinutes converts duration strings like "30 minutes" to just the number.
func parseDurationToMinutes(s string) string {
	re := regexp.MustCompile(`(\d+)`)
	match := re.FindString(s)
	if match != "" {
		return match
	}
	return s
}

// testPasswordAgainstPolicy performs client-side password policy validation.
func testPasswordAgainstPolicy(password, username string, policy models.PasswordPolicy) models.PasswordTestResult {
	var errors []string

	// Check minimum length
	if len(password) < policy.MinLength {
		errors = append(errors, fmt.Sprintf("Password must be at least %d characters (currently %d)", policy.MinLength, len(password)))
	}

	// Check complexity (if enabled)
	if policy.Complexity {
		categories := 0
		hasUpper := false
		hasLower := false
		hasDigit := false
		hasSpecial := false

		for _, ch := range password {
			switch {
			case unicode.IsUpper(ch):
				hasUpper = true
			case unicode.IsLower(ch):
				hasLower = true
			case unicode.IsDigit(ch):
				hasDigit = true
			case unicode.IsPunct(ch) || unicode.IsSymbol(ch) || unicode.IsSpace(ch):
				hasSpecial = true
			}
		}
		if hasUpper {
			categories++
		}
		if hasLower {
			categories++
		}
		if hasDigit {
			categories++
		}
		if hasSpecial {
			categories++
		}

		if categories < 3 {
			errors = append(errors, "Password must contain characters from at least 3 of: uppercase, lowercase, digits, special characters")
		}

		// Check if password contains username
		if username != "" && len(username) >= 3 {
			if strings.Contains(strings.ToLower(password), strings.ToLower(username)) {
				errors = append(errors, "Password must not contain the username")
			}
		}
	}

	return models.PasswordTestResult{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
}
