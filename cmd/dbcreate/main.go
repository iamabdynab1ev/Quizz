package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"lms-arvand-backend/internal/config"
	"lms-arvand-backend/internal/repository/postgres"
)

func main() {
	if err := run(); err != nil {
		logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
		logger.Error("dbcreate exited with error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run() error {
	if err := config.LoadEnvFile(".env"); err != nil {
		return fmt.Errorf("dbcreate load env file: %w", err)
	}

	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		return fmt.Errorf("dbcreate database url is empty")
	}

	targetDatabase, err := config.DatabaseNameFromURL(databaseURL)
	if err != nil {
		return fmt.Errorf("dbcreate parse target database name: %w", err)
	}

	adminDatabaseURL, err := config.DatabaseAdminURL(databaseURL, "postgres")
	if err != nil {
		return fmt.Errorf("dbcreate build admin database url: %w", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := postgres.NewPool(ctx, config.DatabaseConfig{
		URL:               adminDatabaseURL,
		MaxConns:          2,
		MinConns:          0,
		MaxConnLifetime:   5 * time.Minute,
		MaxConnIdleTime:   time.Minute,
		HealthCheckPeriod: time.Minute,
	})
	if err != nil {
		return fmt.Errorf("dbcreate initialize postgres pool: %w", err)
	}
	defer pool.Close()

	exists, err := databaseExists(ctx, pool, targetDatabase)
	if err != nil {
		return fmt.Errorf("dbcreate check database exists: %w", err)
	}

	if exists {
		logger.Info("database already exists", slog.String("database", targetDatabase))
		return nil
	}

	query := "CREATE DATABASE " + quoteIdentifier(targetDatabase)
	if _, err := pool.Exec(ctx, query); err != nil {
		return fmt.Errorf("dbcreate create database %s: %w", targetDatabase, err)
	}

	logger.Info("database created", slog.String("database", targetDatabase))

	return nil
}

type dbExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func databaseExists(ctx context.Context, pool dbExecutor, databaseName string) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1
			FROM pg_database
			WHERE datname = $1
		)
	`

	var exists bool
	if err := pool.QueryRow(ctx, query, databaseName).Scan(&exists); err != nil {
		return false, fmt.Errorf("dbcreate query database exists: %w", err)
	}

	return exists, nil
}

func quoteIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}
