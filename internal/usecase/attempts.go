package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"lms-arvand-backend/internal/domain"
)

type attemptRepository interface {
	GetCourseForAttempt(ctx context.Context, courseID string) (domain.Course, error)
	GetUserCourseAttemptWindow(ctx context.Context, courseID, userID string, since *time.Time) (domain.AttemptWindow, error)
	UserHasCourseCertificate(ctx context.Context, courseID, userID string) (bool, error)
	CreateAttempt(ctx context.Context, params domain.CreateAttemptRecordParams) (domain.Attempt, error)
	GetAttemptByID(ctx context.Context, attemptID string) (domain.Attempt, error)
	ListAttempts(ctx context.Context, filter domain.AttemptListFilter) ([]domain.Attempt, int, error)
}

type attemptEnrollmentLookup interface {
	GetLatestByCourseAndUser(ctx context.Context, courseID, userID string) (domain.Enrollment, error)
}

type attemptCertificateAutoIssuer interface {
	TryAutoIssueForEnrollment(ctx context.Context, enrollmentID string) (*domain.Certificate, error)
}

type AttemptUseCase struct {
	repository       attemptRepository
	enrollmentLookup attemptEnrollmentLookup
	autoIssuer       attemptCertificateAutoIssuer
	now              func() time.Time
	audit            *AuditLogger
}

func NewAttemptUseCase(repository attemptRepository) *AttemptUseCase {
	return &AttemptUseCase{
		repository: repository,
		now:        time.Now,
	}
}

func (u *AttemptUseCase) WithAudit(audit *AuditLogger) *AttemptUseCase {
	u.audit = audit
	return u
}

func (u *AttemptUseCase) WithEnrollmentLookup(enrollmentLookup attemptEnrollmentLookup) *AttemptUseCase {
	u.enrollmentLookup = enrollmentLookup
	return u
}

func (u *AttemptUseCase) WithCertificateAutoIssuer(autoIssuer attemptCertificateAutoIssuer) *AttemptUseCase {
	u.autoIssuer = autoIssuer
	return u
}

func (u *AttemptUseCase) Submit(ctx context.Context, params domain.SubmitAttemptParams) (domain.Attempt, error) {
	normalized, err := normalizeSubmitAttemptParams(params)
	if err != nil {
		return domain.Attempt{}, fmt.Errorf("usecase attempts submit: %w", err)
	}

	course, err := u.repository.GetCourseForAttempt(ctx, normalized.CourseID)
	if err != nil {
		return domain.Attempt{}, fmt.Errorf("usecase attempts submit course load: %w", err)
	}

	if err := u.ensureUserCanSubmit(ctx, course, normalized.UserID); err != nil {
		return domain.Attempt{}, fmt.Errorf("usecase attempts submit access check: %w", err)
	}

	finishedAt := u.now().UTC()
	attemptSince := attemptWindowStart(finishedAt, course.RetakeCooldownDays)
	attemptWindow, err := u.repository.GetUserCourseAttemptWindow(ctx, normalized.CourseID, normalized.UserID, attemptSince)
	if err != nil {
		return domain.Attempt{}, fmt.Errorf("usecase attempts submit count attempts: %w", err)
	}

	maxAttempts := effectiveMaxAttempts(course.MaxAttempts)
	if attemptWindow.Count >= maxAttempts {
		return domain.Attempt{}, attemptLimitExceededError(attemptWindow, course.RetakeCooldownDays)
	}

	questionsSnapshot, err := json.Marshal(course.Questions)
	if err != nil {
		return domain.Attempt{}, fmt.Errorf("usecase attempts submit marshal questions snapshot: %w", err)
	}

	answersData, err := json.Marshal(normalized.Answers)
	if err != nil {
		return domain.Attempt{}, fmt.Errorf("usecase attempts submit marshal answers data: %w", err)
	}

	totalEarned, totalMax := evaluateAttempt(course, normalized.Answers)
	scorePercent := 0.0
	if totalMax > 0 {
		scorePercent = roundToTwo(totalEarned / totalMax * 100)
	}

	passingPercent := effectivePassingPercent(course)
	passed := scorePercent >= passingPercent

	startedAt := finishedAt
	if normalized.StartedAt != nil {
		startedAt = normalized.StartedAt.UTC()
	}

	attempt, err := u.repository.CreateAttempt(ctx, domain.CreateAttemptRecordParams{
		CourseID:          normalized.CourseID,
		UserID:            normalized.UserID,
		StartedAt:         startedAt,
		FinishedAt:        finishedAt,
		QuestionsSnapshot: questionsSnapshot,
		AnswersData:       answersData,
		TotalEarned:       roundToTwo(totalEarned),
		TotalMax:          roundToTwo(totalMax),
		ScorePercent:      scorePercent,
		Passed:            passed,
	})
	if err != nil {
		return domain.Attempt{}, fmt.Errorf("usecase attempts submit create attempt: %w", err)
	}

	if u.audit != nil {
		u.audit.Log(ctx, domain.AppEventAttemptFinished, map[string]any{
			"attempt_id":      attempt.ID,
			"course_id":       attempt.CourseID,
			"user_id":         attempt.UserID,
			"score_percent":   attempt.ScorePercent,
			"total_earned":    attempt.TotalEarned,
			"passing_percent": passingPercent,
			"passed":          attempt.Passed,
		})

		eventType := domain.AppEventAttemptFailed
		if attempt.Passed {
			eventType = domain.AppEventAttemptPassed
		}
		u.audit.Log(ctx, eventType, map[string]any{
			"attempt_id":      attempt.ID,
			"course_id":       attempt.CourseID,
			"user_id":         attempt.UserID,
			"score_percent":   attempt.ScorePercent,
			"passing_percent": passingPercent,
		})
	}

	u.tryAutoIssueCertificate(ctx, course, attempt)

	return attempt, nil
}

