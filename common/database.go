package common

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// DatabaseConfig holds database connection configuration
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// GetDatabaseConfig reads database configuration from environment variables
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

// ConnectDatabase establishes a connection to PostgreSQL database
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

// MustConnectDatabase connects to database or exits if connection fails
func MustConnectDatabase(config DatabaseConfig) *gorm.DB {
	db, err := ConnectDatabase(config)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	return db
}

func getEnvOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvOrFatal(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("%s environment variable is required", key)
	}
	return value
}
