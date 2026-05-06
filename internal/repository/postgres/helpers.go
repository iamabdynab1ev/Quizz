package postgres

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"time"

	"lms-arvand-backend/internal/domain"

	"github.com/jackc/pgx/v5/pgconn"
)

func nullableStringForWrite(value string) any {
	trimmed := value
	if trimmed == "" {
		return nil
	}

	return trimmed
}

func nullableStringPointerForWrite(value *string) any {
	if value == nil {
		return nil
	}

	if *value == "" {
		return nil
	}

	return *value
}

func stringPointerForWrite(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

func nullableIntPointerForWrite(value *int) any {
	if value == nil {
		return nil
	}

	return *value
}

func nullableTimePointerForWrite(value *time.Time) any {
	if value == nil {
		return nil
	}

	return *value
}

func boolPointerForWrite(value *bool) any {
	if value == nil {
		return false
	}

	return *value
}

func nullableBoolPointerForWrite(value *bool) any {
	if value == nil {
		return nil
	}

	return *value
}

func optionalString(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}

	return &value.String
}

func optionalTime(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}

	return &value.Time
}

func optionalInt(value sql.NullInt32) *int {
	if !value.Valid {
		return nil
	}

	number := int(value.Int32)
	return &number
}

func optionalBool(value sql.NullBool) *bool {
	if !value.Valid {
		return nil
	}

	flag := value.Bool
	return &flag
}

func dateString(value sql.NullTime) string {
	if !value.Valid {
		return ""
	}

	return value.Time.Format("2006-01-02")
}

func optionalDateString(value sql.NullTime) *string {
	if !value.Valid {
		return nil
	}

	date := value.Time.Format("2006-01-02")
	return &date
}

func toJSONValue(value driver.Valuer) (any, error) {
	rawValue, err := value.Value()
	if err != nil {
		return nil, fmt.Errorf("repository postgres to json value: %w", err)
	}

	return rawValue, nil
}

func multiLangValueOrNil(value domain.MultiLangText) (any, error) {
	if value.IsZero() {
		return nil, nil
	}

	rawValue, err := toJSONValue(value)
	if err != nil {
		return nil, fmt.Errorf("repository postgres multilang value or nil: %w", err)
	}

	return rawValue, nil
}

func platformsToStrings(platforms []domain.Platform) []string {
	if len(platforms) == 0 {
		return []string{}
	}

	values := make([]string, 0, len(platforms))
	for _, platform := range platforms {
		values = append(values, string(platform))
	}

	return values
}

func stringsToPlatforms(values []string) []domain.Platform {
	if len(values) == 0 {
		return nil
	}

	platforms := make([]domain.Platform, 0, len(values))
	for _, value := range values {
		platforms = append(platforms, domain.Platform(value))
	}

	return platforms
}

func appEventTypesToStrings(values []domain.AppEventType) []string {
	if len(values) == 0 {
		return []string{}
	}

	normalized := make([]string, 0, len(values))
	for _, value := range values {
		normalized = append(normalized, string(value))
	}

	return normalized
}

func stringsToAppEventTypes(values []string) []domain.AppEventType {
	if len(values) == 0 {
		return nil
	}

	normalized := make([]domain.AppEventType, 0, len(values))
	for _, value := range values {
		normalized = append(normalized, domain.AppEventType(value))
	}

	return normalized
}

func wrapPGError(action string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return fmt.Errorf("%s: %w: %s", action, domain.ErrConflict, pgErr.Message)
		case "22P02", "22007", "22008":
			return fmt.Errorf("%s: %w: %s", action, domain.ErrValidation, pgErr.Message)
		}
	}

	return fmt.Errorf("%s: %w", action, err)
}
