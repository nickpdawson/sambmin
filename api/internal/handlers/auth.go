package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/nickdawson/sambmin/internal/auth"
	"github.com/nickdawson/sambmin/internal/middleware"
)

var (
	sessionStore *auth.Store
	ldapAuth     *auth.LDAPAuthenticator
	loginLimiter *middleware.RateLimiter
)

const sessionCookieName = "sambmin_session"

// InitAuth sets up the session store, LDAP authenticator, and login rate limiter.
// Called from main.go during startup.
func InitAuth(store *auth.Store, authenticator *auth.LDAPAuthenticator) {
	sessionStore = store
	ldapAuth = authenticator
	// 10 failed attempts per IP per minute, 5 per username per 15 minutes
	loginLimiter = middleware.NewRateLimiter(10, time.Minute, 5, 15*time.Minute)
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Username    string   `json:"username"`
	DisplayName string   `json:"displayName"`
	DN          string   `json:"dn"`
	Groups      []string `json:"groups"`
}

func handleLoginImpl(w http.ResponseWriter, r *http.Request) {
	if sessionStore == nil || ldapAuth == nil {
		respondError(w, http.StatusServiceUnavailable, "authentication not configured")
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Username == "" || req.Password == "" {
		respondError(w, http.StatusBadRequest, "username and password required")
		return
	}

	// Rate limit check
	clientIP := middleware.ClientIP(r)
	if loginLimiter != nil {
		if blocked, retryAfter := loginLimiter.Check(clientIP, req.Username); blocked {
			slog.Warn("auth: rate limited", "ip", clientIP, "username", req.Username)
			middleware.RateLimitResponse(w, retryAfter)
			return
		}
	}

	// Authenticate against LDAP
	result, err := ldapAuth.Authenticate(r.Context(), req.Username, req.Password)
	if err != nil {
		// Record failed attempt for rate limiting
		if loginLimiter != nil {
			loginLimiter.RecordFailure(clientIP, req.Username)
		}
		slog.Warn("auth: login failed", "username", req.Username, "ip", clientIP, "error", err)
		respondError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	// Create session (stores encrypted password for write operations)
	sess, err := sessionStore.Create(result.Username, result.DN, result.Groups, req.Password)
	if err != nil {
		slog.Error("auth: session creation failed", "error", err)
		respondError(w, http.StatusInternalServerError, "session creation failed")
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sess.ID,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Expires:  sess.Expires,
	})

	// Set CSRF cookie (readable by JS, verified by middleware)
	http.SetCookie(w, &http.Cookie{
		Name:     "sambmin_csrf",
		Value:    sess.CSRFToken,
		Path:     "/",
		HttpOnly: false, // JS must read this to send as header
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Expires:  sess.Expires,
	})

	slog.Info("auth: login successful", "username", result.Username)

	respondJSON(w, http.StatusOK, loginResponse{
		Username: result.Username,
		DN:       result.DN,
		Groups:   result.Groups,
	})
}

func handleLogoutImpl(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookieName)
	if err == nil && cookie.Value != "" {
		if sessionStore != nil {
			sessionStore.Delete(cookie.Value)
		}
	}

	// Clear cookie
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})

	respondJSON(w, http.StatusOK, map[string]string{"status": "logged out"})
}

func handleMeImpl(w http.ResponseWriter, r *http.Request) {
	sess := SessionFromRequest(r)
	if sess == nil {
		respondError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"username": sess.Username,
		"dn":       sess.DN,
		"groups":   sess.Groups,
		"expires":  sess.Expires.Format(time.RFC3339),
	})
}

// SessionFromRequest extracts the session from the request context (set by
// RequireAuth middleware) or falls back to cookie lookup.
func SessionFromRequest(r *http.Request) *auth.Session {
	// Check context first — set by auth.RequireAuth middleware
	if sess, ok := r.Context().Value(auth.SessionKey).(*auth.Session); ok && sess != nil {
		return sess
	}
	// Fallback to cookie lookup (for tests that call handlers directly)
	if sessionStore == nil {
		return nil
	}
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || cookie.Value == "" {
		return nil
	}
	return sessionStore.Get(cookie.Value)
}
