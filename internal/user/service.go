package user

import (
	"database/sql"
	"time"

	"woodpecker-online/internal/auth"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Service struct {
	db *sqlx.DB
}

func NewService(db *sqlx.DB) *Service {
	return &Service{db: db}
}

// CreateUser creates a new user
func (s *Service) CreateUser(email, password string) (*auth.User, error) {
	// Check if user already exists
	var existingUser auth.User
	err := s.db.Get(&existingUser, "SELECT id FROM users WHERE email = ?", email)
	if err == nil {
		return nil, auth.ErrUserExists
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	// Hash password
	hashedPassword, err := auth.HashPassword(password)
	if err != nil {
		return nil, err
	}

	// Create user
	user := &auth.User{
		ID:           uuid.New().String(),
		Email:        email,
		PasswordHash: hashedPassword,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	_, err = s.db.Exec(`
		INSERT INTO users (id, email, password_hash, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`, user.ID, user.Email, user.PasswordHash, user.CreatedAt, user.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return user, nil
}

// GetUserByEmail retrieves a user by email
func (s *Service) GetUserByEmail(email string) (*auth.User, error) {
	var user auth.User
	err := s.db.Get(&user, "SELECT * FROM users WHERE email = ?", email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, auth.ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// GetUserByID retrieves a user by ID
func (s *Service) GetUserByID(id string) (*auth.User, error) {
	var user auth.User
	err := s.db.Get(&user, "SELECT * FROM users WHERE id = ?", id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, auth.ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// ValidateCredentials validates user credentials
func (s *Service) ValidateCredentials(email, password string) (*auth.User, error) {
	user, err := s.GetUserByEmail(email)
	if err != nil {
		return nil, auth.ErrInvalidCredentials
	}

	if !auth.CheckPasswordHash(password, user.PasswordHash) {
		return nil, auth.ErrInvalidCredentials
	}

	return user, nil
}
