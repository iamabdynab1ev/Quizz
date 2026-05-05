package postgres

import (
	"context"
	"fmt"

	"lms-arvand-backend/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DashboardRepository struct {
	pool *pgxpool.Pool
}

func NewDashboardRepository(pool *pgxpool.Pool) *DashboardRepository {
	return &DashboardRepository{pool: pool}
}

func (r *DashboardRepository) GetStudentDashboard(ctx context.Context, userID string) (domain.StudentDashboard, error) {
	user, err := NewUserRepository(r.pool).GetByID(ctx, userID)
	if err != nil {
		return domain.StudentDashboard{}, fmt.Errorf("repository postgres dashboard get student user: %w", err)
	}

	stats, err := r.getStudentStats(ctx, userID)
	if err != nil {
		return domain.StudentDashboard{}, err
	}

	userIDFilter := userID
	recentEnrollments, _, err := NewEnrollmentRepository(r.pool).List(ctx, domain.EnrollmentListFilter{
		UserID: &userIDFilter,
		Limit:  5,
		Offset: 0,
	})
	if err != nil {
		return domain.StudentDashboard{}, fmt.Errorf("repository postgres dashboard list student enrollments: %w", err)
	}

	recentAttempts, _, err := NewAttemptRepository(r.pool).ListAttempts(ctx, domain.AttemptListFilter{
		UserID: &userIDFilter,
		Limit:  5,
		Offset: 0,
	})
	if err != nil {
		return domain.StudentDashboard{}, fmt.Errorf("repository postgres dashboard list student attempts: %w", err)
	}

	certificates, _, err := NewCertificateRepository(r.pool).List(ctx, domain.CertificateListFilter{
		UserID: &userIDFilter,
		Limit:  5,
		Offset: 0,
	})
	if err != nil {
		return domain.StudentDashboard{}, fmt.Errorf("repository postgres dashboard list student certificates: %w", err)
	}

	return domain.StudentDashboard{
		User:              user,
		Stats:             stats,
		RecentEnrollments: recentEnrollments,
		RecentAttempts:    recentAttempts,
		Certificates:      certificates,
	}, nil
}

func (r *DashboardRepository) GetAdminDashboard(ctx context.Context) (domain.AdminDashboard, error) {
	stats, err := r.getAdminStats(ctx)
	if err != nil {
		return domain.AdminDashboard{}, err
	}

	recentUsers, _, err := NewUserRepository(r.pool).List(ctx, domain.UserListFilter{
		Limit:  5,
		Offset: 0,
	})
	if err != nil {
		return domain.AdminDashboard{}, fmt.Errorf("repository postgres dashboard list recent users: %w", err)
	}

	recentAttempts, _, err := NewAttemptRepository(r.pool).ListAttempts(ctx, domain.AttemptListFilter{
		Limit:  5,
		Offset: 0,
	})
	if err != nil {
		return domain.AdminDashboard{}, fmt.Errorf("repository postgres dashboard list recent attempts: %w", err)
	}

	return domain.AdminDashboard{
		Stats:          stats,
		RecentUsers:    recentUsers,
		RecentAttempts: recentAttempts,
	}, nil
}

