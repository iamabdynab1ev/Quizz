package postgres

import (
	"context"
	"fmt"

	"lms-arvand-backend/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type CoursePackageRepository struct {
	pool *pgxpool.Pool
}

func NewCoursePackageRepository(pool *pgxpool.Pool) *CoursePackageRepository {
	return &CoursePackageRepository{pool: pool}
}

// Create creates a course (which now embeds quiz settings and questions).
func (r *CoursePackageRepository) Create(ctx context.Context, params domain.CreateCoursePackageParams) (domain.Course, error) {
	courseRepo := NewCourseRepository(r.pool)
	course, err := courseRepo.Create(ctx, params.Course)
	if err != nil {
		return domain.Course{}, fmt.Errorf("repository postgres course packages create: %w", err)
	}
	return course, nil
}
