package usecase

import (
	"fmt"
	"strings"

	"lms-arvand-backend/internal/domain"
)

type fieldValidationBuilder struct {
	fields []domain.FieldError
}

func (b *fieldValidationBuilder) add(field, code, message string) {
	b.fields = append(b.fields, domain.ValidationField(field, code, message))
}

func (b *fieldValidationBuilder) addRequired(field, value, label string) {
	if strings.TrimSpace(value) == "" {
		b.add(field, "required", fmt.Sprintf("%s обязательно", label))
	}
}

func (b *fieldValidationBuilder) addRequiredMultiLang(field string, value domain.MultiLangText, label string) {
	b.addRequired(field+".ru", value.RU, label+" на русском")
	b.addRequired(field+".tj", value.TJ, label+" на таджикском")
}

func (b *fieldValidationBuilder) addIntRange(field string, value, min, max int, label string) {
	if value < min || value > max {
		b.add(field, "out_of_range", fmt.Sprintf("%s должно быть от %d до %d", label, min, max))
	}
}

func (b *fieldValidationBuilder) addPositiveInt(field string, value int, label string) {
	if value <= 0 {
		b.add(field, "must_be_positive", label+" должно быть больше 0")
	}
}

func (b *fieldValidationBuilder) err() error {
	if len(b.fields) == 0 {
		return nil
	}

	return domain.FieldValidationError("Проверьте поля формы", b.fields...)
}
