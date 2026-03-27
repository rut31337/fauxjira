package server

import (
	"database/sql"
	"embed"
	"html/template"
	"io/fs"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static/*
var staticFS embed.FS

//go:embed assets/fauxjira-logo.png
var logoBytes []byte

//go:embed assets/favicon.ico
var faviconBytes []byte

var validStatuses = []string{"To Do", "In Progress", "In Review", "Done"}

func statusClass(status string) string {
	switch status {
	case "To Do":
		return "status-todo"
	case "In Progress":
		return "status-in-progress"
	case "In Review":
		return "status-in-review"
	case "Done":
		return "status-done"
	default:
		return "status-todo"
	}
}

func loadTemplates() *template.Template {
	funcMap := template.FuncMap{
		"statusClass": statusClass,
		"slice": func(s string, start, end int) string {
			if len(s) == 0 {
				return ""
			}
			if end > len(s) {
				end = len(s)
			}
			return strings.ToUpper(s[start:end])
		},
	}
	return template.Must(template.New("").Funcs(funcMap).ParseFS(templateFS, "templates/*.html"))
}

type webHandler struct {
	db   *sql.DB
	tmpl *template.Template
	cfg  Config
}

func RegisterWebRoutes(mux *http.ServeMux, db *sql.DB, cfg Config) {
	h := &webHandler{db: db, tmpl: loadTemplates(), cfg: cfg}

	// Serve static files
	staticSub, _ := fs.Sub(staticFS, "static")
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticSub))))

	// Serve logo
	mux.HandleFunc("/static/logo.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(logoBytes)
	})

	// Serve favicon
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/x-icon")
		_, _ = w.Write(faviconBytes)
	})

	// Pages
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/login", http.StatusFound)
	})
	mux.HandleFunc("/login", h.handleLogin)
	mux.HandleFunc("/tickets", h.handleTickets)
	mux.HandleFunc("/issue/", h.handleTicketDetail)

	// htmx endpoints
	mux.HandleFunc("/web/create", h.handleWebCreate)
	mux.HandleFunc("/web/update/status/", h.handleWebUpdateStatus)
	mux.HandleFunc("/web/update/assignee/", h.handleWebUpdateAssignee)
}

func (h *webHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		_ = h.tmpl.ExecuteTemplate(w, "login.html", map[string]interface{}{"Error": ""})
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	user, err := GetUser(h.db, username)
	if err != nil {
		_ = h.tmpl.ExecuteTemplate(w, "login.html", map[string]interface{}{"Error": "Invalid credentials"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		_ = h.tmpl.ExecuteTemplate(w, "login.html", map[string]interface{}{"Error": "Invalid credentials"})
		return
	}

	// Set a simple session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "fauxjira_user",
		Value:    user.Username,
		Path:     "/",
		HttpOnly: true,
	})
	http.Redirect(w, r, "/tickets", http.StatusFound)
}

func (h *webHandler) currentWebUser(r *http.Request) *User {
	cookie, err := r.Cookie("fauxjira_user")
	if err != nil {
		return nil
	}
	user, err := GetUser(h.db, cookie.Value)
	if err != nil {
		return nil
	}
	return user
}

func (h *webHandler) requireLogin(w http.ResponseWriter, r *http.Request) *User {
	user := h.currentWebUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return nil
	}
	return user
}

func (h *webHandler) handleTickets(w http.ResponseWriter, r *http.Request) {
	user := h.requireLogin(w, r)
	if user == nil {
		return
	}

	tickets, _ := ListTickets(h.db)
	users, _ := ListUsers(h.db)

	data := map[string]interface{}{
		"CurrentUser": user,
		"Tickets":     tickets,
		"Users":       users,
	}
	_ = h.tmpl.ExecuteTemplate(w, "tickets", data)
}

func (h *webHandler) handleTicketDetail(w http.ResponseWriter, r *http.Request) {
	user := h.requireLogin(w, r)
	if user == nil {
		return
	}

	key := strings.TrimPrefix(r.URL.Path, "/issue/")
	ticket, err := GetTicket(h.db, key)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	users, _ := ListUsers(h.db)

	data := map[string]interface{}{
		"CurrentUser": user,
		"Ticket":      ticket,
		"Users":       users,
		"Statuses":    validStatuses,
	}
	_ = h.tmpl.ExecuteTemplate(w, "ticket_detail", data)
}

func (h *webHandler) handleWebCreate(w http.ResponseWriter, r *http.Request) {
	user := h.currentWebUser(r)
	if user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	summary := r.FormValue("summary")
	description := r.FormValue("description")
	assignee := r.FormValue("assignee")
	labelsStr := r.FormValue("labels")

	var labels []string
	if labelsStr != "" {
		for _, l := range strings.Split(labelsStr, ",") {
			l = strings.TrimSpace(l)
			if l != "" {
				labels = append(labels, l)
			}
		}
	}
	if labels == nil {
		labels = []string{}
	}

	ticket := &Ticket{
		Summary:     summary,
		Description: description,
		Assignee:    assignee,
		Reporter:    user.Username,
		Labels:      labels,
	}
	_ = CreateTicket(h.db, ticket)

	// Return the new row as an htmx fragment
	_ = h.tmpl.ExecuteTemplate(w, "ticket_row", ticket)
}

func (h *webHandler) handleWebUpdateStatus(w http.ResponseWriter, r *http.Request) {
	user := h.currentWebUser(r)
	if user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	key := strings.TrimPrefix(r.URL.Path, "/web/update/status/")
	status := r.FormValue("status")
	_, _ = UpdateTicket(h.db, key, map[string]interface{}{"status": status})

	ticket, _ := GetTicket(h.db, key)
	users, _ := ListUsers(h.db)
	data := map[string]interface{}{
		"Ticket":   ticket,
		"Users":    users,
		"Statuses": validStatuses,
	}
	_ = h.tmpl.ExecuteTemplate(w, "status_dropdown", data)
}

func (h *webHandler) handleWebUpdateAssignee(w http.ResponseWriter, r *http.Request) {
	user := h.currentWebUser(r)
	if user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	key := strings.TrimPrefix(r.URL.Path, "/web/update/assignee/")
	assignee := r.FormValue("assignee")
	_, _ = UpdateTicket(h.db, key, map[string]interface{}{"assignee": assignee})

	ticket, _ := GetTicket(h.db, key)
	users, _ := ListUsers(h.db)
	data := map[string]interface{}{
		"Ticket":   ticket,
		"Users":    users,
		"Statuses": validStatuses,
	}
	_ = h.tmpl.ExecuteTemplate(w, "assignee_dropdown", data)
}
