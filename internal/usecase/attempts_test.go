package usecase

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"lms-arvand-backend/internal/domain"
)

type attemptSubmitRepoStub struct {
	course         domain.Course
	attempt        domain.Attempt
	attemptWindow  domain.AttemptWindow
	hasCertificate bool
	getCourseErr   error
	windowErr      error
	certificateErr error
	createErr      error
	getAttemptErr  error

	createCalled bool
	createParams domain.CreateAttemptRecordParams
}

func (r *attemptSubmitRepoStub) GetCourseForAttempt(ctx context.Context, courseID string) (domain.Course, error) {
	return r.course, r.getCourseErr
}

func (r *attemptSubmitRepoStub) GetUserCourseAttemptWindow(ctx context.Context, courseID, userID string, since *time.Time) (domain.AttemptWindow, error) {
	if r.windowErr != nil {
		return domain.AttemptWindow{}, r.windowErr
	}
	return r.attemptWindow, nil
}

func (r *attemptSubmitRepoStub) UserHasCourseCertificate(ctx context.Context, courseID, userID string) (bool, error) {
	if r.certificateErr != nil {
		return false, r.certificateErr
	}
	return r.hasCertificate, nil
}

func (r *attemptSubmitRepoStub) CreateAttempt(ctx context.Context, params domain.CreateAttemptRecordParams) (domain.Attempt, error) {
	r.createCalled = true
	r.createParams = params
	if r.createErr != nil {
		return domain.Attempt{}, r.createErr
	}

	attempt := r.attempt
	if attempt.ID == "" {
		attempt.ID = "attempt-created"
	}
	if attempt.CourseID == "" {
		attempt.CourseID = params.CourseID
	}
	if attempt.UserID == nil {
		userID := params.UserID
		attempt.UserID = &userID
	}
	attempt.Passed = params.Passed
	attempt.TotalEarned = params.TotalEarned
	attempt.TotalMax = params.TotalMax
	attempt.ScorePercent = params.ScorePercent

	return attempt, nil
}

func (r *attemptSubmitRepoStub) GetAttemptByID(ctx context.Context, attemptID string) (domain.Attempt, error) {
	if r.getAttemptErr != nil {
		return domain.Attempt{}, r.getAttemptErr
	}
	return r.attempt, nil
}

func (r *attemptSubmitRepoStub) ListAttempts(ctx context.Context, filter domain.AttemptListFilter) ([]domain.Attempt, int, error) {
	panic("unexpected ListAttempts call")
}

type attemptEnrollmentLookupStub struct {
	enrollment domain.Enrollment
	err        error
	called     bool
	courseID   string
	userID     string
}

func (l *attemptEnrollmentLookupStub) GetLatestByCourseAndUser(ctx context.Context, courseID, userID string) (domain.Enrollment, error) {
	l.called = true
	l.courseID = courseID
	l.userID = userID
	if l.err != nil {
		return domain.Enrollment{}, l.err
	}
	return l.enrollment, nil
}

type attemptAutoIssuerStub struct {
	called       bool
	enrollmentID string
	certificate  *domain.Certificate
	err          error
}

func (s *attemptAutoIssuerStub) TryAutoIssueForEnrollment(ctx context.Context, enrollmentID string) (*domain.Certificate, error) {
	s.called = true
	s.enrollmentID = enrollmentID
	if s.err != nil {
		return nil, s.err
	}
	return s.certificate, nil
}

func TestAttemptUseCaseSubmitAutoIssuesCertificate(t *testing.T) {
	course := domain.Course{
		ID:                 "course-1",
		QuizPassPercent:    50,
		MaxAttempts:        3,
		RetakeCooldownDays: 30,
		Questions: []domain.Question{
			{
				ID:     "q1",
				Type:   domain.QuestionTypeSingleChoice,
				Points: 10,
				Config: mustJSON(t, map[string]any{
					"options": []map[string]any{
						{"id": "a", "is_correct": true},
						{"id": "b", "is_correct": false},
					},
				}),
			},
		},
	}

	repo := &attemptSubmitRepoStub{
		course:  course,
		attempt: domain.Attempt{ID: "attempt-submit-1"},
	}
	enrollmentLookup := &attemptEnrollmentLookupStub{
		enrollment: domain.Enrollment{ID: "enrollment-1"},
	}
	autoIssuer := &attemptAutoIssuerStub{
		certificate: &domain.Certificate{ID: "cert-1"},
	}

	uc := NewAttemptUseCase(repo).
		WithEnrollmentLookup(enrollmentLookup).
		WithCertificateAutoIssuer(autoIssuer)

	result, err := uc.Submit(context.Background(), domain.SubmitAttemptParams{
		CourseID: course.ID,
		UserID:   "user-1",
		Answers: []domain.AttemptAnswer{
			{QuestionID: "q1", SelectedOptionIDs: []string{"a"}},
		},
	})
	if err != nil {
		t.Fatalf("submit returned error: %v", err)
	}

	if !repo.createCalled {
		t.Fatalf("expected attempt create to be called")
	}
	if !enrollmentLookup.called {
		t.Fatalf("expected enrollment lookup to be called")
	}
	if enrollmentLookup.courseID != "course-1" || enrollmentLookup.userID != "user-1" {
		t.Fatalf("unexpected enrollment lookup args: course_id=%q user_id=%q", enrollmentLookup.courseID, enrollmentLookup.userID)
	}
	if !autoIssuer.called {
		t.Fatalf("expected auto issuer to be called")
	}
	if autoIssuer.enrollmentID != "enrollment-1" {
		t.Fatalf("unexpected auto issuer enrollment id: %q", autoIssuer.enrollmentID)
	}
	if !result.Passed {
		t.Fatalf("expected passed attempt")
	}
}

