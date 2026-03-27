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
		json.NewEncoder(&buf).Encode(body)
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
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["key"] != "FJ-1" {
		t.Errorf("expected key FJ-1, got %v", resp["key"])
	}
}

func TestAPIGetIssue(t *testing.T) {
	db, _, mux := testServer(t)
	CreateTicket(db, &Ticket{Summary: "Get me", Reporter: "admin", Labels: []string{}})

	req := authedRequest("GET", "/rest/api/2/issue/FJ-1", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	fields := resp["fields"].(map[string]interface{})
	if fields["summary"] != "Get me" {
		t.Errorf("expected summary 'Get me', got '%v'", fields["summary"])
	}
}

func TestAPIUpdateIssue(t *testing.T) {
	db, _, mux := testServer(t)
	CreateTicket(db, &Ticket{Summary: "Update me", Reporter: "admin", Labels: []string{}})

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
	CreateTicket(db, &Ticket{Summary: "Delete me", Reporter: "admin", Labels: []string{}})

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
