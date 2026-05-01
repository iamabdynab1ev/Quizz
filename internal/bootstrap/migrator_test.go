package bootstrap

import "testing"

func TestMigrationVersionFromFilename(t *testing.T) {
	t.Parallel()

	got, err := migrationVersionFromFilename("00001_init_schema.sql")
	if err != nil {
		t.Fatalf("migrationVersionFromFilename() error = %v", err)
	}

	if got != 1 {
		t.Fatalf("migrationVersionFromFilename() = %d, want %d", got, 1)
	}
}

func TestExtractGooseUpSQL(t *testing.T) {
	t.Parallel()

	content := `-- +goose Up
-- +goose StatementBegin
CREATE TABLE demo (
    id INT
);
-- +goose StatementEnd

-- +goose Down
DROP TABLE demo;
`

	got, err := extractGooseUpSQL(content)
	if err != nil {
		t.Fatalf("extractGooseUpSQL() error = %v", err)
	}

	want := "CREATE TABLE demo (\n    id INT\n);"
	if got != want {
		t.Fatalf("extractGooseUpSQL() = %q, want %q", got, want)
	}
}
