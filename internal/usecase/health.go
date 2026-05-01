package usecase

import (
	"context"
	"fmt"
	"time"
)

type healthRepository interface {
	Ping(ctx context.Context) error
}

type HealthStatus struct {
	Status  string    `json:"status"`
	Service string    `json:"service"`
	Time    time.Time `json:"time"`
}

type HealthUseCase struct {
	serviceName string
	repository  healthRepository
	now         func() time.Time
}

func NewHealthUseCase(serviceName string, repository healthRepository) *HealthUseCase {
	return &HealthUseCase{
		serviceName: serviceName,
		repository:  repository,
		now:         time.Now,
	}
}

func (u *HealthUseCase) Check(ctx context.Context) (HealthStatus, error) {
	if err := u.repository.Ping(ctx); err != nil {
		return HealthStatus{}, fmt.Errorf("usecase health check: %w", err)
	}

	return HealthStatus{
		Status:  "ok",
		Service: u.serviceName,
		Time:    u.now().UTC(),
	}, nil
}
