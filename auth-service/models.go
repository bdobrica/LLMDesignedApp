package main

import (
	"time"

	"github.com/gocql/gocql"
)

// User represents the user table schema
type User struct {
	ID       gocql.UUID `json:"id"`
	Username string     `json:"username"`
	Email    string     `json:"email"`
	Password string     `json:"password"`
}

// RefreshToken represents the refresh_tokens table schema
type RefreshToken struct {
	Token     string     `json:"token"`
	UserID    gocql.UUID `json:"user_id"`
	ExpiresAt time.Time  `json:"expires_at"`
}