func (u *AttemptUseCase) GetByID(ctx context.Context, attemptID string) (domain.Attempt, error) {
	attemptID = strings.TrimSpace(attemptID)
	if attemptID == "" {
		return domain.Attempt{}, fmt.Errorf("usecase attempts get by id: %w", domain.ErrValidation)
	}

	attempt, err := u.repository.GetAttemptByID(ctx, attemptID)
	if err != nil {
		return domain.Attempt{}, fmt.Errorf("usecase attempts get by id: %w", err)
	}

	return attempt, nil
}

func (u *AttemptUseCase) List(ctx context.Context, filter domain.AttemptListFilter) ([]domain.Attempt, int, error) {
	normalized, err := normalizeAttemptListFilter(filter)
	if err != nil {
		return nil, 0, fmt.Errorf("usecase attempts list: %w", err)
	}

	attempts, total, err := u.repository.ListAttempts(ctx, normalized)
	if err != nil {
		return nil, 0, fmt.Errorf("usecase attempts list: %w", err)
	}

	return attempts, total, nil
}

func (u *AttemptUseCase) tryAutoIssueCertificate(ctx context.Context, course domain.Course, attempt domain.Attempt) {
	if u.autoIssuer == nil || u.enrollmentLookup == nil {
		return
	}

	if !attempt.Passed {
		return
	}

	if attempt.UserID == nil || strings.TrimSpace(*attempt.UserID) == "" {
		return
	}

	userID := strings.TrimSpace(*attempt.UserID)

	enrollment, err := u.enrollmentLookup.GetLatestByCourseAndUser(ctx, course.ID, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return
		}

		slog.ErrorContext(ctx, "attempt certificate enrollment lookup failed",
			slog.String("attempt_id", attempt.ID),
			slog.String("course_id", attempt.CourseID),
			slog.String("user_id", userID),
			slog.String("error", err.Error()),
		)
		return
	}

	certificate, err := u.autoIssuer.TryAutoIssueForEnrollment(ctx, enrollment.ID)
	if err != nil {
		slog.ErrorContext(ctx, "attempt certificate auto issue failed",
			slog.String("attempt_id", attempt.ID),
			slog.String("course_id", attempt.CourseID),
			slog.String("enrollment_id", enrollment.ID),
			slog.String("error", err.Error()),
		)
		return
	}

	if certificate != nil {
		slog.InfoContext(ctx, "attempt certificate auto issued",
			slog.String("attempt_id", attempt.ID),
			slog.String("course_id", attempt.CourseID),
			slog.String("enrollment_id", enrollment.ID),
			slog.String("certificate_id", certificate.ID),
		)
	}
}

