package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"lms-arvand-backend/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type contentBlockRowScanner interface {
	Scan(dest ...any) error
}

type ContentBlockRepository struct {
	pool *pgxpool.Pool
}

func NewContentBlockRepository(pool *pgxpool.Pool) *ContentBlockRepository {
	return &ContentBlockRepository{pool: pool}
}

func (r *ContentBlockRepository) Create(ctx context.Context, params domain.CreateContentBlockParams) (domain.ContentBlock, error) {
	titleValue, err := multiLangValueOrNil(params.Title)
	if err != nil {
		return domain.ContentBlock{}, fmt.Errorf("repository postgres content blocks create title value: %w", err)
	}

	block, err := scanContentBlockRow(r.pool.QueryRow(ctx, `
		INSERT INTO content_blocks (
			course_id,
			module_id,
			position,
			type,
			title,
			payload
		) VALUES (
			$1::uuid, $2::uuid, $3, $4, $5::jsonb, $6::jsonb
		)
		RETURNING id, course_id::text, module_id::text, position, type, title, payload
	`,
		nullableStringPointerForWrite(params.CourseID),
		nullableStringPointerForWrite(params.ModuleID),
		params.Position,
		string(params.Type),
		titleValue,
		[]byte(params.Payload),
	))
	if err != nil {
		return domain.ContentBlock{}, wrapPGError("repository postgres content blocks create", err)
	}

	return block, nil
}

func (r *ContentBlockRepository) GetByID(ctx context.Context, blockID string) (domain.ContentBlock, error) {
	block, err := scanContentBlockRow(r.pool.QueryRow(ctx, `
		SELECT id, course_id::text, module_id::text, position, type, title, payload
		FROM content_blocks
		WHERE id = $1
	`, blockID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ContentBlock{}, fmt.Errorf("repository postgres content blocks get by id: %w", domain.ErrNotFound)
		}

		return domain.ContentBlock{}, fmt.Errorf("repository postgres content blocks get by id: %w", err)
	}

	return block, nil
}

func (r *ContentBlockRepository) List(ctx context.Context, filter domain.ContentBlockListFilter) ([]domain.ContentBlock, int, error) {
	query := strings.Builder{}
	query.WriteString(`
		SELECT id, course_id::text, module_id::text, position, type, title, payload,
			COUNT(*) OVER() AS total_count
		FROM content_blocks
		WHERE 1 = 1
	`)

	args := make([]any, 0, 3)
	position := 1

	if filter.CourseID != nil {
		query.WriteString(fmt.Sprintf(` AND course_id = $%d::uuid`, position))
		args = append(args, *filter.CourseID)
		position++
	}

	if filter.ModuleID != nil {
		query.WriteString(fmt.Sprintf(` AND module_id = $%d::uuid`, position))
		args = append(args, *filter.ModuleID)
		position++
	}

	query.WriteString(` ORDER BY position ASC`)

	rows, err := r.pool.Query(ctx, query.String(), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("repository postgres content blocks list query: %w", err)
	}
	defer rows.Close()

	var total int
	blocks := make([]domain.ContentBlock, 0)
	for rows.Next() {
		block, rowTotal, err := scanContentBlockRowWithTotal(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("repository postgres content blocks list scan: %w", err)
		}
		total = rowTotal
		blocks = append(blocks, block)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("repository postgres content blocks list rows: %w", err)
	}

	return blocks, total, nil
}

func (r *ContentBlockRepository) Update(ctx context.Context, params domain.UpdateContentBlockParams) (domain.ContentBlock, error) {
	titleValue, err := multiLangValueOrNil(params.Title)
	if err != nil {
		return domain.ContentBlock{}, fmt.Errorf("repository postgres content blocks update title value: %w", err)
	}

	block, err := scanContentBlockRow(r.pool.QueryRow(ctx, `
		UPDATE content_blocks
		SET
			course_id = $2::uuid,
			module_id = $3::uuid,
			position = $4,
			type = $5,
			title = $6::jsonb,
			payload = $7::jsonb
		WHERE id = $1
		RETURNING id, course_id::text, module_id::text, position, type, title, payload
	`,
		params.ID,
		nullableStringPointerForWrite(params.CourseID),
		nullableStringPointerForWrite(params.ModuleID),
		params.Position,
		string(params.Type),
		titleValue,
		[]byte(params.Payload),
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ContentBlock{}, fmt.Errorf("repository postgres content blocks update: %w", domain.ErrNotFound)
		}

		return domain.ContentBlock{}, wrapPGError("repository postgres content blocks update", err)
	}

	return block, nil
}

func (r *ContentBlockRepository) Delete(ctx context.Context, blockID string) error {
	commandTag, err := r.pool.Exec(ctx, `DELETE FROM content_blocks WHERE id = $1`, blockID)
	if err != nil {
		return fmt.Errorf("repository postgres content blocks delete: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("repository postgres content blocks delete: %w", domain.ErrNotFound)
	}

	return nil
}

func scanContentBlockRowWithTotal(scanner contentBlockRowScanner) (domain.ContentBlock, int, error) {
	var block domain.ContentBlock
	var courseID sql.NullString
	var moduleID sql.NullString
	var blockType string
	var titleBytes []byte
	var total int

	if err := scanner.Scan(
		&block.ID,
		&courseID,
		&moduleID,
		&block.Position,
		&blockType,
		&titleBytes,
		&block.Payload,
		&total,
	); err != nil {
		return domain.ContentBlock{}, 0, err
	}

	if len(titleBytes) > 0 {
		if err := block.Title.Scan(titleBytes); err != nil {
			return domain.ContentBlock{}, 0, fmt.Errorf("repository postgres scan content block title: %w", err)
		}
	}

	block.CourseID = optionalString(courseID)
	block.ModuleID = optionalString(moduleID)
	block.Type = domain.ContentBlockType(blockType)

	return block, total, nil
}

func scanContentBlockRow(scanner contentBlockRowScanner) (domain.ContentBlock, error) {
	var block domain.ContentBlock
	var courseID sql.NullString
	var moduleID sql.NullString
	var blockType string
	var titleBytes []byte

	if err := scanner.Scan(
		&block.ID,
		&courseID,
		&moduleID,
		&block.Position,
		&blockType,
		&titleBytes,
		&block.Payload,
	); err != nil {
		return domain.ContentBlock{}, err
	}

	if len(titleBytes) > 0 {
		if err := block.Title.Scan(titleBytes); err != nil {
			return domain.ContentBlock{}, fmt.Errorf("repository postgres scan content block title: %w", err)
		}
	}

	block.CourseID = optionalString(courseID)
	block.ModuleID = optionalString(moduleID)
	block.Type = domain.ContentBlockType(blockType)

	return block, nil
}
