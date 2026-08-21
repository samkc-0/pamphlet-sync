// Package db handles the database connection and migrations.
package db

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/samkc-0/pamphlet-sync/internal/models"
)

// Connect opens a GORM connection to Postgres using the given DSN.
func Connect(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, err
	}
	return db, nil
}

// AutoMigrate runs GORM auto-migration for all registered models.
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(models.All()...)
}
