package main

import "testing"

func TestJQLParseSimple(t *testing.T) {
	expr, err := ParseJQL(`status = "To Do"`)
	if err != nil {
		t.Fatalf("ParseJQL: %v", err)
	}
	ticket := &Ticket{Status: "To Do"}
	if !expr.Match(ticket) {
		t.Error("expected match")
	}
	ticket2 := &Ticket{Status: "Done"}
	if expr.Match(ticket2) {
		t.Error("expected no match")
	}
}

func TestJQLParseAND(t *testing.T) {
	expr, err := ParseJQL(`status = "To Do" AND assignee = "alice"`)
	if err != nil {
		t.Fatalf("ParseJQL: %v", err)
	}
	match := &Ticket{Status: "To Do", Assignee: "alice"}
	if !expr.Match(match) {
		t.Error("expected match")
	}
	noMatch := &Ticket{Status: "To Do", Assignee: "bob"}
	if expr.Match(noMatch) {
		t.Error("expected no match")
	}
}

func TestJQLParseOR(t *testing.T) {
	expr, err := ParseJQL(`assignee = "alice" OR assignee = "bob"`)
	if err != nil {
		t.Fatalf("ParseJQL: %v", err)
	}
	if !expr.Match(&Ticket{Assignee: "alice"}) {
		t.Error("expected alice to match")
	}
	if !expr.Match(&Ticket{Assignee: "bob"}) {
		t.Error("expected bob to match")
	}
	if expr.Match(&Ticket{Assignee: "charlie"}) {
		t.Error("expected charlie not to match")
	}
}

func TestJQLParseLabels(t *testing.T) {
	expr, err := ParseJQL(`labels = "bug"`)
	if err != nil {
		t.Fatalf("ParseJQL: %v", err)
	}
	if !expr.Match(&Ticket{Labels: []string{"bug", "infra"}}) {
		t.Error("expected match when label present")
	}
	if expr.Match(&Ticket{Labels: []string{"infra"}}) {
		t.Error("expected no match when label absent")
	}
}

func TestJQLEmpty(t *testing.T) {
	expr, err := ParseJQL("")
	if err != nil {
		t.Fatalf("ParseJQL: %v", err)
	}
	if !expr.Match(&Ticket{}) {
		t.Error("empty JQL should match everything")
	}
}
