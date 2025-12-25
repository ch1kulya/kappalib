package database

import (
	"context"
	"os"
	"time"

	"github.com/ch1kulya/logger"
	"github.com/jackc/pgx/v5/pgxpool"
)

var DB *pgxpool.Pool

func Init() error {
	logger.Info("Connecting to database...")

	dbConfig, err := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		return err
	}

	dbConfig.MaxConns = 25
	dbConfig.MinConns = 5
	dbConfig.MaxConnLifetime = time.Hour
	dbConfig.MaxConnIdleTime = 30 * time.Minute

	DB, err = pgxpool.NewWithConfig(context.Background(), dbConfig)
	if err != nil {
		return err
	}

	if err := DB.Ping(context.Background()); err != nil {
		return err
	}

	logger.Info("Database connected successfully")
	return nil
}

func Close() {
	if DB != nil {
		DB.Close()
	}
}
