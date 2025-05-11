// internal/userservice/service/user_service_test.go
package service

import (
	"context"
	"errors"
	"microservices-project/internal/userservice/model"
	"microservices-project/internal/userservice/repository" // For ErrUserNotFound
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockUserRepository is a mock type for the UserRepositoryInterface
type MockUserRepository struct {
	mock.Mock
}

// Implementing UserRepositoryInterface for the mock
func (m *MockUserRepository) CreateUser(ctx context.Context, user *model.User) (*model.User, error) {
	args := m.Called(ctx, user)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepository) GetUserByID(ctx context.Context, id string) (*model.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepository) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}


func TestUserService_CreateUser_Success(t *testing.T) {
	mockRepo := new(MockUserRepository)
	userService := NewUserService(mockRepo)

	username := "newuser"
	email := "new@example.com"
	password := "password123"

	// Expected user after hashing password and before DB insert
	// The ID, CreatedAt, UpdatedAt will be set by the repo or DB
	expectedUserAfterRepo := &model.User{
		ID:        "some-uuid",
		Username:  username,
		Email:     email,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		// PasswordHash is internal to the createdUser object by repo
	}

	// Mock GetUserByEmail to return ErrUserNotFound (user doesn't exist)
	mockRepo.On("GetUserByEmail", mock.Anything, email).Return(nil, repository.ErrUserNotFound)

	// Mock CreateUser
	// We need to match on a user object where PasswordHash is set.
	// The actual value of PasswordHash is not easily predictable without running HashPassword.
	// So, we use mock.MatchedBy to check the fields we care about.
	mockRepo.On("CreateUser", mock.Anything, mock.MatchedBy(func(user *model.User) bool {
		return user.Username == username && user.Email == email && user.PasswordHash != ""
	})).Return(expectedUserAfterRepo, nil)

	createdUser, err := userService.CreateUser(context.Background(), username, email, password)

	assert.NoError(t, err)
	assert.NotNil(t, createdUser)
	assert.Equal(t, expectedUserAfterRepo.ID, createdUser.ID)
	assert.Equal(t, username, createdUser.Username)
	assert.Equal(t, email, createdUser.Email)

	mockRepo.AssertExpectations(t)
}

func TestUserService_CreateUser_AlreadyExists(t *testing.T) {
	mockRepo := new(MockUserRepository)
	userService := NewUserService(mockRepo)

	email := "existing@example.com"
	existingUser := &model.User{Email: email}

	// Mock GetUserByEmail to return an existing user
	mockRepo.On("GetUserByEmail", mock.Anything, email).Return(existingUser, nil)

	_, err := userService.CreateUser(context.Background(), "user", email, "pass")

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrUserAlreadyExists))
	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "CreateUser", mock.Anything, mock.AnythingOfType("*model.User"))
}


func TestUserService_GetUserByID_Success(t *testing.T) {
	mockRepo := new(MockUserRepository)
	userService := NewUserService(mockRepo)

	userID := "user123"
	expectedUser := &model.User{ID: userID, Username: "test"}

	mockRepo.On("GetUserByID", mock.Anything, userID).Return(expectedUser, nil)

	user, err := userService.GetUserByID(context.Background(), userID)

	assert.NoError(t, err)
	assert.Equal(t, expectedUser, user)
	mockRepo.AssertExpectations(t)
}

func TestUserService_GetUserByID_NotFound(t *testing.T) {
	mockRepo := new(MockUserRepository)
	userService := NewUserService(mockRepo)

	userID := "nonexistent"
	mockRepo.On("GetUserByID", mock.Anything, userID).Return(nil, repository.ErrUserNotFound)

	_, err := userService.GetUserByID(context.Background(), userID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrUserNotFound)) // Check if it's the specific error
	mockRepo.AssertExpectations(t)
}

func TestUserService_LoginUser_Success(t *testing.T) {
	mockRepo := new(MockUserRepository)
	userService := NewUserService(mockRepo)

	email := "login@example.com"
	password := "password123"
	hashedPassword, _ := HashPassword(password) // Hash it for the mock
	
	dbUser := &model.User{
		ID: "user-login-id",
		Email: email,
		PasswordHash: hashedPassword,
		Username: "loginuser",
	}

	mockRepo.On("GetUserByEmail", mock.Anything, email).Return(dbUser, nil)

	user, token, err := userService.LoginUser(context.Background(), email, password)

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, dbUser.ID, user.ID)
	assert.NotEmpty(t, token)
	assert.Contains(t, token, "mock-jwt-token-for-") // Check our mock token format
	mockRepo.AssertExpectations(t)
}

func TestUserService_LoginUser_WrongPassword(t *testing.T) {
	mockRepo := new(MockUserRepository)
	userService := NewUserService(mockRepo)

	email := "login@example.com"
	correctPassword := "password123"
	wrongPassword := "wrongpass"
	hashedPassword, _ := HashPassword(correctPassword)
	
	dbUser := &model.User{Email: email, PasswordHash: hashedPassword}

	mockRepo.On("GetUserByEmail", mock.Anything, email).Return(dbUser, nil)

	_, _, err := userService.LoginUser(context.Background(), email, wrongPassword)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidCredentials))
	mockRepo.AssertExpectations(t)
}

func TestUserService_LoginUser_UserNotFound(t *testing.T) {
	mockRepo := new(MockUserRepository)
	userService := NewUserService(mockRepo)
	email := "nonexistent@example.com"

	mockRepo.On("GetUserByEmail", mock.Anything, email).Return(nil, repository.ErrUserNotFound)

	_, _, err := userService.LoginUser(context.Background(), email, "anypassword")
	
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidCredentials)) // Service maps UserNotFound to InvalidCredentials for login
	mockRepo.AssertExpectations(t)
}