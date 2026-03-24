package db

import "time"

type Contact struct {
	ID             string    `db:"id"`
	Name           string    `db:"name"`
	Gender         string    `db:"gender"`
	Tags           string    `db:"tags"`
	ProfileSummary string    `db:"profile_summary"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}

type Session struct {
	ID              string    `db:"id"`
	ContactID       string    `db:"contact_id"`
	ParentSessionID *string   `db:"parent_session_id"`
	Status          string    `db:"status"`
	Summary         string    `db:"summary"`
	CreatedAt       time.Time `db:"created_at"`
	UpdatedAt       time.Time `db:"updated_at"`
}

type Message struct {
	ID        string    `db:"id"`
	SessionID string    `db:"session_id"`
	Speaker   string    `db:"speaker"`
	Content   string    `db:"content"`
	Emotion   string    `db:"emotion"`
	Intent    string    `db:"intent"`
	Timestamp time.Time `db:"timestamp"`
}
