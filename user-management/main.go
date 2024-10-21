package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gocql/gocql"
	"github.com/gofiber/fiber/v2"
)

var session *gocql.Session

type User struct {
	ID                gocql.UUID `json:"id"`
	Username          string     `json:"username"`
	Email             string     `json:"email"`
	Password          string     `json:"password"`
	EmailVerified     bool       `json:"email_verified"`
	VerificationToken string     `json:"verification_token"`
}

type RecoverRequest struct {
	Email string `json:"email"`
}

type ResetRequest struct {
	Password string `json:"password"`
}

type Response struct {
	Status  bool   `json:"status"`
	Message string `json:"message,omitempty"` // Use omitempty to skip empty messages
	Data    *User  `json:"data,omitempty"`    // Use pointer to User to allow for nil when there's no user data
	Error   *Error `json:"error,omitempty"`   // Optional error field for detailed errors
}

type Error struct {
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

func main() {
	// Connect to Cassandra
	cluster := gocql.NewCluster("127.0.0.1")
	cluster.Keyspace = "user_management"
	cluster.Consistency = gocql.Quorum
	var err error
	session, err = cluster.CreateSession()
	if err != nil {
		log.Fatal("Unable to connect to Cassandra:", err)
	}
	defer session.Close()

	// Initialize Fiber
	app := fiber.New()

	// Routes
	app.Post("/register", registerUser)
	app.Get("/verify/:token", verifyEmail)
	app.Post("/recover", recoverPassword)
	app.Post("/reset/:token", resetPassword)

	log.Fatal(app.Listen(":3000"))
}

// Register a new user
func registerUser(c *fiber.Ctx) error {
	user := new(User)
	if err := c.BodyParser(user); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(Response{
			Status:  false,
			Message: "Invalid request",
		})
	}

	// Check if the username already exists
	var existingUsername string
	err := session.Query(`SELECT username FROM users WHERE username = ? LIMIT 1`, user.Username).Scan(&existingUsername)
	if err != nil && err != gocql.ErrNotFound {
		return c.Status(fiber.StatusInternalServerError).JSON(Response{
			Status:  false,
			Message: "Error checking username",
		})
	}
	if existingUsername != "" {
		return c.Status(fiber.StatusConflict).JSON(Response{
			Status:  false,
			Message: "Username already exists",
		})
	}

	// Check if the email already exists
	var existingEmail string
	err = session.Query(`SELECT email FROM users WHERE email = ? LIMIT 1`, user.Email).Scan(&existingEmail)
	if err != nil && err != gocql.ErrNotFound {
		return c.Status(fiber.StatusInternalServerError).JSON(Response{
			Status:  false,
			Message: "Error checking email",
		})
	}
	if existingEmail != "" {
		return c.Status(fiber.StatusConflict).JSON(Response{
			Status:  false,
			Message: "Email already exists",
		})
	}

	user.ID = gocql.TimeUUID()
	user.EmailVerified = false
	user.VerificationToken = generateToken()
	hashedPassword, err := hashPassword(user.Password)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(Response{
			Status:  false,
			Message: "Error hashing password",
			Data:    nil,
			Error: &Error{
				Message: "Internal server error",
				Detail:  err.Error(),
			},
		})
	}

	// Insert user into Cassandra
	if err := session.Query(`
        INSERT INTO users (id, username, email, password, email_verified, verification_token, created_at)
        VALUES (?, ?, ?, ?, ?, ?, ?)`,
		user.ID, user.Username, user.Email, hashedPassword, user.EmailVerified, user.VerificationToken, time.Now()).Exec(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(Response{
			Status:  false,
			Message: "Error registering user",
		})
	}

	// Simulate sending an email (just print the link)
	log.Printf("Email verification link: http://localhost:3000/verify/%s\n", user.VerificationToken)

	return c.Status(fiber.StatusCreated).JSON(Response{
		Status: true,
		Data:   user, // Include user data in the response
	})
}

