package usecase

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"lms-arvand-backend/internal/domain"
)

type attemptReviewRepoStub struct {
	attempt       domain.Attempt
	quiz          domain.Quiz
	getAttemptErr error
	getQuizErr    error
	updateErr     error

	updateCalled bool
	updated      domain.ReviewAttemptParams
}

func (r *attemptReviewRepoStub) GetQuizForAttempt(ctx context.Context, quizID string) (domain.Quiz, error) {
	return r.quiz, r.getQuizErr
}

func (r *attemptReviewRepoStub) GetUserQuizAttemptWindow(ctx context.Context, quizID, userID string, since *time.Time) (domain.AttemptWindow, error) {
	panic("unexpected GetUserQuizAttemptWindow call")
}

func (r *attemptReviewRepoStub) UserHasCourseCertificate(ctx context.Context, courseID, userID string) (bool, error) {
	panic("unexpected UserHasCourseCertificate call")
}

func (r *attemptReviewRepoStub) CreateAttempt(ctx context.Context, params domain.CreateAttemptRecordParams) (domain.Attempt, error) {
	panic("unexpected CreateAttempt call")
}

func (r *attemptReviewRepoStub) GetAttemptByID(ctx context.Context, attemptID string) (domain.Attempt, error) {
	return r.attempt, r.getAttemptErr
}

func (r *attemptReviewRepoStub) ListAttempts(ctx context.Context, filter domain.AttemptListFilter) ([]domain.Attempt, int, error) {
	panic("unexpected ListAttempts call")
}

func (r *attemptReviewRepoStub) UpdateReview(ctx context.Context, params domain.ReviewAttemptParams) (domain.Attempt, error) {
	r.updateCalled = true
	r.updated = params
	if r.updateErr != nil {
		return domain.Attempt{}, r.updateErr
	}

	attempt := r.attempt
	attempt.Passed = params.Passed
	attempt.TotalEarned = params.TotalEarned
	attempt.ScorePercent = params.ScorePercent
	attempt.NeedsReview = false
	attempt.ReviewComment = params.Comment
	attempt.ReviewerID = &params.ReviewerID
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	attempt.ReviewedAt = &now
	attempt.ManualPassed = &params.Passed
	if len(params.ReviewScores) > 0 {
		if err := json.Unmarshal(params.ReviewScores, &attempt.ReviewScores); err != nil {
			return domain.Attempt{}, err
		}
	} else {
		attempt.ReviewScores = nil
	}

	return attempt, nil
}

type attemptSubmitRepoStub struct {
	quiz           domain.Quiz
	attempt        domain.Attempt
	attemptWindow  domain.AttemptWindow
	hasCertificate bool
	getQuizErr     error
	windowErr      error
	certificateErr error
	createErr      error
	getAttemptErr  error

	createCalled bool
	createParams domain.CreateAttemptRecordParams
}

func (r *attemptSubmitRepoStub) GetQuizForAttempt(ctx context.Context, quizID string) (domain.Quiz, error) {
	return r.quiz, r.getQuizErr
}

