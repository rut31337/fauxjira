package server

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
)

func RegisterIssueRoutes(mux *http.ServeMux, db *sql.DB) {
	mux.Handle("/rest/api/2/issue/", BasicAuth(db, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimPrefix(r.URL.Path, "/rest/api/2/issue/")
		if key == "" {
			http.Error(w, "issue key required", http.StatusBadRequest)
			return
		}
		switch r.Method {
		case "GET":
			handleGetIssue(w, r, db, key)
		case "PUT":
			handleUpdateIssue(w, r, db, key)
		case "DELETE":
			handleDeleteIssue(w, r, db, key)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})))

	mux.Handle("/rest/api/2/issue", BasicAuth(db, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handleCreateIssue(w, r, db)
	})))
}

func handleCreateIssue(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var req struct {
		Fields struct {
			Summary     string            `json:"summary"`
			Description string            `json:"description"`
			Assignee    map[string]string `json:"assignee"`
			Labels      []string          `json:"labels"`
		} `json:"fields"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	user := CurrentUser(r)
	ticket := &Ticket{
		Summary:     req.Fields.Summary,
		Description: req.Fields.Description,
		Assignee:    req.Fields.Assignee["name"],
		Reporter:    user.Username,
		Labels:      req.Fields.Labels,
	}
	if ticket.Labels == nil {
		ticket.Labels = []string{}
	}

	if err := CreateTicket(db, ticket); err != nil {
		http.Error(w, "failed to create issue", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"id":   ticket.ID,
		"key":  ticket.Key,
		"self": r.Host + "/rest/api/2/issue/" + ticket.Key,
	})
}

func handleGetIssue(w http.ResponseWriter, r *http.Request, db *sql.DB, key string) {
	ticket, err := GetTicket(db, key)
	if err != nil {
		http.Error(w, "issue not found", http.StatusNotFound)
		return
	}

	assigneeUser, _ := GetUser(db, ticket.Assignee)
	reporterUser, _ := GetUser(db, ticket.Reporter)

	assignee := map[string]string{"name": ticket.Assignee}
	if assigneeUser != nil {
		assignee["displayName"] = assigneeUser.DisplayName
	}
	reporter := map[string]string{"name": ticket.Reporter}
	if reporterUser != nil {
		reporter["displayName"] = reporterUser.DisplayName
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"id":  ticket.ID,
		"key": ticket.Key,
		"fields": map[string]interface{}{
			"summary":     ticket.Summary,
			"description": ticket.Description,
			"status":      map[string]string{"name": ticket.Status},
			"assignee":    assignee,
			"reporter":    reporter,
			"labels":      ticket.Labels,
			"created":     ticket.CreatedAt.Format("2006-01-02T15:04:05.000+0000"),
			"updated":     ticket.UpdatedAt.Format("2006-01-02T15:04:05.000+0000"),
		},
	})
}

func handleUpdateIssue(w http.ResponseWriter, r *http.Request, db *sql.DB, key string) {
	var req struct {
		Fields map[string]interface{} `json:"fields"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	updates := map[string]interface{}{}
	for k, v := range req.Fields {
		switch k {
		case "summary":
			updates["summary"] = v
		case "description":
			updates["description"] = v
		case "status":
			if m, ok := v.(map[string]interface{}); ok {
				updates["status"] = m["name"]
			}
		case "assignee":
			if m, ok := v.(map[string]interface{}); ok {
				updates["assignee"] = m["name"]
			}
		case "labels":
			if arr, ok := v.([]interface{}); ok {
				labels := make([]string, len(arr))
				for i, l := range arr {
					labels[i] = l.(string)
				}
				updates["labels"] = labels
			}
		}
	}

	if _, err := UpdateTicket(db, key, updates); err != nil {
		http.Error(w, "issue not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handleDeleteIssue(w http.ResponseWriter, r *http.Request, db *sql.DB, key string) {
	if err := DeleteTicket(db, key); err != nil {
		http.Error(w, "issue not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
