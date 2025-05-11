// internal/userservice/model/user.go
package model

import "time"

// User represents the domain model for a user.
// This is separate from the protobuf User message to allow for domain-specific fields
// or different representations (e.g., password hash is here, not in proto).
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // "-" means don't include in JSON if marshaled directly
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}