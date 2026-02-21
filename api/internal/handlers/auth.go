package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/nickdawson/sambmin/internal/auth"
)

var (
	sessionStore *auth.Store
	ldapAuth     *auth.LDAPAuthenticator
)

const sessionCookieName = "sambmin_session"

// InitAuth sets up the session store and LDAP authenticator.
// Called from main.go during startup.
func InitAuth(store *auth.Store, authenticator *auth.LDAPAuthenticator) {
	sessionStore = store
	ldapAuth = authenticator
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

	// Authenticate against LDAP
	result, err := ldapAuth.Authenticate(r.Context(), req.Username, req.Password)
	if err != nil {
		slog.Warn("auth: login failed", "username", req.Username, "error", err)
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

// SessionFromRequest extracts the session from a request cookie.
// Used by middleware and handlers.
func SessionFromRequest(r *http.Request) *auth.Session {
	if sessionStore == nil {
		return nil
	}
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || cookie.Value == "" {
		return nil
	}
	return sessionStore.Get(cookie.Value)
}

// contextKey for storing session in request context.
type authContextKey string

const sessionContextKey authContextKey = "session"

// SessionFromContext retrieves the session from the request context.
func SessionFromContext(ctx interface{ Value(any) any }) *auth.Session {
	sess, _ := ctx.Value(sessionContextKey).(*auth.Session)
	return sess
}
