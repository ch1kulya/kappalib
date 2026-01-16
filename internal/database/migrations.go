package database

import (
	"errors"

	"github.com/ch1kulya/logger"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func RunMigrations(databaseURL, migrationsPath string) error {
	logger.Info("Starting database migrations...")

	m, err := migrate.New(migrationsPath, databaseURL)
	if err != nil {
		logger.Error("Migration initialization failed: %v", err)
		return err
	}

	defer func() {
		srcErr, dbErr := m.Close()
		if srcErr != nil {
			logger.Warn("Migration source close error: %v", srcErr)
		}
		if dbErr != nil {
			logger.Warn("Migration db close error: %v", dbErr)
		}
	}()

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			logger.Info("Already up to date.")
			return nil
		}
		logger.Error("Migration failed: %v", err)
		return err
	}

	logger.Info("Migrations applied successfully")
	return nil
}
