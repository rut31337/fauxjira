package main

import (
	"database/sql"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

func InitDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if err := migrate(db); err != nil {
		return nil, err
	}
	return db, nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			username TEXT PRIMARY KEY,
			display_name TEXT NOT NULL,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'user'
		);
		CREATE TABLE IF NOT EXISTS tickets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			key TEXT UNIQUE NOT NULL,
			summary TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'To Do',
			assignee TEXT NOT NULL DEFAULT '',
			reporter TEXT NOT NULL DEFAULT '',
			labels TEXT NOT NULL DEFAULT '[]',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS ticket_seq (
			next_id INTEGER NOT NULL DEFAULT 1
		);
		INSERT OR IGNORE INTO ticket_seq (rowid, next_id) VALUES (1, 1);
	`)
	return err
}

func SeedUsers(db *sql.DB, cfg Config) error {
	users := []struct {
		username    string
		displayName string
		password    string
		role        string
	}{
		{"admin", "Admin", cfg.AdminPassword, "admin"},
		{"alice", "Alice Chen", cfg.UserPassword, "user"},
		{"bob", "Bob Park", cfg.UserPassword, "user"},
	}
	for _, u := range users {
		hash, err := bcrypt.GenerateFromPassword([]byte(u.password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		_, err = db.Exec(
			`INSERT OR IGNORE INTO users (username, display_name, password_hash, role) VALUES (?, ?, ?, ?)`,
			u.username, u.displayName, string(hash), u.role,
		)
		if err != nil {
			return err
		}
	}
	fmt.Printf("Demo users seeded: alice, bob (password: %s)\n", cfg.UserPassword)
	return nil
}

func NextTicketKey(db *sql.DB) (string, error) {
	tx, err := db.Begin()
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback() }()

	var id int
	err = tx.QueryRow("SELECT next_id FROM ticket_seq WHERE rowid = 1").Scan(&id)
	if err != nil {
		return "", err
	}
	_, err = tx.Exec("UPDATE ticket_seq SET next_id = ? WHERE rowid = 1", id+1)
	if err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return fmt.Sprintf("FJ-%d", id), nil
}

func CreateTicket(db *sql.DB, t *Ticket) error {
	key, err := NextTicketKey(db)
	if err != nil {
		return err
	}
	t.Key = key
	now := time.Now().UTC()
	t.CreatedAt = now
	t.UpdatedAt = now
	if t.Status == "" {
		t.Status = "To Do"
	}
	result, err := db.Exec(
		`INSERT INTO tickets (key, summary, description, status, assignee, reporter, labels, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.Key, t.Summary, t.Description, t.Status, t.Assignee, t.Reporter, t.LabelsJSON(), t.CreatedAt, t.UpdatedAt,
	)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	t.ID = int(id)
	return nil
}

func GetTicket(db *sql.DB, key string) (*Ticket, error) {
	t := &Ticket{}
	var labelsStr string
	err := db.QueryRow(
		`SELECT id, key, summary, description, status, assignee, reporter, labels, created_at, updated_at
		 FROM tickets WHERE key = ?`, key,
	).Scan(&t.ID, &t.Key, &t.Summary, &t.Description, &t.Status, &t.Assignee, &t.Reporter, &labelsStr, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	t.Labels = ParseLabels(labelsStr)
	return t, nil
}

func UpdateTicket(db *sql.DB, key string, updates map[string]interface{}) (*Ticket, error) {
	t, err := GetTicket(db, key)
	if err != nil {
		return nil, err
	}
	if v, ok := updates["summary"].(string); ok {
		t.Summary = v
	}
	if v, ok := updates["description"].(string); ok {
		t.Description = v
	}
	if v, ok := updates["status"].(string); ok {
		t.Status = v
	}
	if v, ok := updates["assignee"].(string); ok {
		t.Assignee = v
	}
	if v, ok := updates["labels"].([]string); ok {
		t.Labels = v
	}
	t.UpdatedAt = time.Now().UTC()
	_, err = db.Exec(
		`UPDATE tickets SET summary=?, description=?, status=?, assignee=?, labels=?, updated_at=? WHERE key=?`,
		t.Summary, t.Description, t.Status, t.Assignee, t.LabelsJSON(), t.UpdatedAt, t.Key,
	)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func DeleteTicket(db *sql.DB, key string) error {
	result, err := db.Exec("DELETE FROM tickets WHERE key = ?", key)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func ListTickets(db *sql.DB) ([]Ticket, error) {
	rows, err := db.Query(
		`SELECT id, key, summary, description, status, assignee, reporter, labels, created_at, updated_at
		 FROM tickets ORDER BY id DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var tickets []Ticket
	for rows.Next() {
		var t Ticket
		var labelsStr string
		err := rows.Scan(&t.ID, &t.Key, &t.Summary, &t.Description, &t.Status, &t.Assignee, &t.Reporter, &labelsStr, &t.CreatedAt, &t.UpdatedAt)
		if err != nil {
			return nil, err
		}
		t.Labels = ParseLabels(labelsStr)
		tickets = append(tickets, t)
	}
	return tickets, nil
}

func GetUser(db *sql.DB, username string) (*User, error) {
	u := &User{}
	err := db.QueryRow(
		`SELECT username, display_name, password_hash, role FROM users WHERE username = ?`, username,
	).Scan(&u.Username, &u.DisplayName, &u.PasswordHash, &u.Role)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func ListUsers(db *sql.DB) ([]User, error) {
	rows, err := db.Query(`SELECT username, display_name, role FROM users ORDER BY username`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.Username, &u.DisplayName, &u.Role); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func ResetData(db *sql.DB, cfg Config) error {
	_, err := db.Exec("DELETE FROM tickets; UPDATE ticket_seq SET next_id = 1 WHERE rowid = 1; DELETE FROM users;")
	if err != nil {
		return err
	}
	return SeedUsers(db, cfg)
}
