package usecase

import (
	"context"
	"testing"

	"lms-arvand-backend/internal/domain"
)

type courseRepoStub struct {
	createCalled bool
	createParams domain.CreateCourseParams
	updateCalled bool
	updateParams domain.UpdateCourseParams
}

func (r *courseRepoStub) Create(ctx context.Context, params domain.CreateCourseParams) (domain.Course, error) {
	r.createCalled = true
	r.createParams = params
	return domain.Course{ID: "course-created", Title: params.Title}, nil
}

func (r *courseRepoStub) GetByID(ctx context.Context, courseID string) (domain.Course, error) {
	return domain.Course{}, nil
}

func (r *courseRepoStub) List(ctx context.Context, filter domain.CourseListFilter) ([]domain.Course, int, error) {
	return nil, 0, nil
}

func (r *courseRepoStub) Update(ctx context.Context, params domain.UpdateCourseParams) (domain.Course, error) {
	r.updateCalled = true
	r.updateParams = params
	return domain.Course{ID: params.ID, Title: params.Title}, nil
}

func (r *courseRepoStub) Archive(ctx context.Context, courseID string) error {
	return nil
}

func TestCourseUseCaseCreateNormalizesQuizFieldsAndQuestions(t *testing.T) {
	repo := &courseRepoStub{}
	uc := NewCourseUseCase(repo)

	_, err := uc.Create(context.Background(), domain.CreateCourseParams{
		Title: domain.MultiLangText{RU: "Курс", TJ: "Курс"},
		Questions: []domain.QuestionPayload{
			{
				Type:   domain.QuestionTypeSingleChoice,
				Prompt: domain.MultiLangText{RU: "Вопрос", TJ: "Савол"},
				Config: mustJSON(t, map[string]any{
					"options": []map[string]any{
						{"id": "a", "is_correct": true},
						{"id": "b", "is_correct": false},
					},
				}),
			},
		},
	})
	if err != nil {
		t.Fatalf("create returned error: %v", err)
	}
	if !repo.createCalled {
		t.Fatalf("expected repository create to be called")
	}
	if repo.createParams.Status != domain.CourseStatusDraft {
		t.Fatalf("expected draft status, got %q", repo.createParams.Status)
	}
	if repo.createParams.QuizPassPercent != 80 {
		t.Fatalf("expected default quiz pass percent 80, got %d", repo.createParams.QuizPassPercent)
	}
	if repo.createParams.MaxAttempts != 3 {
		t.Fatalf("expected default max attempts 3, got %d", repo.createParams.MaxAttempts)
	}
	if repo.createParams.RetakeCooldownDays != 0 {
		t.Fatalf("expected default retake cooldown 0, got %d", repo.createParams.RetakeCooldownDays)
	}
	if len(repo.createParams.Questions) != 1 {
		t.Fatalf("expected 1 normalized question, got %d", len(repo.createParams.Questions))
	}
	if repo.createParams.Questions[0].Position != 1 {
		t.Fatalf("expected default question position 1, got %d", repo.createParams.Questions[0].Position)
	}
	if repo.createParams.Questions[0].Points != 1 {
		t.Fatalf("expected default question points 1, got %v", repo.createParams.Questions[0].Points)
	}
}

func TestCourseUseCaseCreateRejectsInvalidQuestionConfig(t *testing.T) {
	repo := &courseRepoStub{}
	uc := NewCourseUseCase(repo)

	_, err := uc.Create(context.Background(), domain.CreateCourseParams{
		Title: domain.MultiLangText{RU: "Курс", TJ: "Курс"},
		Questions: []domain.QuestionPayload{
			{
				Type:   domain.QuestionTypeSingleChoice,
				Prompt: domain.MultiLangText{RU: "Вопрос", TJ: "Савол"},
				Config: mustJSON(t, map[string]any{
					"options": []map[string]any{
						{"id": "a", "is_correct": true},
						{"id": "b", "is_correct": true},
					},
				}),
			},
		},
	})
	if err == nil {
		t.Fatalf("expected invalid question config error")
	}
	if repo.createCalled {
		t.Fatalf("repository create must not be called for invalid questions")
	}
}

func TestCourseUseCaseUpdateKeepsCertificatesEnabledByDefault(t *testing.T) {
	repo := &courseRepoStub{}
	uc := NewCourseUseCase(repo)

	_, err := uc.Update(context.Background(), domain.UpdateCourseParams{
		ID:     "course-1",
		Title:  domain.MultiLangText{RU: "Курс", TJ: "Курс"},
		Status: domain.CourseStatusDraft,
	})
	if err != nil {
		t.Fatalf("update returned error: %v", err)
	}
	if !repo.updateCalled {
		t.Fatalf("expected repository update to be called")
	}
	if !repo.updateParams.CertificateEnabled {
		t.Fatalf("expected certificate_enabled to default to true on update")
	}
}

var _ courseRepository = (*courseRepoStub)(nil)
