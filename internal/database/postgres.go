// internal/database/postgres.go
package database

import (
	"database/sql"
	"fmt"
	"log"
	"os" // For environment variables

	"github.com/joho/godotenv"
	_ "github.com/lib/pq" // PostgreSQL driver
)

// DB is a global database connection pool.
// In a real app, you might pass this around or use a more sophisticated DI approach.
var DB *sql.DB

// ConnectDB initializes the database connection.
func ConnectDB() error {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
        log.Println("No .env file found")
    }
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbSSLMode := os.Getenv("DB_SSLMODE") 

	if dbHost == "" {
		dbHost = "localhost"
	}
	if dbPort == "" {
		dbPort = "5432"
	}
	if dbUser == "" {
		dbUser = "user" // Default, set via env
	}
	if dbPassword == "" {
		dbPassword = "password" // Default, set via env
	}
	if dbName == "" {
		dbName = "user_service_db" // Default, set via env
	}
	if dbSSLMode == "" {
		dbSSLMode = "disable"
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)

	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}

	err = DB.Ping()
	if err != nil {
		DB.Close() // Close the connection if ping fails
		return fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Successfully connected to PostgreSQL database!")
	return nil
}

// CloseDB closes the database connection.
func CloseDB() {
	if DB != nil {
		DB.Close()
		log.Println("Database connection closed.")
	}
}