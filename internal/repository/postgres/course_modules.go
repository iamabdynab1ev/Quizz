package postgres

import (
	"context"
	"errors"
	"fmt"

	"lms-arvand-backend/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type courseModuleRowScanner interface {
	Scan(dest ...any) error
}

type CourseModuleRepository struct {
	pool *pgxpool.Pool
}

func NewCourseModuleRepository(pool *pgxpool.Pool) *CourseModuleRepository {
	return &CourseModuleRepository{pool: pool}
}

func (r *CourseModuleRepository) Create(ctx context.Context, params domain.CreateCourseModuleParams) (domain.CourseModule, error) {
	titleValue, err := toJSONValue(params.Title)
	if err != nil {
		return domain.CourseModule{}, fmt.Errorf("repository postgres course modules create title value: %w", err)
	}

	descriptionValue, err := multiLangValueOrNil(params.Description)
	if err != nil {
		return domain.CourseModule{}, fmt.Errorf("repository postgres course modules create description value: %w", err)
	}

	module, err := scanCourseModuleRow(r.pool.QueryRow(ctx, `
		INSERT INTO course_modules (
			course_id,
			position,
			title,
			description
		) VALUES (
			$1, $2, $3::jsonb, $4::jsonb
		)
		RETURNING id, course_id, position, title, description
	`,
		params.CourseID,
		params.Position,
		titleValue,
		descriptionValue,
	))
	if err != nil {
		return domain.CourseModule{}, wrapPGError("repository postgres course modules create", err)
	}

	return module, nil
}

func (r *CourseModuleRepository) GetByID(ctx context.Context, moduleID string) (domain.CourseModule, error) {
	module, err := scanCourseModuleRow(r.pool.QueryRow(ctx, `
		SELECT id, course_id, position, title, description
		FROM course_modules
		WHERE id = $1
	`, moduleID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.CourseModule{}, fmt.Errorf("repository postgres course modules get by id: %w", domain.ErrNotFound)
		}

		return domain.CourseModule{}, fmt.Errorf("repository postgres course modules get by id: %w", err)
	}

	return module, nil
}

func (r *CourseModuleRepository) List(ctx context.Context, filter domain.CourseModuleListFilter) ([]domain.CourseModule, int, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, course_id, position, title, description,
			COUNT(*) OVER() AS total_count
		FROM course_modules
		WHERE course_id = $1
		ORDER BY position ASC
	`, filter.CourseID)
	if err != nil {
		return nil, 0, fmt.Errorf("repository postgres course modules list query: %w", err)
	}
	defer rows.Close()

	var total int
	modules := make([]domain.CourseModule, 0)
	for rows.Next() {
		module, rowTotal, err := scanCourseModuleRowWithTotal(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("repository postgres course modules list scan: %w", err)
		}
		total = rowTotal
		modules = append(modules, module)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("repository postgres course modules list rows: %w", err)
	}

	return modules, total, nil
}

func (r *CourseModuleRepository) Update(ctx context.Context, params domain.UpdateCourseModuleParams) (domain.CourseModule, error) {
	titleValue, err := toJSONValue(params.Title)
	if err != nil {
		return domain.CourseModule{}, fmt.Errorf("repository postgres course modules update title value: %w", err)
	}

	descriptionValue, err := multiLangValueOrNil(params.Description)
	if err != nil {
		return domain.CourseModule{}, fmt.Errorf("repository postgres course modules update description value: %w", err)
	}

	module, err := scanCourseModuleRow(r.pool.QueryRow(ctx, `
		UPDATE course_modules
		SET
			course_id = $2,
			position = $3,
			title = $4::jsonb,
			description = $5::jsonb
		WHERE id = $1
		RETURNING id, course_id, position, title, description
	`,
		params.ID,
		params.CourseID,
		params.Position,
		titleValue,
		descriptionValue,
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.CourseModule{}, fmt.Errorf("repository postgres course modules update: %w", domain.ErrNotFound)
		}

		return domain.CourseModule{}, wrapPGError("repository postgres course modules update", err)
	}

	return module, nil
}

func (r *CourseModuleRepository) Delete(ctx context.Context, moduleID string) error {
	commandTag, err := r.pool.Exec(ctx, `DELETE FROM course_modules WHERE id = $1`, moduleID)
	if err != nil {
		return fmt.Errorf("repository postgres course modules delete: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("repository postgres course modules delete: %w", domain.ErrNotFound)
	}

	return nil
}

func scanCourseModuleRowWithTotal(scanner courseModuleRowScanner) (domain.CourseModule, int, error) {
	var module domain.CourseModule
	var titleBytes []byte
	var descriptionBytes []byte
	var total int

	if err := scanner.Scan(
		&module.ID,
		&module.CourseID,
		&module.Position,
		&titleBytes,
		&descriptionBytes,
		&total,
	); err != nil {
		return domain.CourseModule{}, 0, err
	}

	if err := module.Title.Scan(titleBytes); err != nil {
		return domain.CourseModule{}, 0, fmt.Errorf("repository postgres scan course module title: %w", err)
	}

	if len(descriptionBytes) > 0 {
		if err := module.Description.Scan(descriptionBytes); err != nil {
			return domain.CourseModule{}, 0, fmt.Errorf("repository postgres scan course module description: %w", err)
		}
	}

	return module, total, nil
}

func scanCourseModuleRow(scanner courseModuleRowScanner) (domain.CourseModule, error) {
	var module domain.CourseModule
	var titleBytes []byte
	var descriptionBytes []byte

	if err := scanner.Scan(
		&module.ID,
		&module.CourseID,
		&module.Position,
		&titleBytes,
		&descriptionBytes,
	); err != nil {
		return domain.CourseModule{}, err
	}

	if err := module.Title.Scan(titleBytes); err != nil {
		return domain.CourseModule{}, fmt.Errorf("repository postgres scan course module title: %w", err)
	}

	if len(descriptionBytes) > 0 {
		if err := module.Description.Scan(descriptionBytes); err != nil {
			return domain.CourseModule{}, fmt.Errorf("repository postgres scan course module description: %w", err)
		}
	}

	return module, nil
}
