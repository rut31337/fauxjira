package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testServer(t *testing.T) (*sql.DB, Config, *http.ServeMux) {
	t.Helper()
	db, cfg := testDB(t)
	mux := http.NewServeMux()
	RegisterIssueRoutes(mux, db)
	return db, cfg, mux
}

func authedRequest(method, path string, body interface{}) *http.Request {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth("admin", "testadmin")
	return req
}

func TestAPICreateIssue(t *testing.T) {
	_, _, mux := testServer(t)

	body := map[string]interface{}{
		"fields": map[string]interface{}{
			"summary":     "Test issue",
			"description": "A test issue",
			"assignee":    map[string]string{"name": "alice"},
			"labels":      []string{"bug"},
		},
	}

	req := authedRequest("POST", "/rest/api/2/issue", body)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != 201 {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["key"] != "FJ-1" {
		t.Errorf("expected key FJ-1, got %v", resp["key"])
	}
}

func TestAPIGetIssue(t *testing.T) {
	db, _, mux := testServer(t)
	_ = CreateTicket(db, &Ticket{Summary: "Get me", Reporter: "admin", Labels: []string{}})

	req := authedRequest("GET", "/rest/api/2/issue/FJ-1", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	fields := resp["fields"].(map[string]interface{})
	if fields["summary"] != "Get me" {
		t.Errorf("expected summary 'Get me', got '%v'", fields["summary"])
	}
}

func TestAPIUpdateIssue(t *testing.T) {
	db, _, mux := testServer(t)
	_ = CreateTicket(db, &Ticket{Summary: "Update me", Reporter: "admin", Labels: []string{}})

	body := map[string]interface{}{
		"fields": map[string]interface{}{
			"summary": "Updated",
			"status":  map[string]string{"name": "In Progress"},
		},
	}
	req := authedRequest("PUT", "/rest/api/2/issue/FJ-1", body)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != 204 {
		t.Fatalf("expected 204, got %d: %s", rr.Code, rr.Body.String())
	}

	ticket, _ := GetTicket(db, "FJ-1")
	if ticket.Summary != "Updated" {
		t.Errorf("expected 'Updated', got '%s'", ticket.Summary)
	}
	if ticket.Status != "In Progress" {
		t.Errorf("expected 'In Progress', got '%s'", ticket.Status)
	}
}

func TestAPIDeleteIssue(t *testing.T) {
	db, _, mux := testServer(t)
	_ = CreateTicket(db, &Ticket{Summary: "Delete me", Reporter: "admin", Labels: []string{}})

	req := authedRequest("DELETE", "/rest/api/2/issue/FJ-1", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != 204 {
		t.Fatalf("expected 204, got %d", rr.Code)
	}

	_, err := GetTicket(db, "FJ-1")
	if err == nil {
		t.Error("expected ticket to be deleted")
	}
}

func TestAPISearch(t *testing.T) {
	db, _, mux := testServer(t)
	RegisterSearchRoutes(mux, db)

	_ = CreateTicket(db, &Ticket{Summary: "Bug one", Status: "To Do", Assignee: "alice", Reporter: "admin", Labels: []string{"bug"}})
	_ = CreateTicket(db, &Ticket{Summary: "Task two", Status: "In Progress", Assignee: "bob", Reporter: "admin", Labels: []string{"infra"}})

	req := authedRequest("GET", `/rest/api/2/search?jql=assignee+%3D+%22alice%22`, nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	total := int(resp["total"].(float64))
	if total != 1 {
		t.Errorf("expected 1 result, got %d", total)
	}
}

func TestAPIUserLookup(t *testing.T) {
	db, _, mux := testServer(t)
	RegisterUserRoutes(mux, db)

	req := authedRequest("GET", "/rest/api/2/user?username=alice", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var resp map[string]interface{}
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["displayName"] != "Alice Chen" {
		t.Errorf("expected 'Alice Chen', got '%v'", resp["displayName"])
	}
}

func TestAPIServerInfo(t *testing.T) {
	db, _, mux := testServer(t)
	RegisterUserRoutes(mux, db)

	req := authedRequest("GET", "/rest/api/2/serverInfo", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var resp map[string]interface{}
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["serverTitle"] != "fauxjira" {
		t.Errorf("expected 'fauxjira', got '%v'", resp["serverTitle"])
	}
}

func TestAPIAdminReset(t *testing.T) {
	db, cfg, mux := testServer(t)
	RegisterAdminRoutes(mux, db, cfg)

	_ = CreateTicket(db, &Ticket{Summary: "Will be wiped", Reporter: "admin", Labels: []string{}})

	req := authedRequest("POST", "/admin/reset", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	tickets, _ := ListTickets(db)
	if len(tickets) != 0 {
		t.Errorf("expected 0 tickets, got %d", len(tickets))
	}
}

func TestAPIAdminListUsers(t *testing.T) {
	db, cfg, mux := testServer(t)
	RegisterAdminRoutes(mux, db, cfg)

	req := authedRequest("GET", "/admin/users", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var users []map[string]interface{}
	_ = json.Unmarshal(rr.Body.Bytes(), &users)
	if len(users) != 3 {
		t.Errorf("expected 3 users, got %d", len(users))
	}
}