func (r *attemptSubmitRepoStub) GetUserQuizAttemptWindow(ctx context.Context, quizID, userID string, since *time.Time) (domain.AttemptWindow, error) {
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
	if attempt.QuizID == "" {
		attempt.QuizID = params.QuizID
	}
	if attempt.UserID == nil {
		userID := params.UserID
		attempt.UserID = &userID
	}
	attempt.Passed = params.Passed
	attempt.NeedsReview = params.NeedsReview
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

func (r *attemptSubmitRepoStub) UpdateReview(ctx context.Context, params domain.ReviewAttemptParams) (domain.Attempt, error) {
	panic("unexpected UpdateReview call")
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

func TestAttemptUseCaseReviewManualScoring(t *testing.T) {
	questions := []domain.Question{
		{
			ID:       "q1",
			Type:     domain.QuestionTypeSingleChoice,
			Points:   5,
			Required: true,
			Config: mustJSON(t, map[string]any{
				"options": []map[string]any{
					{"id": "a", "is_correct": true},
					{"id": "b", "is_correct": false},
				},
			}),
		},
		{
			ID:       "q2",
			Type:     domain.QuestionTypeCode,
			Points:   10,
			Required: true,
			Config:   mustJSON(t, map[string]any{}),
		},
	}

	current := domain.Attempt{
		ID:                "attempt-1",
		QuizID:            "quiz-1",
		UserID:            ptr("user-1"),
		StartedAt:         time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC),
		FinishedAt:        ptrTime(time.Date(2026, 5, 1, 10, 30, 0, 0, time.UTC)),
		QuestionsSnapshot: mustJSON(t, questions),
		AnswersData: mustJSON(t, []domain.AttemptAnswer{
			{QuestionID: "q1", SelectedOptionIDs: []string{"a"}},
			{QuestionID: "q2", TextAnswer: ptr("print(\"hello\")")},
		}),
		TotalEarned:  5,
		TotalMax:     15,
		ScorePercent: 33.33,
		Passed:       false,
		NeedsReview:  true,
	}

	repo := &attemptReviewRepoStub{
		attempt: current,
		quiz: domain.Quiz{
			ID:           "quiz-1",
			PassingScore: 80,
		},
	}

	uc := NewAttemptUseCase(repo)

	result, err := uc.Review(context.Background(), domain.ReviewAttemptParams{
		AttemptID:  current.ID,
		ReviewerID: "admin-1",
		Passed:     false,
		Comment:    ptr("manual review"),
		Scores: []domain.AttemptReviewScore{
			{
				QuestionID: "q2",
				Points:     8,
				Comment:    ptr("good answer"),
			},
		},
	})
	if err != nil {
		t.Fatalf("review returned error: %v", err)
	}

	if !repo.updateCalled {
		t.Fatalf("expected update review to be called")
	}

	if got, want := repo.updated.TotalEarned, 13.0; got != want {
		t.Fatalf("updated total earned = %v, want %v", got, want)
	}
	if got, want := repo.updated.ScorePercent, 86.67; got != want {
		t.Fatalf("updated score percent = %v, want %v", got, want)
	}
	if !repo.updated.Passed {
		t.Fatalf("expected manual review to pass")
	}
	var updatedScores []domain.AttemptReviewScore
	if err := json.Unmarshal(repo.updated.ReviewScores, &updatedScores); err != nil {
		t.Fatalf("unmarshal updated review scores: %v", err)
	}
	if got, want := len(updatedScores), 1; got != want {
		t.Fatalf("review scores count = %d, want %d", got, want)
	}
	if !result.Passed || !almostEqual(result.TotalEarned, 13) || !almostEqual(result.ScorePercent, 86.67) {
		t.Fatalf("unexpected result: %#v", result)
	}
	if result.ManualPassed == nil || !*result.ManualPassed {
		t.Fatalf("expected manual passed true")
	}
	if got, want := len(result.ReviewScores), 1; got != want {
		t.Fatalf("result review scores count = %d, want %d", got, want)
	}
}

func TestAttemptUseCaseReviewLegacyPassFail(t *testing.T) {
	current := domain.Attempt{
		ID:                "attempt-2",
		QuizID:            "quiz-2",
		UserID:            ptr("user-2"),
		StartedAt:         time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC),
		FinishedAt:        ptrTime(time.Date(2026, 5, 1, 10, 30, 0, 0, time.UTC)),
		QuestionsSnapshot: mustJSON(t, []domain.Question{}),
		AnswersData:       mustJSON(t, []domain.AttemptAnswer{}),
		TotalEarned:       4,
		TotalMax:          10,
		ScorePercent:      40,
		Passed:            false,
		NeedsReview:       true,
	}

	repo := &attemptReviewRepoStub{attempt: current}
	uc := NewAttemptUseCase(repo)

	result, err := uc.Review(context.Background(), domain.ReviewAttemptParams{
		AttemptID:  current.ID,
		ReviewerID: "admin-2",
		Passed:     true,
	})
	if err != nil {
		t.Fatalf("review returned error: %v", err)
	}

	if !repo.updateCalled {
		t.Fatalf("expected update review to be called")
	}
	if got, want := repo.updated.TotalEarned, current.TotalEarned; got != want {
		t.Fatalf("updated total earned = %v, want %v", got, want)
	}
	if got, want := repo.updated.ScorePercent, current.ScorePercent; got != want {
		t.Fatalf("updated score percent = %v, want %v", got, want)
	}
	if !repo.updated.Passed {
		t.Fatalf("expected legacy pass override to be preserved")
	}
	if string(repo.updated.ReviewScores) != "[]" {
		t.Fatalf("expected empty review scores payload in legacy mode, got %q", string(repo.updated.ReviewScores))
	}
	if !result.Passed || !almostEqual(result.TotalEarned, current.TotalEarned) || !almostEqual(result.ScorePercent, current.ScorePercent) {
		t.Fatalf("unexpected result: %#v", result)
	}
	if result.ManualPassed == nil || !*result.ManualPassed {
		t.Fatalf("expected manual passed true")
	}
}

func TestAttemptUseCaseReviewManualScoringMissingScore(t *testing.T) {
	questions := []domain.Question{
		{
			ID:       "q1",
			Type:     domain.QuestionTypeCode,
			Points:   10,
			Required: true,
			Config:   mustJSON(t, map[string]any{}),
		},
		{
			ID:       "q2",
			Type:     domain.QuestionTypeLongText,
			Points:   5,
			Required: true,
			Config:   mustJSON(t, map[string]any{}),
		},
	}

	current := domain.Attempt{
		ID:                "attempt-3",
		QuizID:            "quiz-3",
		UserID:            ptr("user-3"),
		StartedAt:         time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC),
		FinishedAt:        ptrTime(time.Date(2026, 5, 1, 10, 30, 0, 0, time.UTC)),
		QuestionsSnapshot: mustJSON(t, questions),
		AnswersData:       mustJSON(t, []domain.AttemptAnswer{}),
		NeedsReview:       true,
	}

	repo := &attemptReviewRepoStub{
		attempt: current,
		quiz:    domain.Quiz{ID: "quiz-3", PassingScore: 70},
	}
	uc := NewAttemptUseCase(repo)

	_, err := uc.Review(context.Background(), domain.ReviewAttemptParams{
		AttemptID:  current.ID,
		ReviewerID: "admin-3",
		Scores: []domain.AttemptReviewScore{
			{
				QuestionID: "q1",
				Points:     4,
			},
		},
	})
	if err == nil {
		t.Fatalf("expected manual review to fail when score is missing")
	}
	if repo.updateCalled {
		t.Fatalf("update review should not be called on validation error")
	}
}

