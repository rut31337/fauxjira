package server

import (
	"encoding/json"
	"time"
)

type User struct {
	Username     string `json:"username"`
	DisplayName  string `json:"displayName"`
	PasswordHash string `json:"-"`
	Role         string `json:"role"`
}

type Ticket struct {
	ID          int       `json:"id"`
	Key         string    `json:"key"`
	Summary     string    `json:"summary"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	Assignee    string    `json:"assignee"`
	Reporter    string    `json:"reporter"`
	Labels      []string  `json:"labels"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (t *Ticket) LabelsJSON() string {
	b, _ := json.Marshal(t.Labels)
	return string(b)
}

func ParseLabels(s string) []string {
	var labels []string
	if s == "" {
		return labels
	}
	_ = json.Unmarshal([]byte(s), &labels)
	return labels
}
