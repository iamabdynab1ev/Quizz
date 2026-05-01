package config

import "testing"

func TestDatabaseNameFromURL(t *testing.T) {
	t.Parallel()

	got, err := DatabaseNameFromURL("postgres://postgres:postgres@127.0.0.1:5432/lms_arvand?sslmode=disable")
	if err != nil {
		t.Fatalf("DatabaseNameFromURL() error = %v", err)
	}

	if got != "lms_arvand" {
		t.Fatalf("DatabaseNameFromURL() = %q, want %q", got, "lms_arvand")
	}
}

func TestDatabaseAdminURL(t *testing.T) {
	t.Parallel()

	got, err := DatabaseAdminURL("postgres://postgres:postgres@127.0.0.1:5432/lms_arvand?sslmode=disable", "postgres")
	if err != nil {
		t.Fatalf("DatabaseAdminURL() error = %v", err)
	}

	want := "postgres://postgres:postgres@127.0.0.1:5432/postgres?sslmode=disable"
	if got != want {
		t.Fatalf("DatabaseAdminURL() = %q, want %q", got, want)
	}
}

func TestDatabaseNameFromURLEmptyPath(t *testing.T) {
	t.Parallel()

	if _, err := DatabaseNameFromURL("postgres://postgres:postgres@127.0.0.1:5432"); err == nil {
		t.Fatal("DatabaseNameFromURL() error = nil, want non-nil")
	}
}
