// internal/data/users.go
package data

import (
	"context"
	"database/sql"
	"errors"
	"strings" // Needed for duplicate email check
	"time"

	"github.com/amari03/habit-tracker/internal/validator" // Keep your validator import
	"golang.org/x/crypto/bcrypt"                          // Needed for password hashing
)

var (
	ErrDuplicateEmail     = errors.New("duplicate email")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// --- Custom Password Type ---
// password struct to hold both plaintext (optional) and hash
type password struct {
	plaintext *string // Pointer to distinguish between unset and empty
	hash      []byte
}

// Set calculates the bcrypt hash of a plaintext password and stores it.
func (p *password) Set(plaintextPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintextPassword), 12)
	if err != nil {
		return err
	}
	p.plaintext = &plaintextPassword // Store plaintext pointer (might be useful for debugging/confirmation)
	p.hash = hash                    // Store the hash
	return nil
}

// Matches checks if a provided plaintext password matches the stored hash.
func (p *password) Matches(plaintextPassword string) (bool, error) {
	err := bcrypt.CompareHashAndPassword(p.hash, []byte(plaintextPassword))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return false, nil // Passwords don't match
		}
		return false, err // Other bcrypt error
	}
	return true, nil // Passwords match
}

// --- User Struct Definition ---
type User struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Password  password  `json:"-"` // Use the custom password type, hide from JSON
	CreatedAt time.Time `json:"created_at"`
	Active    bool      `json:"active"` // Keep your 'active' field
}

// --- User Validation ---
// Keep your existing validation function for name and email
func ValidateUser(v *validator.Validator, u *User) {
	v.Check(validator.NotBlank(u.Name), "name", "must be provided")
	v.Check(validator.MaxLength(u.Name, 255), "name", "must not be more than 255 characters")

	v.Check(validator.NotBlank(u.Email), "email", "must be provided")
	v.Check(validator.MaxLength(u.Email, 255), "email", "must not be more than 255 characters")
	v.Check(validator.Matches(u.Email, validator.EmailRX), "email", "must be a valid email address")
}

// --- UserModel Struct ---
type UserModel struct {
	DB *sql.DB
}

// --- UserModel Methods ---

// Insert a new user record. Assumes Password.Set has been called.
func (m *UserModel) Insert(user *User) error {
	// Match the column name from your schema diagram: 'password_hash'
	// Add the 'activated' column based on the example's logic.
	query := `
		INSERT INTO users (name, email, password_hash, activated)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`

	args := []any{
		user.Name,
		user.Email,
		user.Password.hash, // Use the hash from the password struct
		user.Active,        // Use the Active field
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, args...).Scan(&user.ID, &user.CreatedAt)
	if err != nil {
		if err.Error() != "" && strings.Contains(err.Error(), `duplicate key value violates unique constraint "users_email_key"`) {
			return ErrDuplicateEmail
		}
		return err
	}
	return nil
}

// Authenticate checks email and password for an *active* user.
func (m *UserModel) Authenticate(email, plaintextPassword string) (int64, error) {
	var id int64
	var dbPasswordHash []byte // Variable to store the hash from the DB

	query := `
		SELECT id, password_hash FROM users
		WHERE email = $1 AND activated = TRUE`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, email).Scan(&id, &dbPasswordHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrInvalidCredentials // User not found or not active
		}
		return 0, err // Other DB error
	}

	// Compare the provided plaintext password with the stored hash from the DB.
	err = bcrypt.CompareHashAndPassword(dbPasswordHash, []byte(plaintextPassword))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return 0, ErrInvalidCredentials // Password doesn't match
		}
		return 0, err // Other bcrypt error
	}

	// Authentication successful
	return id, nil
}

// Get retrieves a specific user by ID.
func (m *UserModel) Get(id int64) (*User, error) { // Renamed from GetUser to Get
	if id < 1 {
		return nil, ErrRecordNotFound
	}

	query := `
		SELECT id, name, email, created_at, password_hash, activated
		FROM users
		WHERE id = $1`

	var user User

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Scan the results, including the hash into user.Password.hash
	err := m.DB.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.CreatedAt,
		&user.Password.hash, // Scan directly into the hash field
		&user.Active,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRecordNotFound
		}
		return nil, err
	}
	return &user, nil
}

// GetByEmail retrieves a specific user by Email. (Added from example)
func (m *UserModel) GetByEmail(email string) (*User, error) {
	query := `
        SELECT id, name, email, created_at, password_hash, activated
        FROM users
        WHERE email = $1`

	var user User

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.CreatedAt,
		&user.Password.hash,
		&user.Active,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRecordNotFound
		}
		return nil, err
	}

	return &user, nil
}

// Update user details. (Added from example)
func (m *UserModel) Update(user *User) error {
	// Ensure column names ('password_hash', 'activated') match your schema.
	query := `
        UPDATE users
        SET name = $1, email = $2, password_hash = $3, activated = $4
        WHERE id = $5
        RETURNING id` // RETURNING helps confirm the update happened

	args := []any{
		user.Name,
		user.Email,
		user.Password.hash, // Assumes hash is updated if password was changed via user.Password.Set()
		user.Active,
		user.ID,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var returnedID int64 // Variable to scan the returned ID into
	err := m.DB.QueryRowContext(ctx, query, args...).Scan(&returnedID)
	if err != nil {
		// Check for duplicate email error on update
		// *** Use the same constraint name check as in Insert ***
		if strings.Contains(err.Error(), `violates unique constraint "users_email_key"`) ||
			strings.Contains(err.Error(), `violates unique constraint "users_email_idx"`) {
			return ErrDuplicateEmail
		}
		// If RETURNING doesn't find a row (e.g., wrong ID), QueryRowContext returns ErrNoRows
		if errors.Is(err, sql.ErrNoRows) {
			return ErrRecordNotFound // Indicate the user to update wasn't found
		}
		return err
	}

	// Optional check: ensure the returned ID matches the input ID
	if returnedID != user.ID {
		// This case should be rare if the WHERE clause worked, but good sanity check
		return errors.New("failed to update user: ID mismatch")
	}

	return nil
}
