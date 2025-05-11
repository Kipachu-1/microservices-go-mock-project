// internal/userservice/service/user_service.go
package service

import (
	"context"
	"errors"
	"log"
	"microservices-project/internal/userservice/model"
	"microservices-project/internal/userservice/repository"

	"golang.org/x/crypto/bcrypt" // For password hashing
)

// Custom errors for the service layer
var (
	ErrUserAlreadyExists = errors.New("user with this email or username already exists")
	ErrUserNotFound      = repository.ErrUserNotFound // Propagate repository error
	ErrInvalidCredentials = errors.New("invalid email or password")
)


// UserServiceInterface defines the business logic operations for users.
type UserServiceInterface interface {
	CreateUser(ctx context.Context, username, email, password string) (*model.User, error)
	GetUserByID(ctx context.Context, id string) (*model.User, error)
	LoginUser(ctx context.Context, email, password string) (*model.User, string, error) // Returns user and token
}

// UserService implements UserServiceInterface.
type UserService struct {
	repo repository.UserRepositoryInterface // Dependency on the repository
}

// NewUserService creates a new UserService.
func NewUserService(repo repository.UserRepositoryInterface) *UserService {
	return &UserService{repo: repo}
}

// HashPassword generates a bcrypt hash of the password.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPasswordHash compares a plain text password with a bcrypt hash.
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// CreateUser creates a new user after validating and hashing the password.
func (s *UserService) CreateUser(ctx context.Context, username, email, password string) (*model.User, error) {
	// Check if user already exists by email (can add username check too)
	_, err := s.repo.GetUserByEmail(ctx, email)
	if err == nil { // User found, so already exists
		return nil, ErrUserAlreadyExists
	}
	if err != ErrUserNotFound { // Some other DB error
		log.Printf("Error checking for existing user by email: %v", err)
		return nil, err
	}
	// User not found by email, proceed. Add username check if needed.

	hashedPassword, err := HashPassword(password)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		return nil, errors.New("failed to process password")
	}

	user := &model.User{
		Username:     username,
		Email:        email,
		PasswordHash: hashedPassword,
	}

	createdUser, err := s.repo.CreateUser(ctx, user)
	if err != nil {
		// Potentially map DB specific errors (like unique constraint violation on username if not checked above)
		log.Printf("Error creating user in service: %v", err)
		return nil, err
	}
	return createdUser, nil
}

// GetUserByID retrieves a user by their ID.
func (s *UserService) GetUserByID(ctx context.Context, id string) (*model.User, error) {
	user, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		if err == ErrUserNotFound {
			return nil, ErrUserNotFound
		}
		log.Printf("Error getting user by ID in service: %v", err)
		return nil, err // Or a more generic service error
	}
	return user, nil
}

// LoginUser authenticates a user and returns the user and a token (placeholder for token).
func (s *UserService) LoginUser(ctx context.Context, email, password string) (*model.User, string, error) {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if err == ErrUserNotFound {
			return nil, "", ErrInvalidCredentials
		}
		log.Printf("Error during login (GetUserByEmail): %v", err)
		return nil, "", err
	}

	if !CheckPasswordHash(password, user.PasswordHash) {
		return nil, "", ErrInvalidCredentials
	}

	// TODO: Generate JWT token here
	token := "mock-jwt-token-for-" + user.ID
	log.Printf("User %s logged in successfully", user.Email)

	return user, token, nil
}