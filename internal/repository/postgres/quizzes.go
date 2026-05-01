package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"lms-arvand-backend/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type quizRowScanner interface {
	Scan(dest ...any) error
}

type QuizRepository struct {
	pool *pgxpool.Pool
}

func NewQuizRepository(pool *pgxpool.Pool) *QuizRepository {
	return &QuizRepository{pool: pool}
}

func (r *QuizRepository) Create(ctx context.Context, params domain.CreateQuizParams) (domain.Quiz, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return domain.Quiz{}, fmt.Errorf("repository postgres quizzes create begin tx: %w", err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	titleValue, err := toJSONValue(params.Title)
	if err != nil {
		return domain.Quiz{}, fmt.Errorf("repository postgres quizzes create title value: %w", err)
	}

	descriptionValue, err := multiLangValueOrNil(params.Description)
	if err != nil {
		return domain.Quiz{}, fmt.Errorf("repository postgres quizzes create description value: %w", err)
	}

	var quizID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO quizzes (
			title,
			description,
			category,
			status,
			platforms,
			time_limit_minutes,
			passing_score,
			max_attempts,
			shuffle_questions,
			show_results,
			allow_retry
		) VALUES (
			$1::jsonb,
			$2::jsonb,
			$3,
			$4,
			$5::platform[],
			$6,
			$7,
			$8,
			$9,
			$10,
			$11
		)
		RETURNING id
	`,
		titleValue,
		descriptionValue,
		nullableStringPointerForWrite(params.Category),
		string(params.Status),
		platformsToStrings(params.Platforms),
		nullableIntPointerForWrite(params.TimeLimitMinutes),
		params.PassingScore,
		params.MaxAttempts,
		params.ShuffleQuestions,
		params.ShowResults,
		params.AllowRetry,
	).Scan(&quizID); err != nil {
		return domain.Quiz{}, wrapPGError("repository postgres quizzes create insert", err)
	}

	if err := r.replaceQuestions(ctx, tx, quizID, params.Questions); err != nil {
		return domain.Quiz{}, fmt.Errorf("repository postgres quizzes create questions: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Quiz{}, fmt.Errorf("repository postgres quizzes create commit: %w", err)
	}

	quiz, err := r.GetByID(ctx, quizID)
	if err != nil {
		return domain.Quiz{}, fmt.Errorf("repository postgres quizzes create fetch by id: %w", err)
	}

	return quiz, nil
}

func (r *QuizRepository) GetByID(ctx context.Context, quizID string) (domain.Quiz, error) {
	quiz, err := scanQuizRow(r.pool.QueryRow(ctx, `
		SELECT
			id,
			title,
			description,
			category,
			status,
			platforms,
			time_limit_minutes,
			passing_score,
			max_attempts,
			shuffle_questions,
			show_results,
			allow_retry,
			created_at,
			updated_at
		FROM quizzes
		WHERE id = $1
	`, quizID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Quiz{}, fmt.Errorf("repository postgres quizzes get by id: %w", domain.ErrNotFound)
		}

		return domain.Quiz{}, fmt.Errorf("repository postgres quizzes get by id: %w", err)
	}

	questions, err := r.listQuestionsByQuizID(ctx, quizID)
	if err != nil {
		return domain.Quiz{}, fmt.Errorf("repository postgres quizzes get by id questions: %w", err)
	}
	quiz.Questions = questions

	return quiz, nil
}

func (r *QuizRepository) List(ctx context.Context, filter domain.QuizListFilter) ([]domain.Quiz, int, error) {
	buildQuery := func(includePagination bool) (string, []any) {
		query := strings.Builder{}
		if includePagination {
			query.WriteString(`
				SELECT
					id,
					title,
					description,
					category,
					status,
					platforms,
					time_limit_minutes,
					passing_score,
					max_attempts,
					shuffle_questions,
					show_results,
					allow_retry,
					created_at,
					updated_at
				FROM quizzes
				WHERE 1 = 1
			`)
		} else {
			query.WriteString(`
				SELECT COUNT(*)
				FROM quizzes
				WHERE 1 = 1
			`)
		}

		args := make([]any, 0, 6)
		position := 1

		if filter.Search != "" {
			query.WriteString(fmt.Sprintf(`
				AND (
					title->>'ru' ILIKE $%d OR
					title->>'tj' ILIKE $%d OR
					COALESCE(category, '') ILIKE $%d
				)
			`, position, position, position))
			args = append(args, "%"+filter.Search+"%")
			position++
		}

		if filter.Status != nil {
			query.WriteString(fmt.Sprintf(" AND status = $%d", position))
			args = append(args, string(*filter.Status))
			position++
		}

		if filter.Category != nil {
			query.WriteString(fmt.Sprintf(" AND category = $%d", position))
			args = append(args, *filter.Category)
			position++
		}

		if filter.Platform != nil {
			query.WriteString(fmt.Sprintf(" AND $%d = ANY(platforms)", position))
			args = append(args, string(*filter.Platform))
			position++
		}

		if includePagination {
			query.WriteString(fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", position, position+1))
			args = append(args, filter.Limit, filter.Offset)
		}

		return query.String(), args
	}

	countQuery, countArgs := buildQuery(false)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("repository postgres quizzes list count: %w", err)
	}

	query, args := buildQuery(true)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("repository postgres quizzes list query: %w", err)
	}
	defer rows.Close()

	quizzes := make([]domain.Quiz, 0, filter.Limit)
	for rows.Next() {
		quiz, err := scanQuizRow(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("repository postgres quizzes list scan: %w", err)
		}

		quizzes = append(quizzes, quiz)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("repository postgres quizzes list rows: %w", err)
	}

	return quizzes, total, nil
}

func (r *QuizRepository) Update(ctx context.Context, params domain.UpdateQuizParams) (domain.Quiz, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return domain.Quiz{}, fmt.Errorf("repository postgres quizzes update begin tx: %w", err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	titleValue, err := toJSONValue(params.Title)
	if err != nil {
		return domain.Quiz{}, fmt.Errorf("repository postgres quizzes update title value: %w", err)
	}

	descriptionValue, err := multiLangValueOrNil(params.Description)
	if err != nil {
		return domain.Quiz{}, fmt.Errorf("repository postgres quizzes update description value: %w", err)
	}

	var quizID string
	if err := tx.QueryRow(ctx, `
		UPDATE quizzes
		SET
			title = $2::jsonb,
			description = $3::jsonb,
			category = $4,
			status = $5,
			platforms = $6::platform[],
			time_limit_minutes = $7,
			passing_score = $8,
			max_attempts = $9,
			shuffle_questions = $10,
			show_results = $11,
			allow_retry = $12,
			updated_at = NOW()
		WHERE id = $1
		RETURNING id
	`,
		params.ID,
		titleValue,
		descriptionValue,
		nullableStringPointerForWrite(params.Category),
		string(params.Status),
		platformsToStrings(params.Platforms),
		nullableIntPointerForWrite(params.TimeLimitMinutes),
		params.PassingScore,
		params.MaxAttempts,
		params.ShuffleQuestions,
		params.ShowResults,
		params.AllowRetry,
	).Scan(&quizID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Quiz{}, fmt.Errorf("repository postgres quizzes update: %w", domain.ErrNotFound)
		}

		return domain.Quiz{}, wrapPGError("repository postgres quizzes update", err)
	}

	if err := r.replaceQuestions(ctx, tx, quizID, params.Questions); err != nil {
		return domain.Quiz{}, fmt.Errorf("repository postgres quizzes update questions: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Quiz{}, fmt.Errorf("repository postgres quizzes update commit: %w", err)
	}

	quiz, err := r.GetByID(ctx, quizID)
	if err != nil {
		return domain.Quiz{}, fmt.Errorf("repository postgres quizzes update fetch by id: %w", err)
	}

	return quiz, nil
}

func (r *QuizRepository) Archive(ctx context.Context, quizID string) error {
	var returnedID string
	if err := r.pool.QueryRow(ctx, `
		UPDATE quizzes
		SET
			status = 'archived',
			updated_at = NOW()
		WHERE id = $1
		RETURNING id
	`, quizID).Scan(&returnedID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("repository postgres quizzes archive: %w", domain.ErrNotFound)
		}

		return fmt.Errorf("repository postgres quizzes archive: %w", err)
	}

	return nil
}

func (r *QuizRepository) replaceQuestions(ctx context.Context, tx pgx.Tx, quizID string, questions []domain.QuestionPayload) error {
	if _, err := tx.Exec(ctx, `DELETE FROM questions WHERE quiz_id = $1`, quizID); err != nil {
		return fmt.Errorf("repository postgres quizzes replace questions delete: %w", err)
	}

	for _, question := range questions {
		promptValue, err := toJSONValue(question.Prompt)
		if err != nil {
			return fmt.Errorf("repository postgres quizzes question prompt value: %w", err)
		}

		explanationValue, err := multiLangValueOrNil(question.Explanation)
		if err != nil {
			return fmt.Errorf("repository postgres quizzes question explanation value: %w", err)
		}

		configValue := question.Config
		if len(configValue) == 0 {
			configValue = []byte("{}")
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO questions (
				quiz_id,
				position,
				type,
				prompt,
				explanation,
				points,
				required,
				config
			) VALUES (
				$1,
				$2,
				$3,
				$4::jsonb,
				$5::jsonb,
				$6,
				$7,
				$8::jsonb
			)
		`,
			quizID,
			question.Position,
			string(question.Type),
			promptValue,
			explanationValue,
			question.Points,
			question.Required,
			[]byte(configValue),
		); err != nil {
			return wrapPGError("repository postgres quizzes insert question", err)
		}
	}

	return nil
}

