// internal/userservice/repository/user_repository_test.go
package repository

import (
	"context"
	"database/sql"
	"errors"
	"regexp" // For matching SQL queries
	"testing"
	"time"

	"microservices-project/internal/userservice/model"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create a mock DB and repository
func newMockDBAndRepo(t *testing.T) (*sql.DB, sqlmock.Sqlmock, *UserRepository) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	repo := NewUserRepository(db)
	return db, mock, repo
}

func TestUserRepository_CreateUser(t *testing.T) {
	db, mock, repo := newMockDBAndRepo(t)
	defer db.Close()

	userID := uuid.New().String()
	now := time.Now()
	userToCreate := &model.User{
		// ID will be set by the repo if empty
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hashedpassword",
	}

	// We expect an INSERT query. Use regexp.QuoteMeta for fixed parts of the query.
	// The actual query will have placeholders like $1, $2, etc.
	// sqlmock expects the exact query string or a regex.
	expectedSQL := regexp.QuoteMeta(`INSERT INTO users (id, username, email, password_hash, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6)
	          RETURNING id, created_at, updated_at`)

	mock.ExpectQuery(expectedSQL). // ExpectQuery for QueryRowContext
					WithArgs(sqlmock.AnyArg(), userToCreate.Username, userToCreate.Email, userToCreate.PasswordHash, sqlmock.AnyArg(), sqlmock.AnyArg()). // Match arguments
					WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
						AddRow(userID, now, now)) // Values returned by RETURNING

	createdUser, err := repo.CreateUser(context.Background(), userToCreate)

	assert.NoError(t, err)
	require.NotNil(t, createdUser)
	assert.Equal(t, userID, createdUser.ID) // ID should be the one returned
	assert.Equal(t, userToCreate.Username, createdUser.Username)
	assert.Equal(t, userToCreate.Email, createdUser.Email)
	assert.WithinDuration(t, now, createdUser.CreatedAt, time.Second) // Check time is close
	assert.WithinDuration(t, now, createdUser.UpdatedAt, time.Second)

	// Ensure all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_GetUserByID_Found(t *testing.T) {
	db, mock, repo := newMockDBAndRepo(t)
	defer db.Close()

	userID := uuid.New().String()
	now := time.Now()
	expectedUser := &model.User{
		ID:           userID,
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hashedpassword",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	expectedSQL := regexp.QuoteMeta(`SELECT id, username, email, password_hash, created_at, updated_at
	          FROM users WHERE id = $1`)

	rows := sqlmock.NewRows([]string{"id", "username", "email", "password_hash", "created_at", "updated_at"}).
		AddRow(expectedUser.ID, expectedUser.Username, expectedUser.Email, expectedUser.PasswordHash, expectedUser.CreatedAt, expectedUser.UpdatedAt)

	mock.ExpectQuery(expectedSQL).WithArgs(userID).WillReturnRows(rows)

	user, err := repo.GetUserByID(context.Background(), userID)

	assert.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, expectedUser, user)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_GetUserByID_NotFound(t *testing.T) {
	db, mock, repo := newMockDBAndRepo(t)
	defer db.Close()

	userID := uuid.New().String()
	expectedSQL := regexp.QuoteMeta(`SELECT id, username, email, password_hash, created_at, updated_at
	          FROM users WHERE id = $1`)

	mock.ExpectQuery(expectedSQL).WithArgs(userID).WillReturnError(sql.ErrNoRows)

	user, err := repo.GetUserByID(context.Background(), userID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrUserNotFound))
	assert.Nil(t, user)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_GetUserByEmail_Found(t *testing.T) {
	db, mock, repo := newMockDBAndRepo(t)
	defer db.Close()

	email := "test@example.com"
	now := time.Now()
	expectedUser := &model.User{
		ID:           uuid.NewString(),
		Username:     "testuser",
		Email:        email,
		PasswordHash: "hashedpassword",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	expectedSQL := regexp.QuoteMeta(`SELECT id, username, email, password_hash, created_at, updated_at
	          FROM users WHERE email = $1`)

	rows := sqlmock.NewRows([]string{"id", "username", "email", "password_hash", "created_at", "updated_at"}).
		AddRow(expectedUser.ID, expectedUser.Username, expectedUser.Email, expectedUser.PasswordHash, expectedUser.CreatedAt, expectedUser.UpdatedAt)

	mock.ExpectQuery(expectedSQL).WithArgs(email).WillReturnRows(rows)

	user, err := repo.GetUserByEmail(context.Background(), email)

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, expectedUser, user)
	assert.NoError(t, mock.ExpectationsWereMet())
}