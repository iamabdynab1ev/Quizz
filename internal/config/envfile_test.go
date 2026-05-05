package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEnvFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")

	content := "TEST_ENV_ONE=development\nTEST_ENV_TWO=:8080\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	if err := LoadEnvFile(path); err != nil {
		t.Fatalf("LoadEnvFile() error = %v", err)
	}

	if got := os.Getenv("TEST_ENV_ONE"); got != "development" {
		t.Fatalf("TEST_ENV_ONE = %q, want %q", got, "development")
	}

	if got := os.Getenv("TEST_ENV_TWO"); got != ":8080" {
		t.Fatalf("TEST_ENV_TWO = %q, want %q", got, ":8080")
	}
}

func TestLoadEnvFileOverridesExistingValues(t *testing.T) {
	t.Setenv("TEST_ENV_THREE", "production")

	dir := t.TempDir()
	path := filepath.Join(dir, ".env")

	content := "TEST_ENV_THREE=development\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	if err := LoadEnvFile(path); err != nil {
		t.Fatalf("LoadEnvFile() error = %v", err)
	}

	if got := os.Getenv("TEST_ENV_THREE"); got != "development" {
		t.Fatalf("TEST_ENV_THREE = %q, want %q", got, "development")
	}
}

func TestLoadEnvFileInvalidLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")

	content := "BROKEN_LINE\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	if err := LoadEnvFile(path); err == nil {
		t.Fatal("LoadEnvFile() error = nil, want non-nil")
	}
}
