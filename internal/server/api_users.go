package server

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

func RegisterUserRoutes(mux *http.ServeMux, db *sql.DB) {
	mux.Handle("/rest/api/2/user", BasicAuth(db, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username := r.URL.Query().Get("username")
		if username == "" {
			http.Error(w, "username parameter required", http.StatusBadRequest)
			return
		}
		user, err := GetUser(db, username)
		if err != nil {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"name":        user.Username,
			"displayName": user.DisplayName,
			"active":      true,
		})
	})))

	mux.Handle("/rest/api/2/serverInfo", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"baseUrl":     "http://" + r.Host,
			"version":     "0.1.0",
			"serverTitle": "fauxjira",
		})
	}))
}
