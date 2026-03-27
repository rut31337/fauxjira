package server

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

func RegisterSearchRoutes(mux *http.ServeMux, db *sql.DB) {
	mux.Handle("/rest/api/2/search", BasicAuth(db, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handleSearch(w, r, db)
	})))
}

func handleSearch(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	jqlStr := r.URL.Query().Get("jql")
	expr, err := ParseJQL(jqlStr)
	if err != nil {
		http.Error(w, "invalid JQL: "+err.Error(), http.StatusBadRequest)
		return
	}

	allTickets, err := ListTickets(db)
	if err != nil {
		http.Error(w, "failed to list tickets", http.StatusInternalServerError)
		return
	}

	var issues []map[string]interface{}
	for _, t := range allTickets {
		if !expr.Match(&t) {
			continue
		}

		assignee := map[string]string{"name": t.Assignee}
		if u, _ := GetUser(db, t.Assignee); u != nil {
			assignee["displayName"] = u.DisplayName
		}
		reporter := map[string]string{"name": t.Reporter}
		if u, _ := GetUser(db, t.Reporter); u != nil {
			reporter["displayName"] = u.DisplayName
		}

		issues = append(issues, map[string]interface{}{
			"id":  t.ID,
			"key": t.Key,
			"fields": map[string]interface{}{
				"summary":     t.Summary,
				"description": t.Description,
				"status":      map[string]string{"name": t.Status},
				"assignee":    assignee,
				"reporter":    reporter,
				"labels":      t.Labels,
				"created":     t.CreatedAt.Format("2006-01-02T15:04:05.000+0000"),
				"updated":     t.UpdatedAt.Format("2006-01-02T15:04:05.000+0000"),
			},
		})
	}

	if issues == nil {
		issues = []map[string]interface{}{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"startAt":    0,
		"maxResults": len(issues),
		"total":      len(issues),
		"issues":     issues,
	})
}
