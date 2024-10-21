package auth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
)

// generateRandomToken generates a secure random token of a given length
func GenerateBase64RandomToken(length int) (string, error) {
	// Create a byte slice to hold the random bytes
	bytes := make([]byte, length)

	// Read random bytes from the crypto/rand package
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	// Encode the random bytes to a base64 string
	token := base64.URLEncoding.EncodeToString(bytes)

	// Optionally truncate the token if it exceeds the required length
	return token[:length], nil
}

func GenerateHexRandomToken(length int) (string, error) {
	// Create a byte slice to hold the random bytes; simulate ceil division by adding 1
	bytes := make([]byte, (length+1)/2)

	// Read random bytes from the crypto/rand package
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	// Encode the random bytes to a hex string
	token := hex.EncodeToString(bytes)

	// Optionally truncate the token if it exceeds the required length
	return token[:length], nil
}