func (r *DashboardRepository) getStudentStats(ctx context.Context, userID string) (domain.StudentDashboardStats, error) {
	var stats domain.StudentDashboardStats

	if err := r.pool.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE status = 'active')::int,
			COUNT(*) FILTER (WHERE status = 'completed')::int
		FROM enrollments
		WHERE user_id = $1
	`, userID).Scan(&stats.ActiveEnrollments, &stats.CompletedEnrollments); err != nil {
		return domain.StudentDashboardStats{}, fmt.Errorf("repository postgres dashboard student enrollment stats: %w", err)
	}

	if err := r.pool.QueryRow(ctx, `
		SELECT
			COUNT(*)::int,
			COUNT(*) FILTER (WHERE passed = true)::int,
			COALESCE(ROUND(AVG(score_percent)::numeric, 2), 0)::float8
		FROM attempts
		WHERE user_id = $1
	`, userID).Scan(&stats.AttemptsTotal, &stats.PassedAttempts, &stats.AverageScorePercent); err != nil {
		return domain.StudentDashboardStats{}, fmt.Errorf("repository postgres dashboard student attempt stats: %w", err)
	}

	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*)::int FROM certificates WHERE user_id = $1`, userID).Scan(&stats.CertificatesTotal); err != nil {
		return domain.StudentDashboardStats{}, fmt.Errorf("repository postgres dashboard student certificate stats: %w", err)
	}

	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*)::int FROM notifications WHERE user_id = $1 AND read = false`, userID).Scan(&stats.UnreadNotifications); err != nil {
		return domain.StudentDashboardStats{}, fmt.Errorf("repository postgres dashboard student notification stats: %w", err)
	}

	return stats, nil
}

func (r *DashboardRepository) getAdminStats(ctx context.Context) (domain.AdminDashboardStats, error) {
	var stats domain.AdminDashboardStats

	if err := r.pool.QueryRow(ctx, `
		SELECT
			COUNT(*)::int,
			COUNT(*) FILTER (WHERE is_active = true)::int,
			COUNT(*) FILTER (WHERE role = 'student')::int
		FROM users
	`).Scan(&stats.UsersTotal, &stats.ActiveUsers, &stats.StudentsTotal); err != nil {
		return domain.AdminDashboardStats{}, fmt.Errorf("repository postgres dashboard admin user stats: %w", err)
	}

	if err := r.pool.QueryRow(ctx, `
		SELECT
			COUNT(*)::int,
			COUNT(*) FILTER (WHERE status = 'published')::int
		FROM courses
	`).Scan(&stats.CoursesTotal, &stats.PublishedCourses); err != nil {
		return domain.AdminDashboardStats{}, fmt.Errorf("repository postgres dashboard admin course stats: %w", err)
	}

	if err := r.pool.QueryRow(ctx, `
		SELECT
			COUNT(*)::int,
			COUNT(*) FILTER (WHERE status = 'published')::int
		FROM quizzes
	`).Scan(&stats.QuizzesTotal, &stats.PublishedQuizzes); err != nil {
		return domain.AdminDashboardStats{}, fmt.Errorf("repository postgres dashboard admin quiz stats: %w", err)
	}

	if err := r.pool.QueryRow(ctx, `
		SELECT
			COUNT(*)::int,
			COUNT(*) FILTER (WHERE status = 'completed')::int
		FROM enrollments
	`).Scan(&stats.EnrollmentsTotal, &stats.CompletedEnrollments); err != nil {
		return domain.AdminDashboardStats{}, fmt.Errorf("repository postgres dashboard admin enrollment stats: %w", err)
	}

	if err := r.pool.QueryRow(ctx, `
		SELECT
			COUNT(*)::int,
			COUNT(*) FILTER (WHERE passed = true)::int,
			COUNT(*) FILTER (WHERE needs_review = true)::int
		FROM attempts
	`).Scan(&stats.AttemptsTotal, &stats.PassedAttempts, &stats.AttemptsNeedReview); err != nil {
		return domain.AdminDashboardStats{}, fmt.Errorf("repository postgres dashboard admin attempt stats: %w", err)
	}

	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*)::int FROM certificates`).Scan(&stats.CertificatesTotal); err != nil {
		return domain.AdminDashboardStats{}, fmt.Errorf("repository postgres dashboard admin certificate stats: %w", err)
	}

	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*)::int FROM reviews WHERE status = 'pending'`).Scan(&stats.PendingReviews); err != nil {
		return domain.AdminDashboardStats{}, fmt.Errorf("repository postgres dashboard admin review stats: %w", err)
	}

	return stats, nil
}
