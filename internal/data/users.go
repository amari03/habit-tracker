package data

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/amari03/habit-tracker/internal/validator"
)

var(
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrDuplicateEmail = errors.New("duplicate email")
)

type User struct {
	ID        int64     `json:"id"`
	Name	  string    `json:"name"`
	Email     string    `json:"email"`
	HashedPassword  []byte
	CreatedAt  time.Time `json:"created_at"`
	Active     bool      `json:"active"`
}

func ValidateUser(v *validator.Validator, u *User) {
	v.Check(validator.NotBlank(u.Name), "name", "must be provided")
	v.Check(validator.MaxLength(u.Name, 255), "name", "must not be more than 255 characters")

	v.Check(validator.NotBlank(u.Email), "email", "must be provided")
	v.Check(validator.MaxLength(u.Email, 255), "email", "must not be more than 255 characters")
	v.Check(validator.Matches(u.Email, validator.EmailRX), "email", "must be a valid email address")

	v.Check(validator.NotBlank(string(u.HashedPassword)), "password", "must be provided")
	v.Check(validator.MinLength(string(u.HashedPassword), 8), "password", "must be at least 8 bytes long")
	v.Check(validator.MaxLength(string(u.HashedPassword), 72), "password", "must not be more than 72 bytes long")
}

type UserModel struct {
	DB *sql.DB
}

func (m *UserModel) Insert(user *User) error {
	query := `
		INSERT INTO users (name, email, hashed_password)
		VALUES ($1, $2, $3)
		RETURNING id, created_at`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return m.DB.QueryRowContext(ctx, query,
		user.Name,
		user.Email,
		user.HashedPassword,
	).Scan(&user.ID, &user.CreatedAt)
}

func (m *UserModel) Authenticate(email, password string) (int, error) {
	query := `
		SELECT id
		FROM users
		WHERE email = $1 AND hashed_password = $2`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var id int
	err := m.DB.QueryRowContext(ctx, query, email, password).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, ErrInvalidCredentials
		}
		return 0, err
	}

	return id, nil
}

//check back with this function
func (m *UserModel) GetUser(id int64) (*User, error) {
	query := `
		SELECT id, name, email, created_at
		FROM users
		WHERE id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var user User
	err := m.DB.QueryRowContext(ctx, query, id).Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	return &user, nil
}