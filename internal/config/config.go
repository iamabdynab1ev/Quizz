package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	App      AppConfig
	HTTP     HTTPConfig
	Auth     AuthConfig
	Google   GoogleConfig
	Upload   UploadConfig
	Database DatabaseConfig
	Migrate  MigrateConfig
	Seed     SeedConfig
}

type AppConfig struct {
	Name     string
	Env      string
	LogLevel string
}

type HTTPConfig struct {
	Address            string
	CORSAllowedOrigins []string
	ReadTimeout        time.Duration
	ReadHeaderTimeout  time.Duration
	WriteTimeout       time.Duration
	IdleTimeout        time.Duration
	ShutdownTimeout    time.Duration
}

type AuthConfig struct {
	SessionTTL         time.Duration
	SessionCacheTTL    time.Duration
	BcryptCost         int
	LoginMaxAttempts   int
	LoginAttemptWindow time.Duration
}

type GoogleConfig struct {
	ClientID string
}

type UploadConfig struct {
	Dir          string
	MaxSizeBytes int64
}

type DatabaseConfig struct {
	URL               string
	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
}

type MigrateConfig struct {
	RunOnStart bool
	Dir        string
}

type SeedConfig struct {
	RunOnStart bool
	Admin      SeedAdminConfig
}

type SeedAdminConfig struct {
	Username     string
	Email        string
	Password     string
	FirstName    string
	LastName     string
	Patronymic   string
	IsSuperAdmin bool
	Permissions  []string
}

func Load() (Config, error) {
	readTimeout, err := getDuration("HTTP_READ_TIMEOUT", 5*time.Second)
	if err != nil {
		return Config{}, fmt.Errorf("config load http read timeout: %w", err)
	}

	readHeaderTimeout, err := getDuration("HTTP_READ_HEADER_TIMEOUT", 2*time.Second)
	if err != nil {
		return Config{}, fmt.Errorf("config load http read header timeout: %w", err)
	}

	writeTimeout, err := getDuration("HTTP_WRITE_TIMEOUT", 15*time.Second)
	if err != nil {
		return Config{}, fmt.Errorf("config load http write timeout: %w", err)
	}

	idleTimeout, err := getDuration("HTTP_IDLE_TIMEOUT", 60*time.Second)
	if err != nil {
		return Config{}, fmt.Errorf("config load http idle timeout: %w", err)
	}

	shutdownTimeout, err := getDuration("HTTP_SHUTDOWN_TIMEOUT", 15*time.Second)
	if err != nil {
		return Config{}, fmt.Errorf("config load http shutdown timeout: %w", err)
	}

	sessionTTL, err := getDuration("AUTH_SESSION_TTL", 24*time.Hour)
	if err != nil {
		return Config{}, fmt.Errorf("config load auth session ttl: %w", err)
	}

	sessionCacheTTL, err := getDuration("AUTH_SESSION_CACHE_TTL", 5*time.Minute)
	if err != nil {
		return Config{}, fmt.Errorf("config load auth session cache ttl: %w", err)
	}

	bcryptCost, err := getInt("AUTH_BCRYPT_COST", 12)
	if err != nil {
		return Config{}, fmt.Errorf("config load auth bcrypt cost: %w", err)
	}

	loginMaxAttempts, err := getInt("AUTH_LOGIN_MAX_ATTEMPTS", 5)
	if err != nil {
		return Config{}, fmt.Errorf("config load auth login max attempts: %w", err)
	}

	loginAttemptWindow, err := getDuration("AUTH_LOGIN_ATTEMPT_WINDOW", 15*time.Minute)
	if err != nil {
		return Config{}, fmt.Errorf("config load auth login attempt window: %w", err)
	}

	uploadMaxSizeMB, err := getInt("UPLOAD_MAX_SIZE_MB", 20)
	if err != nil {
		return Config{}, fmt.Errorf("config load upload max size mb: %w", err)
	}

	maxConns, err := getInt32("PGX_MAX_CONNS", 20)
	if err != nil {
		return Config{}, fmt.Errorf("config load pgx max conns: %w", err)
	}

	minConns, err := getInt32("PGX_MIN_CONNS", 2)
	if err != nil {
		return Config{}, fmt.Errorf("config load pgx min conns: %w", err)
	}

	maxConnLifetime, err := getDuration("PGX_MAX_CONN_LIFETIME", 30*time.Minute)
	if err != nil {
		return Config{}, fmt.Errorf("config load pgx max conn lifetime: %w", err)
	}

	maxConnIdleTime, err := getDuration("PGX_MAX_CONN_IDLE_TIME", 5*time.Minute)
	if err != nil {
		return Config{}, fmt.Errorf("config load pgx max conn idle time: %w", err)
	}

	healthCheckPeriod, err := getDuration("PGX_HEALTH_CHECK_PERIOD", time.Minute)
	if err != nil {
		return Config{}, fmt.Errorf("config load pgx health check period: %w", err)
	}

	seedRunOnStart, err := getBool("SEED_RUN_ON_START", false)
	if err != nil {
		return Config{}, fmt.Errorf("config load seed run on start: %w", err)
	}

	migrateRunOnStart, err := getBool("MIGRATE_RUN_ON_START", false)
	if err != nil {
		return Config{}, fmt.Errorf("config load migrate run on start: %w", err)
	}

	seedAdminSuperAdmin, err := getBool("SEED_ADMIN_IS_SUPER_ADMIN", true)
	if err != nil {
		return Config{}, fmt.Errorf("config load seed admin is super admin: %w", err)
	}

	databaseURL, err := getRequired("DATABASE_URL")
	if err != nil {
		return Config{}, fmt.Errorf("config load database url: %w", err)
	}

	return Config{
		App: AppConfig{
			Name:     getEnv("APP_NAME", "QUIZ"),
			Env:      getEnv("APP_ENV", "development"),
			LogLevel: strings.ToUpper(getEnv("LOG_LEVEL", "INFO")),
		},
		HTTP: HTTPConfig{
			Address:            getEnv("HTTP_ADDRESS", ":8080"),
			CORSAllowedOrigins: getStringSlice("HTTP_CORS_ALLOWED_ORIGINS", []string{"*"}),
			ReadTimeout:        readTimeout,
			ReadHeaderTimeout:  readHeaderTimeout,
			WriteTimeout:       writeTimeout,
			IdleTimeout:        idleTimeout,
			ShutdownTimeout:    shutdownTimeout,
		},
		Auth: AuthConfig{
			SessionTTL:         sessionTTL,
			SessionCacheTTL:    sessionCacheTTL,
			BcryptCost:         bcryptCost,
			LoginMaxAttempts:   loginMaxAttempts,
			LoginAttemptWindow: loginAttemptWindow,
		},
		Google: GoogleConfig{
			ClientID: getEnv("GOOGLE_CLIENT_ID", ""),
		},
		Upload: UploadConfig{
			Dir:          getEnv("UPLOADS_DIR", "uploads"),
			MaxSizeBytes: int64(uploadMaxSizeMB) * 1024 * 1024,
		},
		Database: DatabaseConfig{
			URL:               databaseURL,
			MaxConns:          maxConns,
			MinConns:          minConns,
			MaxConnLifetime:   maxConnLifetime,
			MaxConnIdleTime:   maxConnIdleTime,
			HealthCheckPeriod: healthCheckPeriod,
		},
		Migrate: MigrateConfig{
			RunOnStart: migrateRunOnStart,
			Dir:        getEnv("MIGRATIONS_DIR", "migrations"),
		},
		Seed: SeedConfig{
			RunOnStart: seedRunOnStart,
			Admin: SeedAdminConfig{
				Username:     getEnv("SEED_ADMIN_USERNAME", "admin"),
				Email:        getEnv("SEED_ADMIN_EMAIL", "admin@local.test"),
				Password:     getEnv("SEED_ADMIN_PASSWORD", "Admin123!"),
				FirstName:    getEnv("SEED_ADMIN_FIRST_NAME", "System"),
				LastName:     getEnv("SEED_ADMIN_LAST_NAME", "Admin"),
				Patronymic:   getEnv("SEED_ADMIN_PATRONYMIC", ""),
				IsSuperAdmin: seedAdminSuperAdmin,
				Permissions:  getStringSlice("SEED_ADMIN_PERMISSIONS", []string{"*"}),
			},
		},
	}, nil
}