func TestAttemptUseCaseSubmitDoesNotPassBelowPassingPercent(t *testing.T) {
	course := domain.Course{
		ID:              "course-2",
		QuizPassPercent: 80,
		MaxAttempts:     3,
		Questions: []domain.Question{
			{
				ID:     "q1",
				Type:   domain.QuestionTypeSingleChoice,
				Points: 10,
				Config: mustJSON(t, map[string]any{
					"options": []map[string]any{
						{"id": "a", "is_correct": true},
						{"id": "b", "is_correct": false},
					},
				}),
			},
		},
	}

	repo := &attemptSubmitRepoStub{course: course}
	autoIssuer := &attemptAutoIssuerStub{}
	uc := NewAttemptUseCase(repo).WithCertificateAutoIssuer(autoIssuer)

	result, err := uc.Submit(context.Background(), domain.SubmitAttemptParams{
		CourseID: course.ID,
		UserID:   "user-2",
		Answers: []domain.AttemptAnswer{
			{QuestionID: "q1", SelectedOptionIDs: []string{"b"}},
		},
	})
	if err != nil {
		t.Fatalf("submit returned error: %v", err)
	}
	if result.Passed {
		t.Fatalf("expected attempt below passing percent to fail")
	}
	if autoIssuer.called {
		t.Fatalf("auto issuer must not run for failed attempt")
	}
}

func TestAttemptUseCaseSubmitBlocksWhenCertificateAlreadyIssued(t *testing.T) {
	course := domain.Course{
		ID:          "course-3",
		MaxAttempts: 3,
		Questions: []domain.Question{{
			ID:     "q1",
			Type:   domain.QuestionTypeSingleChoice,
			Points: 1,
			Config: mustJSON(t, map[string]any{
				"options": []map[string]any{
					{"id": "a", "is_correct": true},
				},
			}),
		}},
	}

	repo := &attemptSubmitRepoStub{course: course, hasCertificate: true}
	uc := NewAttemptUseCase(repo)

	_, err := uc.Submit(context.Background(), domain.SubmitAttemptParams{
		CourseID: course.ID,
		UserID:   "user-3",
		Answers:  []domain.AttemptAnswer{{QuestionID: "q1", SelectedOptionIDs: []string{"a"}}},
	})
	if err == nil {
		t.Fatalf("expected submit to be blocked after certificate")
	}
	if !strings.Contains(err.Error(), "Сертификат") {
		t.Fatalf("expected certificate error, got %v", err)
	}
	if repo.createCalled {
		t.Fatalf("attempt must not be created after certificate")
	}
}

func TestAttemptUseCaseSubmitBlocksAttemptsUntilCooldownExpires(t *testing.T) {
	course := domain.Course{
		ID:                 "course-4",
		MaxAttempts:        3,
		RetakeCooldownDays: 30,
		Questions: []domain.Question{{
			ID:     "q1",
			Type:   domain.QuestionTypeSingleChoice,
			Points: 1,
			Config: mustJSON(t, map[string]any{
				"options": []map[string]any{
					{"id": "a", "is_correct": true},
				},
			}),
		}},
	}

	repo := &attemptSubmitRepoStub{
		course: course,
		attemptWindow: domain.AttemptWindow{
			Count:             3,
			EarliestStartedAt: ptrTime(time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)),
		},
	}
	uc := NewAttemptUseCase(repo)
	uc.now = func() time.Time { return time.Date(2026, 5, 10, 10, 0, 0, 0, time.UTC) }

	_, err := uc.Submit(context.Background(), domain.SubmitAttemptParams{
		CourseID: course.ID,
		UserID:   "user-4",
		Answers:  []domain.AttemptAnswer{{QuestionID: "q1", SelectedOptionIDs: []string{"a"}}},
	})
	if err == nil {
		t.Fatalf("expected submit to be blocked by cooldown")
	}
	if !strings.Contains(err.Error(), "31.05.2026") {
		t.Fatalf("expected cooldown date in error, got %v", err)
	}
	if repo.createCalled {
		t.Fatalf("attempt must not be created while cooldown is active")
	}
}

func ptr[T any](value T) *T {
	return &value
}

func ptrTime(value time.Time) *time.Time {
	return &value
}

func mustJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}

	return data
}

func almostEqual(left, right float64) bool {
	const epsilon = 0.0001
	if left > right {
		return left-right < epsilon
	}
	return right-left < epsilon
}

var _ attemptRepository = (*attemptSubmitRepoStub)(nil)
var _ attemptEnrollmentLookup = (*attemptEnrollmentLookupStub)(nil)
var _ attemptCertificateAutoIssuer = (*attemptAutoIssuerStub)(nil)
