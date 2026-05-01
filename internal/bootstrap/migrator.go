package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Migrator struct {
	pool *pgxpool.Pool
}

func NewMigrator(pool *pgxpool.Pool) *Migrator {
	return &Migrator{pool: pool}
}

func (m *Migrator) Up(ctx context.Context, dir string) error {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return fmt.Errorf("bootstrap migrate up: empty migrations dir")
	}

	if err := m.ensureVersionTable(ctx); err != nil {
		return fmt.Errorf("bootstrap migrate up ensure version table: %w", err)
	}

	appliedVersions, err := m.appliedVersions(ctx)
	if err != nil {
		return fmt.Errorf("bootstrap migrate up applied versions: %w", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("bootstrap migrate up read dir: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}

		versionID, err := migrationVersionFromFilename(entry.Name())
		if err != nil {
			return fmt.Errorf("bootstrap migrate up parse version %s: %w", entry.Name(), err)
		}

		if _, exists := appliedVersions[versionID]; exists {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("bootstrap migrate up read file %s: %w", path, err)
		}

		upSQL, err := extractGooseUpSQL(string(content))
		if err != nil {
			return fmt.Errorf("bootstrap migrate up extract up sql %s: %w", path, err)
		}

		if err := m.applyMigration(ctx, versionID, upSQL); err != nil {
			return fmt.Errorf("bootstrap migrate up apply %s: %w", path, err)
		}
	}

	return nil
}

func (m *Migrator) ensureVersionTable(ctx context.Context) error {
	const query = `
		CREATE TABLE IF NOT EXISTS goose_db_version (
			id BIGSERIAL PRIMARY KEY,
			version_id BIGINT NOT NULL,
			is_applied BOOLEAN NOT NULL,
			tstamp TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`

	if _, err := m.pool.Exec(ctx, query); err != nil {
		return fmt.Errorf("bootstrap migrate ensure version table exec: %w", err)
	}

	return nil
}

func (m *Migrator) appliedVersions(ctx context.Context) (map[int64]struct{}, error) {
	rows, err := m.pool.Query(ctx, `
		SELECT version_id
		FROM goose_db_version
		WHERE is_applied = true
	`)
	if err != nil {
		return nil, fmt.Errorf("bootstrap migrate applied versions query: %w", err)
	}
	defer rows.Close()

	versions := make(map[int64]struct{})
	for rows.Next() {
		var versionID int64
		if err := rows.Scan(&versionID); err != nil {
			return nil, fmt.Errorf("bootstrap migrate applied versions scan: %w", err)
		}

		versions[versionID] = struct{}{}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("bootstrap migrate applied versions rows: %w", err)
	}

	return versions, nil
}

func (m *Migrator) applyMigration(ctx context.Context, versionID int64, upSQL string) error {
	conn, err := m.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("bootstrap migrate apply acquire conn: %w", err)
	}
	defer conn.Release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("bootstrap migrate apply begin tx: %w", err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Conn().PgConn().Exec(ctx, upSQL).ReadAll(); err != nil {
		return fmt.Errorf("bootstrap migrate apply exec up sql: %w", normalizePgConnError(err))
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO goose_db_version (version_id, is_applied)
		VALUES ($1, true)
	`, versionID); err != nil {
		return fmt.Errorf("bootstrap migrate apply insert version: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("bootstrap migrate apply commit: %w", err)
	}

	return nil
}

func migrationVersionFromFilename(name string) (int64, error) {
	prefix, _, found := strings.Cut(name, "_")
	if !found {
		return 0, fmt.Errorf("migration filename has no version prefix")
	}

	versionID, err := strconv.ParseInt(prefix, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse version prefix: %w", err)
	}

	return versionID, nil
}

func extractGooseUpSQL(content string) (string, error) {
	lines := strings.Split(content, "\n")
	var builder strings.Builder
	inUp := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		switch trimmed {
		case "-- +goose Up":
			inUp = true
			continue
		case "-- +goose Down":
			inUp = false
			goto done
		}

		if !inUp {
			continue
		}

		if strings.HasPrefix(trimmed, "-- +goose ") {
			continue
		}

		builder.WriteString(line)
		builder.WriteByte('\n')
	}

done:
	upSQL := strings.TrimSpace(builder.String())
	if upSQL == "" {
		return "", fmt.Errorf("empty up sql")
	}

	return upSQL, nil
}

func normalizePgConnError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr
	}

	return err
}
