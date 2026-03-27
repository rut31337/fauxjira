package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBasicAuthValid(t *testing.T) {
	db, _ := testDB(t)
	handler := BasicAuth(db, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(ctxUserKey).(*User)
		_, _ = w.Write([]byte(user.Username))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.SetBasicAuth("alice", "testuser")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != 200 {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if rr.Body.String() != "alice" {
		t.Errorf("expected 'alice', got '%s'", rr.Body.String())
	}
}

func TestBasicAuthInvalid(t *testing.T) {
	db, _ := testDB(t)
	handler := BasicAuth(db, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("should not reach"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.SetBasicAuth("alice", "wrongpassword")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != 401 {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestBasicAuthNoCredentials(t *testing.T) {
	db, _ := testDB(t)
	handler := BasicAuth(db, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("should not reach"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != 401 {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestAdminOnlyAllowsAdmin(t *testing.T) {
	db, _ := testDB(t)
	handler := BasicAuth(db, AdminOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("admin-only"))
	})))

	req := httptest.NewRequest("GET", "/", nil)
	req.SetBasicAuth("admin", "testadmin")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Errorf("expected 200 for admin, got %d", rr.Code)
	}

	req = httptest.NewRequest("GET", "/", nil)
	req.SetBasicAuth("alice", "testuser")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != 403 {
		t.Errorf("expected 403 for user, got %d", rr.Code)
	}
}