func (u *AttemptUseCase) ensureUserCanSubmit(ctx context.Context, course domain.Course, userID string) error {
	hasCertificate, err := u.repository.UserHasCourseCertificate(ctx, course.ID, userID)
	if err != nil {
		return err
	}

	if hasCertificate {
		return domain.ConflictError("Сертификат по этому курсу уже выдан. Видео доступно, повторная сдача теста закрыта.")
	}

	return nil
}

func attemptWindowStart(now time.Time, cooldownDays int) *time.Time {
	if cooldownDays <= 0 {
		return nil
	}
	start := now.AddDate(0, 0, -cooldownDays)
	return &start
}

func effectiveMaxAttempts(maxAttempts int) int {
	if maxAttempts <= 0 {
		return 3
	}
	return maxAttempts
}

func attemptLimitExceededError(window domain.AttemptWindow, cooldownDays int) error {
	if cooldownDays > 0 && window.EarliestStartedAt != nil {
		unlockAt := window.EarliestStartedAt.AddDate(0, 0, cooldownDays)
		return domain.ConflictError(fmt.Sprintf(
			"Лимит попыток исчерпан. Повторная сдача будет доступна после %s.",
			unlockAt.Format("02.01.2006"),
		))
	}
	return domain.ConflictError("Лимит попыток исчерпан. Повторная сдача теста недоступна.")
}

func effectivePassingPercent(course domain.Course) float64 {
	if course.QuizPassPercent > 0 {
		return float64(course.QuizPassPercent)
	}
	return float64(defaultQuizPassingScore)
}

func normalizeSubmitAttemptParams(params domain.SubmitAttemptParams) (domain.SubmitAttemptParams, error) {
	params.CourseID = strings.TrimSpace(params.CourseID)
	params.UserID = strings.TrimSpace(params.UserID)

	if params.CourseID == "" {
		return domain.SubmitAttemptParams{}, fmt.Errorf("course_id is required: %w", domain.ErrValidation)
	}

	if params.UserID == "" {
		return domain.SubmitAttemptParams{}, fmt.Errorf("user_id is required: %w", domain.ErrValidation)
	}

	if len(params.Answers) == 0 {
		return domain.SubmitAttemptParams{}, fmt.Errorf("answers are required: %w", domain.ErrValidation)
	}

	normalizedAnswers := make([]domain.AttemptAnswer, 0, len(params.Answers))
	seen := make(map[string]struct{}, len(params.Answers))

	for _, answer := range params.Answers {
		answer.QuestionID = strings.TrimSpace(answer.QuestionID)
		answer.TextAnswer = normalizeOptionalString(answer.TextAnswer)
		answer.SelectedOptionIDs = normalizeStringSlice(answer.SelectedOptionIDs)

		if answer.QuestionID == "" {
			return domain.SubmitAttemptParams{}, fmt.Errorf("question_id is required: %w", domain.ErrValidation)
		}

		if _, exists := seen[answer.QuestionID]; exists {
			return domain.SubmitAttemptParams{}, fmt.Errorf("duplicate answer for question: %w", domain.ErrValidation)
		}
		seen[answer.QuestionID] = struct{}{}

		normalizedAnswers = append(normalizedAnswers, answer)
	}

	params.Answers = normalizedAnswers
	return params, nil
}

func normalizeAttemptListFilter(filter domain.AttemptListFilter) (domain.AttemptListFilter, error) {
	if filter.CourseID != nil {
		filter.CourseID = normalizeOptionalString(filter.CourseID)
	}

	if filter.UserID != nil {
		filter.UserID = normalizeOptionalString(filter.UserID)
	}

	if filter.Limit <= 0 {
		filter.Limit = 20
	}

	if filter.Limit > 100 {
		filter.Limit = 100
	}

	if filter.Offset < 0 {
		return domain.AttemptListFilter{}, fmt.Errorf("offset must be non-negative: %w", domain.ErrValidation)
	}

	return filter, nil
}

