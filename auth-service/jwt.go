package main

import (
	"log"
	"os"
	"time"

	"github.com/gocql/gocql"
	"github.com/golang-jwt/jwt/v5"
)

// GenerateJWT generates a new JWT token for the user
func GenerateJWT(userID gocql.UUID) (string, error) {
	jwtSecret := []byte(os.Getenv("JWT_SECRET"))
	if len(jwtSecret) == 0 {
		log.Println("JWT_SECRET environment variable not set")
		return "", jwt.ErrInvalidKeyType
	}
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(15 * time.Minute).Unix(), // Access token valid for 15 minutes
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ParseJWT validates the token and returns the claims
func ParseJWT(tokenStr string) (jwt.MapClaims, error) {
	jwtSecret := []byte(os.Getenv("JWT_SECRET"))
	if len(jwtSecret) == 0 {
		log.Println("JWT_SECRET environment variable not set")
		return nil, jwt.ErrInvalidKeyType
	}
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, err
}
