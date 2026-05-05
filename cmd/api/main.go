package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"lms-arvand-backend/internal/bootstrap"
	"lms-arvand-backend/internal/cache"
	"lms-arvand-backend/internal/config"
	httpHandler "lms-arvand-backend/internal/handler/http"
	httpMiddleware "lms-arvand-backend/internal/handler/http/middleware"
	"lms-arvand-backend/internal/repository/postgres"
	"lms-arvand-backend/internal/storage"
	"lms-arvand-backend/internal/usecase"
)

type runtimeFlags struct {
	migrate   bool
	seedAdmin bool
	all       bool
}

func main() {
	if err := run(); err != nil {
		logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
		logger.Error("api exited with error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run() error {
	flags := parseFlags()

	if err := loadEnvFiles(); err != nil {
		return fmt.Errorf("main load env file: %w", err)
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("main load config: %w", err)
	}

	logger, err := newLogger(cfg.App.LogLevel)
	if err != nil {
		return fmt.Errorf("main create logger: %w", err)
	}

	if err := os.MkdirAll(cfg.Upload.Dir, 0o755); err != nil {
		return fmt.Errorf("main prepare uploads dir: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dbPool, err := postgres.NewPool(ctx, cfg.Database)
	if err != nil {
		return fmt.Errorf("main initialize postgres pool: %w", err)
	}
	defer dbPool.Close()

	migrateRequested, seedRequested := resolveOperationFlags(flags, cfg)

	if migrateRequested {
		migrator := bootstrap.NewMigrator(dbPool)
		if err := migrator.Up(ctx, cfg.Migrate.Dir); err != nil {
			return fmt.Errorf("main migrate up: %w", err)
		}

		logger.Info("database migrations applied", slog.String("dir", cfg.Migrate.Dir))
	}

	healthRepository := postgres.NewHealthRepository(dbPool)
	healthUseCase := usecase.NewHealthUseCase(cfg.App.Name, healthRepository)
	healthHandler := httpHandler.NewHealthHandler(logger, healthUseCase)

	auditLogRepository := postgres.NewAuditLogRepository(dbPool)
	auditLogger := usecase.NewAuditLogger(logger, auditLogRepository)

	userRepository := postgres.NewUserRepository(dbPool)
	userUseCase := usecase.NewUserUseCase(userRepository, cfg.Auth.BcryptCost).WithAudit(auditLogger)
	usersHandler := httpHandler.NewUsersHandler(logger, userUseCase)

	if seedRequested {
		adminSeeder := bootstrap.NewAdminSeeder(userRepository, userUseCase)
		adminUser, err := adminSeeder.Seed(ctx, cfg.Seed.Admin)
		if err != nil {
			return fmt.Errorf("main bootstrap admin seed: %w", err)
		}

		logger.Info("bootstrap admin seed completed",
			slog.String("user_id", adminUser.ID),
			slog.String("username", adminUser.Username),
		)
	}

	if flags.migrate || flags.seedAdmin || flags.all {
		return nil
	}

	sessionRepository := postgres.NewSessionRepository(dbPool)
	sessionCache := cache.NewSessionCache(cfg.Auth.SessionCacheTTL)
	loginAttemptRepository := postgres.NewLoginAttemptRepository(dbPool)
	googleVerifier := usecase.NewGoogleTokenInfoVerifier(cfg.Google.ClientID)
	authUseCase := usecase.NewAuthUseCase(
		userRepository,
		sessionRepository,
		sessionCache,
		cfg.Auth.SessionTTL,
		loginAttemptRepository,
		cfg.Auth.LoginMaxAttempts,
		cfg.Auth.LoginAttemptWindow,
		googleVerifier,
	)
	authHandler := httpHandler.NewAuthHandler(logger, authUseCase, cfg.Google.ClientID)

	courseRepository := postgres.NewCourseRepository(dbPool)
	courseUseCase := usecase.NewCourseUseCase(courseRepository).WithAudit(auditLogger)
	coursesHandler := httpHandler.NewCoursesHandler(logger, courseUseCase)

	quizRepository := postgres.NewQuizRepository(dbPool)
	quizUseCase := usecase.NewQuizUseCase(quizRepository).WithAudit(auditLogger)
	quizzesHandler := httpHandler.NewQuizzesHandler(logger, quizUseCase)

	attemptRepository := postgres.NewAttemptRepository(dbPool)
	attemptUseCase := usecase.NewAttemptUseCase(attemptRepository).WithAudit(auditLogger)
	attemptsHandler := httpHandler.NewAttemptsHandler(logger, attemptUseCase)

	enrollmentRepository := postgres.NewEnrollmentRepository(dbPool)
	enrollmentUseCase := usecase.NewEnrollmentUseCase(enrollmentRepository).WithAudit(auditLogger)
	enrollmentsHandler := httpHandler.NewEnrollmentsHandler(logger, enrollmentUseCase)

	certificateRepository := postgres.NewCertificateRepository(dbPool)
	certificateUseCase := usecase.NewCertificateUseCase(certificateRepository).WithAudit(auditLogger)
	certificatesHandler := httpHandler.NewCertificatesHandler(logger, certificateUseCase)
	enrollmentUseCase.WithCertificateAutoIssuer(certificateUseCase)
	attemptUseCase.WithEnrollmentLookup(enrollmentRepository).WithCertificateAutoIssuer(certificateUseCase)

	courseTestRepository := postgres.NewCourseTestRepository(dbPool)
	courseTestUseCase := usecase.NewCourseTestUseCase(courseTestRepository)
	courseTestsHandler := httpHandler.NewCourseTestsHandler(logger, courseTestUseCase)

	courseModuleRepository := postgres.NewCourseModuleRepository(dbPool)
	courseModuleUseCase := usecase.NewCourseModuleUseCase(courseModuleRepository)
	courseModulesHandler := httpHandler.NewCourseModulesHandler(logger, courseModuleUseCase)

	contentBlockRepository := postgres.NewContentBlockRepository(dbPool)
	contentBlockUseCase := usecase.NewContentBlockUseCase(contentBlockRepository)
	contentBlocksHandler := httpHandler.NewContentBlocksHandler(logger, contentBlockUseCase)

	reviewRepository := postgres.NewReviewRepository(dbPool)
	reviewUseCase := usecase.NewReviewUseCase(reviewRepository).WithAudit(auditLogger)
	reviewsHandler := httpHandler.NewReviewsHandler(logger, reviewUseCase)

	notificationRepository := postgres.NewNotificationRepository(dbPool)
	notificationUseCase := usecase.NewNotificationUseCase(notificationRepository)
	notificationsHandler := httpHandler.NewNotificationsHandler(logger, notificationUseCase)

	uploadStorage := storage.NewLocalFileStorage(cfg.Upload.Dir)
	uploadUseCase := usecase.NewUploadUseCase(uploadStorage, cfg.Upload.MaxSizeBytes)
	uploadsHandler := httpHandler.NewUploadsHandler(logger, uploadUseCase, cfg.Upload.MaxSizeBytes)

	webhookRepository := postgres.NewWebhookRepository(dbPool)
	webhookUseCase := usecase.NewWebhookUseCase(webhookRepository)
	webhooksHandler := httpHandler.NewWebhooksHandler(logger, webhookUseCase)
	webhookDispatcher := usecase.NewWebhookDispatcher(logger, webhookRepository)
	auditWebhookWorker := usecase.NewAuditWebhookOutboxWorker(logger, auditLogRepository, webhookDispatcher, 5*time.Second, 20)
	go auditWebhookWorker.Run(ctx)

	auditLogUseCase := usecase.NewAuditLogUseCase(auditLogRepository)
	auditLogsHandler := httpHandler.NewAuditLogsHandler(logger, auditLogUseCase)

	router := httpHandler.NewRouter(
		logger,
		healthHandler,
		authHandler,
		authUseCase,
		usersHandler,
		coursesHandler,
		quizzesHandler,
		attemptsHandler,
		enrollmentsHandler,
		certificatesHandler,
		courseTestsHandler,
		courseModulesHandler,
		contentBlocksHandler,
		reviewsHandler,
		notificationsHandler,
		webhooksHandler,
		auditLogsHandler,
		uploadsHandler,
		cfg.Upload.Dir,
	)
	router = httpMiddleware.CORS(cfg.HTTP.CORSAllowedOrigins)(router)

	server := &http.Server{
		Addr:              cfg.HTTP.Address,
		Handler:           router,
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
	}

	serverErrCh := make(chan error, 1)

	go func() {
		logger.Info("http server starting",
			slog.String("service", cfg.App.Name),
			slog.String("env", cfg.App.Env),
			slog.String("address", cfg.HTTP.Address),
		)

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrCh <- fmt.Errorf("main listen and serve: %w", err)
			return
		}

		close(serverErrCh)
	}()

	select {
	case err := <-serverErrCh:
		if err != nil {
			return err
		}
		return nil
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer cancel()

	logger.Info("http server shutting down", slog.String("service", cfg.App.Name))

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("main shutdown server: %w", err)
	}

	logger.Info("http server stopped", slog.String("service", cfg.App.Name))

	return nil
}

func loadEnvFiles() error {
	paths := make([]string, 0, 4)

	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		paths = append(paths, filepath.Join(filepath.Dir(exeDir), ".env"))
		paths = append(paths, filepath.Join(exeDir, ".env"))
	}

	paths = append(paths, ".env")

	if override := strings.TrimSpace(os.Getenv("ENV_FILE")); override != "" {
		paths = append(paths, override)
	}

	seen := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		normalized := filepath.Clean(path)
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}

		if _, err := os.Stat(normalized); err != nil {
			if os.IsNotExist(err) {
				continue
			}

			return fmt.Errorf("stat env file %s: %w", normalized, err)
		}

		if err := config.LoadEnvFile(normalized); err != nil {
			return err
		}
	}

	return nil
}

func newLogger(levelText string) (*slog.Logger, error) {
	var level slog.Level
	if err := level.UnmarshalText([]byte(levelText)); err != nil {
		return nil, fmt.Errorf("main parse log level: %w", err)
	}

	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})), nil
}

func parseFlags() runtimeFlags {
	migrate := flag.Bool("migrate", false, "run SQL migrations and exit")
	seedAdmin := flag.Bool("seed-admin", false, "seed bootstrap admin user and exit")
	all := flag.Bool("all", false, "run migrations and admin seed, then exit")
	flag.Parse()

	return runtimeFlags{
		migrate:   *migrate,
		seedAdmin: *seedAdmin,
		all:       *all,
	}
}

func resolveOperationFlags(flags runtimeFlags, cfg config.Config) (bool, bool) {
	explicitOperations := flags.migrate || flags.seedAdmin || flags.all
	if explicitOperations {
		if flags.all {
			return true, true
		}

		return flags.migrate, flags.seedAdmin
	}

	if flags.all {
		return true, true
	}

	return cfg.Migrate.RunOnStart, cfg.Seed.RunOnStart
}
