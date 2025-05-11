// internal/userservice/repository/user_repository.go
package repository

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"microservices-project/internal/database" // Our database package
	"microservices-project/internal/userservice/model"
	"time"

	"github.com/google/uuid" // For generating UUIDs if not handled by DB default
)

// ErrUserNotFound is returned when a user is not found.
var ErrUserNotFound = errors.New("user not found")

// UserRepositoryInterface defines the operations for user data storage.
type UserRepositoryInterface interface {
	CreateUser(ctx context.Context, user *model.User) (*model.User, error)
	GetUserByID(ctx context.Context, id string) (*model.User, error)
	GetUserByEmail(ctx context.Context, email string) (*model.User, error)
	// UpdateUser, DeleteUser, etc. can be added later
}

// UserRepository implements UserRepositoryInterface using PostgreSQL.
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository creates a new UserRepository.
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// CreateUser inserts a new user into the database.
func (r *UserRepository) CreateUser(ctx context.Context, user *model.User) (*model.User, error) {
	// Generate UUID for new user if ID is not set or if DB doesn't do it automatically
	if user.ID == "" {
		user.ID = uuid.New().String()
	}
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	query := `INSERT INTO users (id, username, email, password_hash, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6)
	          RETURNING id, created_at, updated_at` // Return DB generated values if any

	// Use database.DB directly (or pass it in NewUserRepository)
	// For this example, we'll assume database.DB is initialized
	err := database.DB.QueryRowContext(ctx, query,
		user.ID, user.Username, user.Email, user.PasswordHash, user.CreatedAt, user.UpdatedAt,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt) // Update with any values returned by RETURNING

	if err != nil {
		// TODO: Check for specific DB errors like unique constraint violation
		log.Printf("Error creating user in DB: %v", err)
		return nil, err
	}
	return user, nil
}

// GetUserByID retrieves a user by their ID.
func (r *UserRepository) GetUserByID(ctx context.Context, id string) (*model.User, error) {
	user := &model.User{}
	query := `SELECT id, username, email, password_hash, created_at, updated_at
	          FROM users WHERE id = $1`

	err := database.DB.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		log.Printf("Error getting user by ID from DB: %v", err)
		return nil, err
	}
	return user, nil
}

// GetUserByEmail retrieves a user by their email.
func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	user := &model.User{}
	query := `SELECT id, username, email, password_hash, created_at, updated_at
	          FROM users WHERE email = $1`

	err := database.DB.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		log.Printf("Error getting user by email from DB: %v", err)
		return nil, err
	}
	return user, nil
}