func evaluateAttempt(course domain.Course, answers []domain.AttemptAnswer) (float64, float64) {
	answerMap := make(map[string]domain.AttemptAnswer, len(answers))
	for _, answer := range answers {
		answerMap[answer.QuestionID] = answer
	}

	totalEarned := 0.0
	totalMax := 0.0

	for _, question := range course.Questions {
		totalMax += question.Points

		answer, exists := answerMap[question.ID]
		if !exists {
			continue
		}

		earned := evaluateQuestion(question, answer)
		totalEarned += earned
	}

	return totalEarned, totalMax
}

func evaluateQuestion(question domain.Question, answer domain.AttemptAnswer) float64 {
	switch question.Type {
	case domain.QuestionTypeSingleChoice:
		correctIDs, ok := extractCorrectOptionIDs(question.Config)
		if !ok || len(correctIDs) != 1 || len(answer.SelectedOptionIDs) != 1 {
			return 0
		}
		if correctIDs[0] == answer.SelectedOptionIDs[0] {
			return question.Points
		}
		return 0
	case domain.QuestionTypeMultipleChoice:
		correctIDs, ok := extractCorrectOptionIDs(question.Config)
		if !ok {
			return 0
		}
		selected := normalizeStringSlice(answer.SelectedOptionIDs)
		if sameStringSet(correctIDs, selected) {
			return question.Points
		}
		return 0
	case domain.QuestionTypeTrueFalse:
		correctValue, ok := extractCorrectBoolean(question.Config)
		if !ok || answer.BooleanAnswer == nil {
			return 0
		}
		if correctValue == *answer.BooleanAnswer {
			return question.Points
		}
		return 0
	case domain.QuestionTypeShortAnswer, domain.QuestionTypeFillBlank:
		acceptedAnswers, ok := extractAcceptedAnswers(question.Config)
		if !ok || answer.TextAnswer == nil {
			return 0
		}
		candidate := strings.ToLower(strings.TrimSpace(*answer.TextAnswer))
		for _, accepted := range acceptedAnswers {
			if candidate == accepted {
				return question.Points
			}
		}
		return 0
	default:
		return 0
	}
}

func extractCorrectOptionIDs(config json.RawMessage) ([]string, bool) {
	var payload struct {
		Options []struct {
			ID             string `json:"id"`
			IsCorrect      bool   `json:"is_correct"`
			IsCorrectCamel bool   `json:"isCorrect"`
		} `json:"options"`
	}

	if err := json.Unmarshal(config, &payload); err != nil {
		return nil, false
	}

	ids := make([]string, 0)
	for _, option := range payload.Options {
		if option.IsCorrect || option.IsCorrectCamel {
			ids = append(ids, strings.TrimSpace(option.ID))
		}
	}

	return normalizeStringSlice(ids), true
}

func extractCorrectBoolean(config json.RawMessage) (bool, bool) {
	var payload struct {
		Correct *bool `json:"correct"`
	}

	if err := json.Unmarshal(config, &payload); err != nil {
		return false, false
	}

	if payload.Correct == nil {
		return false, false
	}

	return *payload.Correct, true
}

func extractAcceptedAnswers(config json.RawMessage) ([]string, bool) {
	var payload struct {
		AcceptedAnswers      []string `json:"accepted_answers"`
		AcceptedAnswersCamel []string `json:"acceptedAnswers"`
	}

	if err := json.Unmarshal(config, &payload); err != nil {
		return nil, false
	}

	answers := payload.AcceptedAnswers
	if len(answers) == 0 {
		answers = payload.AcceptedAnswersCamel
	}

	normalized := make([]string, 0, len(answers))
	for _, answer := range answers {
		trimmed := strings.ToLower(strings.TrimSpace(answer))
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}

	return normalized, true
}

func roundToTwo(value float64) float64 {
	return math.Round(value*100) / 100
}

func sameStringSet(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}

	index := make(map[string]struct{}, len(left))
	for _, value := range left {
		index[value] = struct{}{}
	}

	for _, value := range right {
		if _, exists := index[value]; !exists {
			return false
		}
	}

	return true
}