// VerifyEmail verifies the user's email based on the provided token
func verifyEmail(c *fiber.Ctx) error {
	token := c.Params("token")

	// Find the user with the provided verification token
	var user User
	err := session.Query(`SELECT id, username, email, email_verified FROM users WHERE verification_token = ?`, token).Scan(&user.ID, &user.Username, &user.Email, &user.EmailVerified)

	if err != nil {
		if err == gocql.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(Response{
				Status:  false,
				Message: "Invalid token",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(Response{
			Status:  false,
			Message: "Error retrieving user",
		})
	}

	// Update the user's email_verified status
	user.EmailVerified = true
	if err := session.Query(`UPDATE users SET email_verified = ? WHERE id = ?`, user.EmailVerified, user.ID).Exec(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(Response{
			Status:  false,
			Message: "Error updating user verification status",
		})
	}

	return c.Status(fiber.StatusOK).JSON(Response{
		Status:  true,
		Message: "Email successfully verified",
	})
}

// PasswordRecovery handles the password recovery request
func recoverPassword(c *fiber.Ctx) error {
	recoverRequest := new(RecoverRequest)
	if err := c.BodyParser(recoverRequest); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(Response{
			Status:  false,
			Message: "Invalid request",
		})
	}

	email := strings.TrimSpace(recoverRequest.Email)
	log.Printf("Received email for recovery: %s\n", email) // Debug log

	// Find the user by email
	var user User
	err := session.Query(`SELECT id, username, email FROM users WHERE email = ?`, email).Scan(&user.ID, &user.Username, &user.Email)

	if err != nil {
		if err == gocql.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(Response{
				Status:  false,
				Message: "Email not found",
				Data:    nil,
				Error:   nil,
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(Response{
			Status:  false,
			Message: "Error processing password recovery",
			Data:    nil,
			Error: &Error{
				Message: "Internal server error",
				Detail:  err.Error(),
			},
		})
	}

	// Generate a verification token
	verificationToken := generateToken()

	// Store the verification token in the database
	err = session.Query(`UPDATE users SET verification_token = ? WHERE id = ?`, verificationToken, user.ID).Exec()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(Response{
			Status:  false,
			Message: "Error storing verification token",
			Data:    nil,
			Error: &Error{
				Message: "Internal server error",
				Detail:  err.Error(),
			},
		})
	}

	// Send email with the verification link
	err = sendEmail(user.Email, verificationToken)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(Response{
			Status:  false,
			Message: "Error sending email",
			Data:    nil,
			Error: &Error{
				Message: "Internal server error",
				Detail:  err.Error(),
			},
		})
	}

	return c.Status(fiber.StatusOK).JSON(Response{
		Status:  true,
		Message: "Password recovery email sent successfully",
		Data:    nil,
		Error:   nil,
	})
}

// ResetPassword handles the actual password reset
func resetPassword(c *fiber.Ctx) error {
	token := c.Params("token")
	resetRequest := new(ResetRequest)
	if err := c.BodyParser(resetRequest); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(Response{
			Status:  false,
			Message: "Invalid request",
		})
	}

	newPassword := resetRequest.Password

	// Find the user by verification token
	var user User
	err := session.Query(`SELECT id, username FROM users WHERE verification_token = ?`, token).Scan(&user.ID, &user.Username)

	if err != nil {
		if err == gocql.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(Response{
				Status:  false,
				Message: "Invalid token",
				Data:    nil,
				Error:   nil,
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(Response{
			Status:  false,
			Message: "Error processing password reset",
			Data:    nil,
			Error: &Error{
				Message: "Internal server error",
				Detail:  err.Error(),
			},
		})
	}

	// Update the user's password (ensure you hash the password before storing it)
	hashedPassword, err := hashPassword(newPassword)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(Response{
			Status:  false,
			Message: "Error hashing password",
			Data:    nil,
			Error: &Error{
				Message: "Internal server error",
				Detail:  err.Error(),
			},
		})
	}

	// Update the password and clear the verification token
	err = session.Query(`UPDATE users SET password = ?, verification_token = null WHERE id = ?`, hashedPassword, user.ID).Exec()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(Response{
			Status:  false,
			Message: "Error updating password",
			Data:    nil,
			Error: &Error{
				Message: "Internal server error",
				Detail:  err.Error(),
			},
		})
	}

	return c.Status(fiber.StatusOK).JSON(Response{
		Status:  true,
		Message: "Password successfully reset",
		Data:    nil,
		Error:   nil,
	})
}

// Generate a random token (for email verification and password reset)
func generateToken() string {
	token := make([]byte, 16)
	rand.Read(token)
	return hex.EncodeToString(token)
}

// Mock function to send email (implement this according to your email service)
func sendEmail(to, token string) error {
	// Log the email sending details
	recoveryLink := fmt.Sprintf("http://localhost:3000/password/reset/%s", token) // Adjust URL as necessary
	log.Printf("Sending password recovery email to: %s\nLink: %s\n", to, recoveryLink)

	// Here you'd typically use an email library to send the email.
	// For demonstration, we'll just return nil.
	return nil
}

// Hash password function (implement according to your security requirements)
func hashPassword(password string) (string, error) {
	// Implement hashing logic, e.g., using bcrypt
	return password, nil // For demonstration only; implement actual hashing
}
