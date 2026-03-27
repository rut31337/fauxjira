package server

import (
	"database/sql"
	"os"
	"testing"
)

func testDB(t *testing.T) (*sql.DB, Config) {
	t.Helper()
	path := t.TempDir() + "/test.db"
	cfg := Config{
		DBPath:        path,
		AdminPassword: "testadmin",
		UserPassword:  "testuser",
	}
	db, err := InitDB(cfg.DBPath)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	if err := SeedUsers(db, cfg); err != nil {
		t.Fatalf("SeedUsers: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
		_ = os.Remove(path)
	})
	return db, cfg
}

func TestCreateAndGetTicket(t *testing.T) {
	db, _ := testDB(t)
	ticket := &Ticket{
		Summary:     "Test ticket",
		Description: "A test",
		Assignee:    "alice",
		Reporter:    "admin",
		Labels:      []string{"bug"},
	}
	if err := CreateTicket(db, ticket); err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}
	if ticket.Key != "FJ-1" {
		t.Errorf("expected key FJ-1, got %s", ticket.Key)
	}

	got, err := GetTicket(db, "FJ-1")
	if err != nil {
		t.Fatalf("GetTicket: %v", err)
	}
	if got.Summary != "Test ticket" {
		t.Errorf("expected summary 'Test ticket', got '%s'", got.Summary)
	}
	if got.Status != "To Do" {
		t.Errorf("expected status 'To Do', got '%s'", got.Status)
	}
	if len(got.Labels) != 1 || got.Labels[0] != "bug" {
		t.Errorf("expected labels [bug], got %v", got.Labels)
	}
}

func TestUpdateTicket(t *testing.T) {
	db, _ := testDB(t)
	ticket := &Ticket{Summary: "Original", Reporter: "admin"}
	_ = CreateTicket(db, ticket)

	updated, err := UpdateTicket(db, "FJ-1", map[string]interface{}{
		"summary": "Updated",
		"status":  "In Progress",
	})
	if err != nil {
		t.Fatalf("UpdateTicket: %v", err)
	}
	if updated.Summary != "Updated" {
		t.Errorf("expected 'Updated', got '%s'", updated.Summary)
	}
	if updated.Status != "In Progress" {
		t.Errorf("expected 'In Progress', got '%s'", updated.Status)
	}
}

func TestDeleteTicket(t *testing.T) {
	db, _ := testDB(t)
	ticket := &Ticket{Summary: "To delete", Reporter: "admin"}
	_ = CreateTicket(db, ticket)

	if err := DeleteTicket(db, "FJ-1"); err != nil {
		t.Fatalf("DeleteTicket: %v", err)
	}
	_, err := GetTicket(db, "FJ-1")
	if err == nil {
		t.Error("expected error getting deleted ticket")
	}
}

func TestListTickets(t *testing.T) {
	db, _ := testDB(t)
	_ = CreateTicket(db, &Ticket{Summary: "First", Reporter: "admin"})
	_ = CreateTicket(db, &Ticket{Summary: "Second", Reporter: "admin"})

	tickets, err := ListTickets(db)
	if err != nil {
		t.Fatalf("ListTickets: %v", err)
	}
	if len(tickets) != 2 {
		t.Errorf("expected 2 tickets, got %d", len(tickets))
	}
}

func TestGetUser(t *testing.T) {
	db, _ := testDB(t)
	user, err := GetUser(db, "alice")
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if user.DisplayName != "Alice Chen" {
		t.Errorf("expected 'Alice Chen', got '%s'", user.DisplayName)
	}
	if user.Role != "user" {
		t.Errorf("expected role 'user', got '%s'", user.Role)
	}
}

func TestResetData(t *testing.T) {
	db, cfg := testDB(t)
	_ = CreateTicket(db, &Ticket{Summary: "Will be wiped", Reporter: "admin"})

	if err := ResetData(db, cfg); err != nil {
		t.Fatalf("ResetData: %v", err)
	}
	tickets, _ := ListTickets(db)
	if len(tickets) != 0 {
		t.Errorf("expected 0 tickets after reset, got %d", len(tickets))
	}
	user, err := GetUser(db, "alice")
	if err != nil {
		t.Fatalf("expected alice to exist after reset: %v", err)
	}
	if user.DisplayName != "Alice Chen" {
		t.Errorf("expected 'Alice Chen', got '%s'", user.DisplayName)
	}
}
