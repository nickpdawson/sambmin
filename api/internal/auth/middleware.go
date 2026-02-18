package auth

import (
	"context"
	"net/http"
)

type contextKeyType string

const SessionKey contextKeyType = "session"

// RequireAuth is middleware that checks for a valid session.
// If no session is found, it returns 401 Unauthorized.
// If a session is found, it's added to the request context.
func RequireAuth(store *Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("sambmin_session")
			if err != nil || cookie.Value == "" {
				http.Error(w, `{"error":"authentication required"}`, http.StatusUnauthorized)
				return
			}

			sess := store.Get(cookie.Value)
			if sess == nil {
				http.Error(w, `{"error":"session expired"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), SessionKey, sess)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