func getRequired(key string) (string, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return "", fmt.Errorf("config required env %s is empty", key)
	}

	return value, nil
}

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	return value
}

func getDuration(key string, fallback time.Duration) (time.Duration, error) {
	rawValue := strings.TrimSpace(os.Getenv(key))
	if rawValue == "" {
		return fallback, nil
	}

	value, err := time.ParseDuration(rawValue)
	if err != nil {
		return 0, fmt.Errorf("config parse duration %s: %w", key, err)
	}

	return value, nil
}

func getInt32(key string, fallback int32) (int32, error) {
	rawValue := strings.TrimSpace(os.Getenv(key))
	if rawValue == "" {
		return fallback, nil
	}

	value, err := strconv.ParseInt(rawValue, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("config parse int32 %s: %w", key, err)
	}

	return int32(value), nil
}

func getInt(key string, fallback int) (int, error) {
	rawValue := strings.TrimSpace(os.Getenv(key))
	if rawValue == "" {
		return fallback, nil
	}

	value, err := strconv.Atoi(rawValue)
	if err != nil {
		return 0, fmt.Errorf("config parse int %s: %w", key, err)
	}

	return value, nil
}

func getBool(key string, fallback bool) (bool, error) {
	rawValue := strings.TrimSpace(os.Getenv(key))
	if rawValue == "" {
		return fallback, nil
	}

	value, err := strconv.ParseBool(rawValue)
	if err != nil {
		return false, fmt.Errorf("config parse bool %s: %w", key, err)
	}

	return value, nil
}

func getStringSlice(key string, fallback []string) []string {
	rawValue := strings.TrimSpace(os.Getenv(key))
	if rawValue == "" {
		return append([]string(nil), fallback...)
	}

	parts := strings.Split(rawValue, ",")
	values := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}

		if _, exists := seen[trimmed]; exists {
			continue
		}

		seen[trimmed] = struct{}{}
		values = append(values, trimmed)
	}

	if len(values) == 0 {
		return append([]string(nil), fallback...)
	}

	return values
}
