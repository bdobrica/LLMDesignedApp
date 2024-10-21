package main

import (
	"github.com/bdobrica/LLMDesignedApp/go-common/auth"
	"github.com/gofiber/fiber/v2"
)

// Login handler - generates JWT and refresh tokens
func login(c *fiber.Ctx) error {
	var data struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.BodyParser(&data); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": false, "message": "Invalid request"})
	}

	// Find user in Cassandra
	var user User
	err := session.Query(`SELECT id, password FROM users WHERE username = ?`, data.Username).
		Scan(&user.ID, &user.Password)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": false, "message": "Invalid email or password"})
	}

	// Check password
	if !auth.CheckPasswordHash(data.Password, user.Password) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": false, "message": "Invalid email or password"})
	}

	// Generate JWT
	jwtToken, err := GenerateJWT(user.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": false, "message": "Error generating token"})
	}

	// Generate Refresh Token
	refreshToken, err := GenerateRefreshToken(user.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": false, "message": "Error generating refresh token"})
	}

	// Return tokens
	return c.JSON(fiber.Map{
		"status":  true,
		"message": "Login successful",
		"data": fiber.Map{
			"access_token":  jwtToken,
			"refresh_token": refreshToken,
			"expires_in":    900, // 15 minutes in seconds
		},
	})
}

// Refresh token handler
func refreshToken(c *fiber.Ctx) error {
	var data struct {
		Token string `json:"refresh_token"`
	}
	if err := c.BodyParser(&data); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": false, "message": "Invalid request"})
	}

	// Validate refresh token
	userID, err := ValidateRefreshToken(data.Token)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": false, "message": "Invalid or expired refresh token"})
	}

	// Generate new JWT
	jwtToken, err := GenerateJWT(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": false, "message": "Error generating token"})
	}

	return c.JSON(fiber.Map{
		"status":  true,
		"message": "Token refreshed",
		"data": fiber.Map{
			"access_token": jwtToken,
			"expires_in":   900, // 15 minutes in seconds
		},
	})
}

// Logout handler - revokes refresh token
func logout(c *fiber.Ctx) error {
	var data struct {
		Token string `json:"refresh_token"`
	}
	if err := c.BodyParser(&data); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": false, "message": "Invalid request"})
	}

	// Revoke refresh token
	if err := RevokeRefreshToken(data.Token); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": false, "message": "Error revoking token"})
	}

	return c.JSON(fiber.Map{
		"status":  true,
		"message": "Logged out successfully",
	})
}
