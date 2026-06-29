package database

import (
	"fmt"
	"log"

	"github.com/f0bima/go-core/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/plugin/opentelemetry/tracing"
)

// NewPostgresDB creates a new GORM database connection with OpenTelemetry tracing plugin.
func NewPostgresDB(cfg *config.Config) *gorm.DB {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
		cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBPort)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.Use(tracing.NewPlugin(tracing.WithoutQueryVariables())); err != nil {
		log.Fatalf("Failed to initialize otel plugin for gorm: %v", err)
	}

	return db
}
