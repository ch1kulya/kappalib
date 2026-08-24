package main

import (
	"context"
	"os"

	"github.com/ch1kulya/kappalib/internal/database"
	"github.com/ch1kulya/kappalib/mock"
	"github.com/ch1kulya/logger"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load(".env.dev")

	if os.Getenv("FORCE_COLOR") == "1" {
		logger.SetForceColor(true)
	}

	databaseURL := os.Getenv("DEV_DATABASE_URL")
	if databaseURL == "" {
		logger.Fatal("DEV_DATABASE_URL environment variable is required")
	}

	logger.Info("Checking database migrations...")
	if err := database.RunMigrations(databaseURL, "file://migrations"); err != nil {
		logger.Fatal("Database migrations failed: %v", err)
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		logger.Fatal("Failed to connect to database: %v", err)
	}
	defer conn.Close(ctx)

	logger.Info("Seeding mock database...")
	if _, err := conn.Exec(ctx, mock.SeedSQL); err != nil {
		logger.Fatal("Failed to execute seed SQL: %v", err)
	}

	logger.Info("Mock database seeded successfully")
}
