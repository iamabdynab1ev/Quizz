package usecase

import (
	"context"
	"fmt"
	"strings"

	"lms-arvand-backend/internal/domain"
)

type dashboardRepository interface {
	GetStudentDashboard(ctx context.Context, userID string) (domain.StudentDashboard, error)
	GetAdminDashboard(ctx context.Context) (domain.AdminDashboard, error)
}

type DashboardUseCase struct {
	repository dashboardRepository
}

func NewDashboardUseCase(repository dashboardRepository) *DashboardUseCase {
	return &DashboardUseCase{repository: repository}
}

func (u *DashboardUseCase) GetStudentDashboard(ctx context.Context, userID string) (domain.StudentDashboard, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return domain.StudentDashboard{}, domain.FieldValidationError("Проверьте параметры запроса",
			domain.ValidationField("user_id", "required", "ID пользователя обязателен"))
	}

	dashboard, err := u.repository.GetStudentDashboard(ctx, userID)
	if err != nil {
		return domain.StudentDashboard{}, fmt.Errorf("usecase dashboard get student: %w", err)
	}

	return dashboard, nil
}

func (u *DashboardUseCase) GetAdminDashboard(ctx context.Context) (domain.AdminDashboard, error) {
	dashboard, err := u.repository.GetAdminDashboard(ctx)
	if err != nil {
		return domain.AdminDashboard{}, fmt.Errorf("usecase dashboard get admin: %w", err)
	}

	return dashboard, nil
}
