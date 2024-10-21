package main

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gocql/gocql"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/gomail.v2"
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

// sendEmail sends a password recovery email using an external SMTP server.
func sendEmail(to, token string) error {
	// Create the recovery link
	recoveryLink := fmt.Sprintf("http://localhost:3000/password/reset/%s", token) // Adjust URL as necessary

	// Log email details for debugging
	log.Printf("Sending password recovery email to: %s\nLink: %s\n", to, recoveryLink)

	// Get sender's email from environment variables
	senderEmail := os.Getenv("SMTP_SENDER_EMAIL")
	if senderEmail == "" {
		log.Println("Error: SMTP_SENDER_EMAIL is not set")
		return fmt.Errorf("SMTP_SENDER_EMAIL is not set")
	}
	log.Printf("Sender email: %s\n", senderEmail)

	// Log recipient's email
	log.Printf("Recipient email: %s\n", to)
	if to == "" {
		return fmt.Errorf("recipient email is empty")
	}

	// Set up Gomail
	m := gomail.NewMessage()

	// Set the sender's email address
	m.SetHeader("From", senderEmail)

	// Set the recipient's email address
	m.SetHeader("To", to)

	// Set the subject of the email
	m.SetHeader("Subject", "Password Recovery")

	// Set the body of the email (HTML or plain text)
	m.SetBody("text/html", fmt.Sprintf(`
        <h1>Password Recovery</h1>
        <p>You have requested to reset your password. Please click the following link to reset your password:</p>
        <a href="%s">Reset Password</a>
    `, recoveryLink))

	// Use an external SMTP server to send the email
	d := gomail.NewDialer(
		os.Getenv("SMTP_HOST"),        // SMTP server host (e.g., smtp.example.com)
		getEnvAsInt("SMTP_PORT", 587), // SMTP server port (default: 587)
		os.Getenv("SMTP_USERNAME"),    // SMTP server username
		os.Getenv("SMTP_PASSWORD"),    // SMTP server password
	)

	// Set TLS configuration (useful for enforcing TLS)
	d.TLSConfig = &tls.Config{
		InsecureSkipVerify: false,            // Set this to true only for self-signed certificates in development
		MinVersion:         tls.VersionTLS12, // Enforce a minimum version of TLS (e.g., TLS 1.2)
	}

	// Send the email
	if err := d.DialAndSend(m); err != nil {
		log.Printf("Failed to send email to %s: %v\n", to, err)
		return err
	}

	log.Printf("Password recovery email successfully sent to %s\n", to)
	return nil
}

// Utility function to retrieve environment variables as integers
func getEnvAsInt(name string, defaultValue int) int {
	valueStr := os.Getenv(name)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		log.Printf("Error parsing environment variable %s: %v. Using default value: %d", name, err, defaultValue)
		return defaultValue
	}

	return value
}

// hashPassword takes a plain password as input and returns its bcrypt hashed version.
func hashPassword(password string) (string, error) {
	// Use bcrypt to generate a hashed password with a cost of bcrypt.DefaultCost (currently 10)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

// comparePasswords compares the plain password with the hashed password stored in the database
func comparePasswords(hashedPassword, plainPassword string) error {
	// Use bcrypt to compare the hashed password with the plaintext password
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword))
	return err
}
