package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"lms-arvand-backend/internal/domain"
)

type attemptRepository interface {
	GetQuizForAttempt(ctx context.Context, quizID string) (domain.Quiz, error)
	CountUserQuizAttempts(ctx context.Context, quizID, userID string) (int, error)
	CreateAttempt(ctx context.Context, params domain.CreateAttemptRecordParams) (domain.Attempt, error)
	GetAttemptByID(ctx context.Context, attemptID string) (domain.Attempt, error)
	ListAttempts(ctx context.Context, filter domain.AttemptListFilter) ([]domain.Attempt, int, error)
	UpdateReview(ctx context.Context, params domain.ReviewAttemptParams) (domain.Attempt, error)
}

type AttemptUseCase struct {
	repository attemptRepository
	now        func() time.Time
	audit      *AuditLogger
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

func (u *AttemptUseCase) Submit(ctx context.Context, params domain.SubmitAttemptParams) (domain.Attempt, error) {
	normalized, err := normalizeSubmitAttemptParams(params)
	if err != nil {
		return domain.Attempt{}, fmt.Errorf("usecase attempts submit: %w", err)
	}

	quiz, err := u.repository.GetQuizForAttempt(ctx, normalized.QuizID)
	if err != nil {
		return domain.Attempt{}, fmt.Errorf("usecase attempts submit quiz load: %w", err)
	}

	attemptCount, err := u.repository.CountUserQuizAttempts(ctx, normalized.QuizID, normalized.UserID)
	if err != nil {
		return domain.Attempt{}, fmt.Errorf("usecase attempts submit count attempts: %w", err)
	}

	if attemptCount >= quiz.MaxAttempts {
		return domain.Attempt{}, fmt.Errorf("max attempts exceeded: %w", domain.ErrConflict)
	}

	questionsSnapshot, err := json.Marshal(quiz.Questions)
	if err != nil {
		return domain.Attempt{}, fmt.Errorf("usecase attempts submit marshal questions snapshot: %w", err)
	}

	answersData, err := json.Marshal(normalized.Answers)
	if err != nil {
		return domain.Attempt{}, fmt.Errorf("usecase attempts submit marshal answers data: %w", err)
	}

	totalEarned, totalMax, needsReview := evaluateAttempt(quiz, normalized.Answers)
	scorePercent := 0.0
	if totalMax > 0 {
		scorePercent = roundToTwo(totalEarned / totalMax * 100)
	}

	passed := !needsReview && scorePercent >= float64(quiz.PassingScore)
	finishedAt := u.now().UTC()
	startedAt := finishedAt
	if normalized.StartedAt != nil {
		startedAt = normalized.StartedAt.UTC()
	}

	attempt, err := u.repository.CreateAttempt(ctx, domain.CreateAttemptRecordParams{
		QuizID:            normalized.QuizID,
		UserID:            normalized.UserID,
		StartedAt:         startedAt,
		FinishedAt:        finishedAt,
		QuestionsSnapshot: questionsSnapshot,
		AnswersData:       answersData,
		TotalEarned:       roundToTwo(totalEarned),
		TotalMax:          roundToTwo(totalMax),
		ScorePercent:      scorePercent,
		Passed:            passed,
		NeedsReview:       needsReview,
	})
	if err != nil {
		return domain.Attempt{}, fmt.Errorf("usecase attempts submit create attempt: %w", err)
	}

	if u.audit != nil {
		u.audit.Log(ctx, domain.AppEventAttemptFinished, map[string]any{
			"attempt_id":    attempt.ID,
			"quiz_id":       attempt.QuizID,
			"user_id":       attempt.UserID,
			"score_percent": attempt.ScorePercent,
			"passed":        attempt.Passed,
			"needs_review":  attempt.NeedsReview,
		})

		if !attempt.NeedsReview {
			eventType := domain.AppEventAttemptFailed
			if attempt.Passed {
				eventType = domain.AppEventAttemptPassed
			}

			u.audit.Log(ctx, eventType, map[string]any{
				"attempt_id":    attempt.ID,
				"quiz_id":       attempt.QuizID,
				"user_id":       attempt.UserID,
				"score_percent": attempt.ScorePercent,
			})
		}
	}

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

func (u *AttemptUseCase) Review(ctx context.Context, params domain.ReviewAttemptParams) (domain.Attempt, error) {
	normalized, err := normalizeReviewAttemptParams(params)
	if err != nil {
		return domain.Attempt{}, fmt.Errorf("usecase attempts review: %w", err)
	}

	current, err := u.repository.GetAttemptByID(ctx, normalized.AttemptID)
	if err != nil {
		return domain.Attempt{}, fmt.Errorf("usecase attempts review load attempt: %w", err)
	}

	if !current.NeedsReview {
		return domain.Attempt{}, fmt.Errorf("attempt does not require review: %w", domain.ErrConflict)
	}

	normalized.TotalEarned = current.TotalEarned
	normalized.ScorePercent = current.ScorePercent
	normalized.ReviewScores = json.RawMessage("[]")

	if len(normalized.Scores) > 0 {
		quiz, err := u.repository.GetQuizForAttempt(ctx, current.QuizID)
		if err != nil {
			return domain.Attempt{}, fmt.Errorf("usecase attempts review load quiz: %w", err)
		}

		manualTotalEarned, manualScorePercent, reviewScores, err := u.evaluateManualReview(current, normalized.Scores)
		if err != nil {
			return domain.Attempt{}, fmt.Errorf("usecase attempts review manual scoring: %w", err)
		}

		normalized.TotalEarned = roundToTwo(manualTotalEarned)
		normalized.ScorePercent = manualScorePercent
		normalized.Passed = manualScorePercent >= float64(quiz.PassingScore)

		reviewScoresJSON, err := json.Marshal(reviewScores)
		if err != nil {
			return domain.Attempt{}, fmt.Errorf("usecase attempts review marshal review scores: %w", err)
		}
		normalized.ReviewScores = reviewScoresJSON
	}

	attempt, err := u.repository.UpdateReview(ctx, normalized)
	if err != nil {
		return domain.Attempt{}, fmt.Errorf("usecase attempts review update: %w", err)
	}

	if u.audit != nil {
		eventType := domain.AppEventAttemptFailed
		if attempt.Passed {
			eventType = domain.AppEventAttemptPassed
		}

		u.audit.Log(ctx, eventType, map[string]any{
			"attempt_id":     attempt.ID,
			"quiz_id":        attempt.QuizID,
			"user_id":        attempt.UserID,
			"reviewer_id":    attempt.ReviewerID,
			"reviewed_at":    attempt.ReviewedAt,
			"manual_passed":  attempt.ManualPassed,
			"review_comment": attempt.ReviewComment,
		})
	}

	return attempt, nil
}

func normalizeSubmitAttemptParams(params domain.SubmitAttemptParams) (domain.SubmitAttemptParams, error) {
	params.QuizID = strings.TrimSpace(params.QuizID)
	params.UserID = strings.TrimSpace(params.UserID)

	if params.QuizID == "" {
		return domain.SubmitAttemptParams{}, fmt.Errorf("quiz_id is required: %w", domain.ErrValidation)
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
		answer.OrderedOptionIDs = normalizeStringSlice(answer.OrderedOptionIDs)

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
	if filter.QuizID != nil {
		filter.QuizID = normalizeOptionalString(filter.QuizID)
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

func normalizeReviewAttemptParams(params domain.ReviewAttemptParams) (domain.ReviewAttemptParams, error) {
	params.AttemptID = strings.TrimSpace(params.AttemptID)
	params.ReviewerID = strings.TrimSpace(params.ReviewerID)
	params.Comment = normalizeOptionalString(params.Comment)
	params.Scores = normalizeReviewAttemptScores(params.Scores)

	if params.AttemptID == "" {
		return domain.ReviewAttemptParams{}, fmt.Errorf("attempt_id is required: %w", domain.ErrValidation)
	}

	if params.ReviewerID == "" {
		return domain.ReviewAttemptParams{}, fmt.Errorf("reviewer_id is required: %w", domain.ErrValidation)
	}

	seen := make(map[string]struct{}, len(params.Scores))
	for _, score := range params.Scores {
		if score.QuestionID == "" {
			return domain.ReviewAttemptParams{}, fmt.Errorf("score.question_id is required: %w", domain.ErrValidation)
		}

		if _, exists := seen[score.QuestionID]; exists {
			return domain.ReviewAttemptParams{}, fmt.Errorf("duplicate score for question: %w", domain.ErrValidation)
		}
		seen[score.QuestionID] = struct{}{}
	}

	return params, nil
}

func normalizeReviewAttemptScores(scores []domain.AttemptReviewScore) []domain.AttemptReviewScore {
	if len(scores) == 0 {
		return nil
	}

	normalized := make([]domain.AttemptReviewScore, 0, len(scores))
	for _, score := range scores {
		score.QuestionID = strings.TrimSpace(score.QuestionID)
		score.Comment = normalizeOptionalString(score.Comment)
		normalized = append(normalized, score)
	}

	return normalized
}

func (u *AttemptUseCase) evaluateManualReview(current domain.Attempt, reviewScores []domain.AttemptReviewScore) (float64, float64, []domain.AttemptReviewScore, error) {
	var questions []domain.Question
	if err := json.Unmarshal(current.QuestionsSnapshot, &questions); err != nil {
		return 0, 0, nil, fmt.Errorf("unmarshal attempt questions snapshot: %w", err)
	}

	var answers []domain.AttemptAnswer
	if len(current.AnswersData) > 0 {
		if err := json.Unmarshal(current.AnswersData, &answers); err != nil {
			return 0, 0, nil, fmt.Errorf("unmarshal attempt answers data: %w", err)
		}
	}

	answerMap := make(map[string]domain.AttemptAnswer, len(answers))
	for _, answer := range answers {
		answerMap[answer.QuestionID] = answer
	}

	scoreMap := make(map[string]domain.AttemptReviewScore, len(reviewScores))
	for _, score := range reviewScores {
		scoreMap[score.QuestionID] = score
	}

	manualQuestionsCount := 0
	totalEarned := 0.0
	totalMax := 0.0
	normalizedScores := make([]domain.AttemptReviewScore, 0, len(reviewScores))

	for _, question := range questions {
		totalMax += question.Points

		if isManualReviewQuestionType(question.Type) {
			manualQuestionsCount++

			score, ok := scoreMap[question.ID]
			if !ok {
				return 0, 0, nil, fmt.Errorf("missing review score for question %s: %w", question.ID, domain.ErrValidation)
			}

			if !isFiniteReviewPoints(score.Points) || score.Points < 0 || score.Points > question.Points {
				return 0, 0, nil, fmt.Errorf("invalid review points for question %s: %w", question.ID, domain.ErrValidation)
			}

			normalizedScores = append(normalizedScores, domain.AttemptReviewScore{
				QuestionID: question.ID,
				Points:     roundToTwo(score.Points),
				Comment:    score.Comment,
			})
			totalEarned += score.Points
			delete(scoreMap, question.ID)
			continue
		}

		answer := answerMap[question.ID]
		earned, requiresReview := evaluateQuestion(question, answer)
		if requiresReview {
			return 0, 0, nil, fmt.Errorf("question %s unexpectedly requires manual review: %w", question.ID, domain.ErrValidation)
		}

		totalEarned += earned
	}

	if len(scoreMap) > 0 {
		return 0, 0, nil, fmt.Errorf("review scores contain unknown or non-reviewable questions: %w", domain.ErrValidation)
	}

	if manualQuestionsCount == 0 {
		return 0, 0, nil, fmt.Errorf("manual scores provided but no manual questions found: %w", domain.ErrValidation)
	}

	scorePercent := 0.0
	if totalMax > 0 {
		scorePercent = roundToTwo(totalEarned / totalMax * 100)
	}

	return totalEarned, scorePercent, normalizedScores, nil
}

func isManualReviewQuestionType(questionType domain.QuestionType) bool {
	switch questionType {
	case domain.QuestionTypeLongText,
		domain.QuestionTypeMatching,
		domain.QuestionTypeOrdering,
		domain.QuestionTypeAudio,
		domain.QuestionTypeVideo,
		domain.QuestionTypeCode:
		return true
	default:
		return false
	}
}

func isFiniteReviewPoints(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func evaluateAttempt(quiz domain.Quiz, answers []domain.AttemptAnswer) (float64, float64, bool) {
	answerMap := make(map[string]domain.AttemptAnswer, len(answers))
	for _, answer := range answers {
		answerMap[answer.QuestionID] = answer
	}

	totalEarned := 0.0
	totalMax := 0.0
	needsReview := false

	for _, question := range quiz.Questions {
		totalMax += question.Points

		answer, exists := answerMap[question.ID]
		if !exists {
			continue
		}

		earned, requiresReview := evaluateQuestion(question, answer)
		if requiresReview {
			needsReview = true
		}

		totalEarned += earned
	}

	return totalEarned, totalMax, needsReview
}

func evaluateQuestion(question domain.Question, answer domain.AttemptAnswer) (float64, bool) {
	switch question.Type {
	case domain.QuestionTypeSingleChoice, domain.QuestionTypeImageChoice:
		correctIDs, ok := extractCorrectOptionIDs(question.Config)
		if !ok || len(correctIDs) != 1 || len(answer.SelectedOptionIDs) != 1 {
			return 0, false
		}

		if correctIDs[0] == answer.SelectedOptionIDs[0] {
			return question.Points, false
		}
		return 0, false
	case domain.QuestionTypeMultipleChoice:
		correctIDs, ok := extractCorrectOptionIDs(question.Config)
		if !ok {
			return 0, false
		}

		selected := normalizeStringSlice(answer.SelectedOptionIDs)
		if sameStringSet(correctIDs, selected) {
			return question.Points, false
		}
		return 0, false
	case domain.QuestionTypeTrueFalse:
		correctValue, ok := extractCorrectBoolean(question.Config)
		if !ok || answer.BooleanAnswer == nil {
			return 0, false
		}

		if correctValue == *answer.BooleanAnswer {
			return question.Points, false
		}
		return 0, false
	case domain.QuestionTypeShortAnswer, domain.QuestionTypeFillBlank:
		acceptedAnswers, ok := extractAcceptedAnswers(question.Config)
		if !ok || answer.TextAnswer == nil {
			return 0, false
		}

		candidate := strings.ToLower(strings.TrimSpace(*answer.TextAnswer))
		for _, accepted := range acceptedAnswers {
			if candidate == accepted {
				return question.Points, false
			}
		}
		return 0, false
	default:
		return 0, true
	}
}

func extractCorrectOptionIDs(config json.RawMessage) ([]string, bool) {
	var payload struct {
		Options []struct {
			ID        string `json:"id"`
			IsCorrect bool   `json:"is_correct"`
		} `json:"options"`
	}

	if err := json.Unmarshal(config, &payload); err != nil {
		return nil, false
	}

	ids := make([]string, 0)
	for _, option := range payload.Options {
		if option.IsCorrect {
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
		AcceptedAnswers []string `json:"accepted_answers"`
	}

	if err := json.Unmarshal(config, &payload); err != nil {
		return nil, false
	}

	normalized := make([]string, 0, len(payload.AcceptedAnswers))
	for _, answer := range payload.AcceptedAnswers {
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
