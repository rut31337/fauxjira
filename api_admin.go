package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

func RegisterAdminRoutes(mux *http.ServeMux, db *sql.DB, cfg Config) {
	mux.Handle("/admin/reset", BasicAuth(db, AdminOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := ResetData(db, cfg); err != nil {
			http.Error(w, "reset failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "reset complete"})
	}))))

	mux.Handle("/admin/users", BasicAuth(db, AdminOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		users, err := ListUsers(db)
		if err != nil {
			http.Error(w, "failed to list users", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(users)
	}))))
}