func (r *QuizRepository) listQuestionsByQuizID(ctx context.Context, quizID string) ([]domain.Question, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			id,
			position,
			type,
			prompt,
			explanation,
			points,
			required,
			config,
			created_at
		FROM questions
		WHERE quiz_id = $1
		ORDER BY position ASC
	`, quizID)
	if err != nil {
		return nil, fmt.Errorf("repository postgres quizzes list questions query: %w", err)
	}
	defer rows.Close()

	questions := make([]domain.Question, 0)
	for rows.Next() {
		question, err := scanQuestionRow(rows)
		if err != nil {
			return nil, fmt.Errorf("repository postgres quizzes list questions scan: %w", err)
		}

		questions = append(questions, question)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository postgres quizzes list questions rows: %w", err)
	}

	return questions, nil
}

func scanQuizRow(scanner quizRowScanner) (domain.Quiz, error) {
	var quiz domain.Quiz
	var titleBytes []byte
	var descriptionBytes []byte
	var category sql.NullString
	var status string
	var platforms []string
	var timeLimitMinutes sql.NullInt32

	if err := scanner.Scan(
		&quiz.ID,
		&titleBytes,
		&descriptionBytes,
		&category,
		&status,
		&platforms,
		&timeLimitMinutes,
		&quiz.PassingScore,
		&quiz.MaxAttempts,
		&quiz.ShuffleQuestions,
		&quiz.ShowResults,
		&quiz.AllowRetry,
		&quiz.CreatedAt,
		&quiz.UpdatedAt,
	); err != nil {
		return domain.Quiz{}, err
	}

	if err := quiz.Title.Scan(titleBytes); err != nil {
		return domain.Quiz{}, fmt.Errorf("repository postgres scan quiz title: %w", err)
	}

	if len(descriptionBytes) > 0 {
		if err := quiz.Description.Scan(descriptionBytes); err != nil {
			return domain.Quiz{}, fmt.Errorf("repository postgres scan quiz description: %w", err)
		}
	}

	quiz.Category = optionalString(category)
	quiz.Status = domain.QuizStatus(status)
	quiz.Platforms = stringsToPlatforms(platforms)
	quiz.TimeLimitMinutes = optionalInt(timeLimitMinutes)

	return quiz, nil
}

func scanQuestionRow(scanner quizRowScanner) (domain.Question, error) {
	var question domain.Question
	var promptBytes []byte
	var explanationBytes []byte
	var questionType string

	if err := scanner.Scan(
		&question.ID,
		&question.Position,
		&questionType,
		&promptBytes,
		&explanationBytes,
		&question.Points,
		&question.Required,
		&question.Config,
		&question.CreatedAt,
	); err != nil {
		return domain.Question{}, err
	}

	if err := question.Prompt.Scan(promptBytes); err != nil {
		return domain.Question{}, fmt.Errorf("repository postgres scan question prompt: %w", err)
	}

	if len(explanationBytes) > 0 {
		if err := question.Explanation.Scan(explanationBytes); err != nil {
			return domain.Question{}, fmt.Errorf("repository postgres scan question explanation: %w", err)
		}
	}

	question.Type = domain.QuestionType(questionType)

	return question, nil
}
