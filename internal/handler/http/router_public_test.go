package http

import (
	"context"
	"log/slog"
	nethttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"lms-arvand-backend/internal/domain"
	"lms-arvand-backend/internal/usecase"
)

type routerHealthStub struct{}

func (routerHealthStub) Check(context.Context) (usecase.HealthStatus, error) {
	return usecase.HealthStatus{Status: "ok", Service: "QUIZ", Time: time.Now().UTC()}, nil
}

type routerCourseUseCaseStub struct{}

func (routerCourseUseCaseStub) Create(context.Context, domain.CreateCourseParams) (domain.Course, error) {
	return domain.Course{}, nil
}

func (routerCourseUseCaseStub) GetByID(context.Context, string) (domain.Course, error) {
	return domain.Course{
		ID:          "course-1",
		Title:       domain.MultiLangText{RU: "Course", TJ: "Course"},
		Description: domain.MultiLangText{RU: "Description", TJ: "Description"},
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}, nil
}

func (routerCourseUseCaseStub) List(context.Context, domain.CourseListFilter) ([]domain.Course, int, error) {
	return []domain.Course{{
		ID:          "course-1",
		Title:       domain.MultiLangText{RU: "Course", TJ: "Course"},
		Description: domain.MultiLangText{RU: "Description", TJ: "Description"},
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}}, 1, nil
}

func (routerCourseUseCaseStub) Update(context.Context, domain.UpdateCourseParams) (domain.Course, error) {
	return domain.Course{}, nil
}

func (routerCourseUseCaseStub) Archive(context.Context, string) error {
	return nil
}

type routerQuizUseCaseStub struct{}

func (routerQuizUseCaseStub) Create(context.Context, domain.CreateQuizParams) (domain.Quiz, error) {
	return domain.Quiz{}, nil
}

func (routerQuizUseCaseStub) GetByID(context.Context, string) (domain.Quiz, error) {
	return domain.Quiz{
		ID:                 "quiz-1",
		Title:              domain.MultiLangText{RU: "Quiz", TJ: "Quiz"},
		Description:        domain.MultiLangText{RU: "Description", TJ: "Description"},
		PassingPoints:      8,
		MaxAttempts:        3,
		RetakeCooldownDays: 30,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}, nil
}

func (routerQuizUseCaseStub) List(context.Context, domain.QuizListFilter) ([]domain.Quiz, int, error) {
	return nil, 0, nil
}

func (routerQuizUseCaseStub) Update(context.Context, domain.UpdateQuizParams) (domain.Quiz, error) {
	return domain.Quiz{}, nil
}

func (routerQuizUseCaseStub) Archive(context.Context, string) error {
	return nil
}

type routerCertificateUseCaseStub struct{}

func (routerCertificateUseCaseStub) Create(context.Context, domain.CreateCertificateParams) (domain.Certificate, error) {
	return domain.Certificate{}, nil
}

func (routerCertificateUseCaseStub) GetByID(context.Context, string) (domain.Certificate, error) {
	return domain.Certificate{
		ID:           "certificate-1",
		EnrollmentID: "enrollment-1",
		UserID:       "user-1",
		CourseID:     "course-1",
		AttemptID:    "attempt-1",
		SerialNumber: "001-002-003",
		VerifyHash:   "hash",
		IssuedAt:     time.Now().UTC(),
		CourseTitle:  domain.MultiLangText{RU: "Course", TJ: "Course"},
	}, nil
}

func (routerCertificateUseCaseStub) GetByVerifyHash(context.Context, string) (domain.Certificate, error) {
	return domain.Certificate{}, nil
}

func (routerCertificateUseCaseStub) List(context.Context, domain.CertificateListFilter) ([]domain.Certificate, int, error) {
	return nil, 0, nil
}

func TestRouterPublicCatalogRoutesDoNotRequireAuth(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(testingWriter{t: t}, nil))
	router := NewRouter(
		logger,
		NewHealthHandler(logger, routerHealthStub{}),
		nil,
		nil,
		nil,
		NewCoursesHandler(logger, routerCourseUseCaseStub{}),
		NewQuizzesHandler(logger, routerQuizUseCaseStub{}),
		nil,
		nil,
		NewCertificatesHandler(logger, routerCertificateUseCaseStub{}),
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		"",
	)

	for _, path := range []string{
		"/api/v1/courses",
		"/api/v1/courses/course-1",
		"/api/v1/quizzes/quiz-1",
		"/api/v1/certificates/certificate-1",
		"/api/v1/certificate/certificate-1",
	} {
		request := httptest.NewRequest(nethttp.MethodGet, path, nil)
		response := httptest.NewRecorder()

		router.ServeHTTP(response, request)

		if response.Code != nethttp.StatusOK {
			t.Fatalf("%s status = %d, want %d; body=%s", path, response.Code, nethttp.StatusOK, response.Body.String())
		}
	}
}

type testingWriter struct {
	t *testing.T
}

func (w testingWriter) Write(p []byte) (int, error) {
	w.t.Log(string(p))
	return len(p), nil
}