func TestAttemptUseCaseSubmitAutoIssuesCertificate(t *testing.T) {
	quiz := domain.Quiz{
		ID:           "quiz-submit-1",
		CourseID:     ptr("course-1"),
		PassingScore: 50,
		MaxAttempts:  3,
		Questions: []domain.Question{
			{
				ID:       "q1",
				Type:     domain.QuestionTypeSingleChoice,
				Points:   10,
				Required: true,
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
		quiz: quiz,
		attempt: domain.Attempt{
			ID: "attempt-submit-1",
		},
	}
	enrollmentLookup := &attemptEnrollmentLookupStub{
		enrollment: domain.Enrollment{
			ID: "enrollment-1",
		},
	}
	autoIssuer := &attemptAutoIssuerStub{
		certificate: &domain.Certificate{ID: "cert-1"},
	}

	uc := NewAttemptUseCase(repo).
		WithEnrollmentLookup(enrollmentLookup).
		WithCertificateAutoIssuer(autoIssuer)

	result, err := uc.Submit(context.Background(), domain.SubmitAttemptParams{
		QuizID: quiz.ID,
		UserID: "user-1",
		Answers: []domain.AttemptAnswer{
			{
				QuestionID:        "q1",
				SelectedOptionIDs: []string{"a"},
			},
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

func TestAttemptUseCaseSubmitDoesNotPassBelowPassingPoints(t *testing.T) {
	quiz := domain.Quiz{
		ID:            "quiz-submit-2",
		CourseID:      ptr("course-2"),
		PassingPoints: 8,
		MaxAttempts:   3,
		Questions: []domain.Question{
			{
				ID:       "q1",
				Type:     domain.QuestionTypeSingleChoice,
				Points:   10,
				Required: true,
				Config: mustJSON(t, map[string]any{
					"options": []map[string]any{
						{"id": "a", "is_correct": true},
						{"id": "b", "is_correct": false},
					},
				}),
			},
		},
	}

	repo := &attemptSubmitRepoStub{quiz: quiz}
	autoIssuer := &attemptAutoIssuerStub{}
	uc := NewAttemptUseCase(repo).WithCertificateAutoIssuer(autoIssuer)

	result, err := uc.Submit(context.Background(), domain.SubmitAttemptParams{
		QuizID: quiz.ID,
		UserID: "user-2",
		Answers: []domain.AttemptAnswer{
			{
				QuestionID:        "q1",
				SelectedOptionIDs: []string{"b"},
			},
		},
	})
	if err != nil {
		t.Fatalf("submit returned error: %v", err)
	}
	if result.Passed {
		t.Fatalf("expected attempt below passing points to fail")
	}
	if autoIssuer.called {
		t.Fatalf("auto issuer must not run for failed attempt")
	}
}

func TestAttemptUseCaseSubmitBlocksWhenCertificateAlreadyIssued(t *testing.T) {
	quiz := domain.Quiz{
		ID:          "quiz-submit-3",
		CourseID:    ptr("course-3"),
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

	repo := &attemptSubmitRepoStub{quiz: quiz, hasCertificate: true}
	uc := NewAttemptUseCase(repo)

	_, err := uc.Submit(context.Background(), domain.SubmitAttemptParams{
		QuizID: quiz.ID,
		UserID: "user-3",
		Answers: []domain.AttemptAnswer{
			{QuestionID: "q1", SelectedOptionIDs: []string{"a"}},
		},
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
	quiz := domain.Quiz{
		ID:                 "quiz-submit-4",
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
		quiz: quiz,
		attemptWindow: domain.AttemptWindow{
			Count:             3,
			EarliestStartedAt: ptrTime(time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)),
		},
	}
	uc := NewAttemptUseCase(repo)
	uc.now = func() time.Time { return time.Date(2026, 5, 10, 10, 0, 0, 0, time.UTC) }

	_, err := uc.Submit(context.Background(), domain.SubmitAttemptParams{
		QuizID: quiz.ID,
		UserID: "user-4",
		Answers: []domain.AttemptAnswer{
			{QuestionID: "q1", SelectedOptionIDs: []string{"a"}},
		},
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

var _ attemptRepository = (*attemptReviewRepoStub)(nil)
var _ attemptRepository = (*attemptSubmitRepoStub)(nil)
var _ attemptEnrollmentLookup = (*attemptEnrollmentLookupStub)(nil)
var _ attemptCertificateAutoIssuer = (*attemptAutoIssuerStub)(nil)
