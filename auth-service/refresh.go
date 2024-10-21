package main

import (
	"fmt"
	"log"
	"time"

	"github.com/bdobrica/LLMDesignedApp/go-common/auth"
	"github.com/gocql/gocql"
)

// GenerateRefreshToken generates a new refresh token for the user
func GenerateRefreshToken(userID gocql.UUID) (string, error) {
	refreshToken, err := auth.GenerateBase64RandomToken(32) // Use a strong random generator here
	if err != nil {
		log.Println("Error generating a random token")
		return "", err
	}
	expiresAt := time.Now().Add(7 * 24 * time.Hour) // Refresh token valid for 7 days

	err = session.Query(`INSERT INTO refresh_tokens ("token", user_id, expires_at) VALUES (?, ?, ?)`,
		refreshToken, userID, expiresAt).Exec()
	if err != nil {
		log.Println("Error inserting refresh token into the database")
		return "", err
	}

	return refreshToken, nil
}

// ValidateRefreshToken checks if the refresh token is valid
func ValidateRefreshToken(token string) (gocql.UUID, error) {
	var userID gocql.UUID
	var expiresAt time.Time

	err := session.Query(`SELECT user_id, expires_at FROM refresh_tokens WHERE "token" = ?`, token).
		Scan(&userID, &expiresAt)
	if err != nil {
		log.Println("Error scanning refresh token from the database")
		return gocql.UUID{}, err
	}

	if time.Now().After(expiresAt) {
		log.Println("Refresh token expired")
		// Revoke the token if it's expired
		err = RevokeRefreshToken(token)
		if err != nil {
			log.Println("Error revoking expired refresh token")
		}
		return gocql.UUID{}, fmt.Errorf("refresh token expired")
	}

	return userID, nil
}

// RevokeRefreshToken deletes the token from the database
func RevokeRefreshToken(token string) error {
	err := session.Query(`DELETE FROM refresh_tokens WHERE "token" = ?`, token).Exec()
	if err != nil {
		log.Println("Error deleting refresh token from the database")
		return err
	}
	return nil
}
