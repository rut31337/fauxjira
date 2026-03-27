package server

import (
	"context"
	"database/sql"
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

type contextKey string

const ctxUserKey contextKey = "user"

func BasicAuth(db *sql.DB, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok {
			w.Header().Set("WWW-Authenticate", `Basic realm="fauxjira"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		user, err := GetUser(db, username)
		if err != nil {
			w.Header().Set("WWW-Authenticate", `Basic realm="fauxjira"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
			w.Header().Set("WWW-Authenticate", `Basic realm="fauxjira"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), ctxUserKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func AdminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(ctxUserKey).(*User)
		if user.Role != "admin" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func CurrentUser(r *http.Request) *User {
	if u, ok := r.Context().Value(ctxUserKey).(*User); ok {
		return u
	}
	return nil
}
