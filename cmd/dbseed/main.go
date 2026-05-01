package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"lms-arvand-backend/internal/bootstrap"
	"lms-arvand-backend/internal/config"
	"lms-arvand-backend/internal/repository/postgres"
	"lms-arvand-backend/internal/usecase"
)

func main() {
	if err := run(); err != nil {
		logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
		logger.Error("dbseed exited with error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run() error {
	if err := config.LoadEnvFile(".env"); err != nil {
		return fmt.Errorf("dbseed load env file: %w", err)
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("dbseed load config: %w", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	dbPool, err := postgres.NewPool(ctx, cfg.Database)
	if err != nil {
		return fmt.Errorf("dbseed initialize postgres pool: %w", err)
	}
	defer dbPool.Close()

	userRepository := postgres.NewUserRepository(dbPool)
	userUseCase := usecase.NewUserUseCase(userRepository, cfg.Auth.BcryptCost)
	adminSeeder := bootstrap.NewAdminSeeder(userRepository, userUseCase)

	adminUser, err := adminSeeder.Seed(ctx, cfg.Seed.Admin)
	if err != nil {
		return fmt.Errorf("dbseed admin seed: %w", err)
	}

	logger.Info("admin seed completed",
		slog.String("user_id", adminUser.ID),
		slog.String("username", adminUser.Username),
		slog.String("email", cfg.Seed.Admin.Email),
	)

	return nil
}
