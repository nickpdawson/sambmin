package handlers

import (
	"crypto/subtle"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/nickdawson/sambmin/internal/validate"
)

// --- Self-Service Password Change ---

type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

func handleSelfPasswordChange(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.CurrentPassword == "" || req.NewPassword == "" {
		respondError(w, http.StatusBadRequest, "current and new password required")
		return
	}
	if err := validate.Password(req.NewPassword); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Verify current password matches the session password
	sessionPW, err := sessionStore.Password(sess)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "session credentials unavailable")
		return
	}
	if subtle.ConstantTimeCompare([]byte(req.CurrentPassword), []byte(sessionPW)) != 1 {
		respondError(w, http.StatusForbidden, "current password is incorrect")
		return
	}

	// Use samba-tool to change the user's own password
	args := []string{"user", "setpassword", sess.Username, "--newpassword=" + req.NewPassword}

	if _, err := runSambaTool(r.Context(), sess, args...); err != nil {
		slog.Error("self password change failed", "username", sess.Username, "error", err)
		respondSafeError(w, http.StatusInternalServerError, "password change failed", err)
		return
	}

	// Update the session with the new password so subsequent write ops work
	newSess, err := sessionStore.Create(sess.Username, sess.DN, sess.Groups, req.NewPassword)
	if err != nil {
		slog.Error("session refresh after password change failed", "error", err)
		// Password was changed but session couldn't be refreshed — user will need to re-login
		respondJSON(w, http.StatusOK, map[string]any{"success": true, "reloginRequired": true})
		return
	}

	// Delete old session
	sessionStore.Delete(sess.ID)

	// Set new session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    newSess.ID,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Expires:  newSess.Expires,
	})

	// Update CSRF cookie for new session
	http.SetCookie(w, &http.Cookie{
		Name:     "sambmin_csrf",
		Value:    newSess.CSRFToken,
		Path:     "/",
		HttpOnly: false,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Expires:  newSess.Expires,
	})

	slog.Info("self password changed", "username", sess.Username)
	respondJSON(w, http.StatusOK, map[string]any{"success": true})
}

// --- Self-Service Profile View ---

func handleSelfProfile(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	if dirClient == nil {
		respondError(w, http.StatusServiceUnavailable, "directory not available")
		return
	}

	user, err := dirClient.GetUser(r.Context(), sess.DN)
	if err != nil {
		slog.Error("self profile lookup failed", "dn", sess.DN, "error", err)
		respondError(w, http.StatusInternalServerError, "failed to retrieve profile")
		return
	}

	respondJSON(w, http.StatusOK, user)
}

// --- Self-Service Profile Update ---

type selfUpdateRequest struct {
	Phone      string `json:"phone"`
	Mobile     string `json:"mobile"`
	Department string `json:"department"`
	Title      string `json:"title"`
	Office     string `json:"office"`
}

func handleSelfProfileUpdate(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	if dirClient == nil {
		respondError(w, http.StatusServiceUnavailable, "directory not available")
		return
	}

	var req selfUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	attrs := make(map[string]string)
	if req.Phone != "" {
		attrs["telephoneNumber"] = req.Phone
	}
	if req.Mobile != "" {
		attrs["mobile"] = req.Mobile
	}
	if req.Department != "" {
		attrs["department"] = req.Department
	}
	if req.Title != "" {
		attrs["title"] = req.Title
	}
	if req.Office != "" {
		attrs["physicalDeliveryOfficeName"] = req.Office
	}

	if len(attrs) == 0 {
		respondError(w, http.StatusBadRequest, "no fields to update")
		return
	}

	password, err := sessionStore.Password(sess)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "session credentials unavailable")
		return
	}

	if err := dirClient.ModifyAttributes(r.Context(), sess.DN, attrs, sess.DN, password); err != nil {
		slog.Error("self profile update failed", "dn", sess.DN, "error", err)
		respondSafeError(w, http.StatusInternalServerError, "profile update failed", err)
		return
	}

	slog.Info("self profile updated", "username", sess.Username, "attrs", len(attrs))
	respondJSON(w, http.StatusOK, map[string]any{"success": true})
}
