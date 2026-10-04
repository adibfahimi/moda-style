package common

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// DatabaseConfig holds the parameters needed to reach the shared PostgreSQL
// database. It mirrors the DB_* environment variables read by
// GetDatabaseConfig.
type DatabaseConfig struct {
	// Host is the database server hostname (DB_HOST).
	Host string
	// Port is the database server port (DB_PORT).
	Port string
	// User is the login role (DB_USER).
	User string
	// Password is the login password (DB_PASSWORD).
	Password string
	// DBName is the database/schema name (DB_NAME).
	DBName string
	// SSLMode is the libpq sslmode, e.g. "disable" or "require" (DB_SSL_MODE).
	SSLMode string
}

// GetDatabaseConfig assembles a DatabaseConfig from environment variables.
//
// DB_HOST, DB_USER, DB_PASSWORD and DB_NAME are required and cause a fatal
// error when missing. DB_PORT defaults to "5432" and DB_SSL_MODE to "disable".
func GetDatabaseConfig() DatabaseConfig {
	return DatabaseConfig{
		Host:     getEnvOrFatal("DB_HOST"),
		Port:     getEnvOrDefault("DB_PORT", "5432"),
		User:     getEnvOrFatal("DB_USER"),
		Password: getEnvOrFatal("DB_PASSWORD"),
		DBName:   getEnvOrFatal("DB_NAME"),
		SSLMode:  getEnvOrDefault("DB_SSL_MODE", "disable"),
	}
}

// ConnectDatabase opens a GORM connection to PostgreSQL using the supplied
// configuration.
//
// The returned error is wrapped with context so callers can log it directly.
func ConnectDatabase(config DatabaseConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.DBName, config.SSLMode)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Println("Database connected successfully")
	return db, nil
}

// MustConnectDatabase connects to the database and terminates the process if the
// connection cannot be established. It is intended for use during service
// startup, where a missing database is fatal.
func MustConnectDatabase(config DatabaseConfig) *gorm.DB {
	db, err := ConnectDatabase(config)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	return db
}

// getEnvOrDefault returns the value of the environment variable named by key,
// or fallback when it is unset (or empty).
func getEnvOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// getEnvOrFatal returns the value of the required environment variable named by
// key, terminating the process with a fatal error when it is unset.
func getEnvOrFatal(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("%s environment variable is required", key)
	}
	return value
}
