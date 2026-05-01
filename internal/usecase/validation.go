package usecase

import (
	"encoding/json"
	"fmt"

	"lms-arvand-backend/internal/domain"
)

func normalizeJSONRawMessage(payload json.RawMessage, allowEmpty bool) (json.RawMessage, error) {
	if len(payload) == 0 {
		if allowEmpty {
			return nil, nil
		}

		return json.RawMessage(`{}`), nil
	}

	if !json.Valid(payload) {
		return nil, fmt.Errorf("invalid json payload: %w", domain.ErrValidation)
	}

	return payload, nil
}

func normalizeAppEventTypes(values []domain.AppEventType) ([]domain.AppEventType, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf("events are required: %w", domain.ErrValidation)
	}

	normalized := make([]domain.AppEventType, 0, len(values))
	seen := make(map[domain.AppEventType]struct{}, len(values))

	for _, value := range values {
		if !value.IsValid() {
			return nil, fmt.Errorf("event is invalid: %w", domain.ErrValidation)
		}

		if _, exists := seen[value]; exists {
			continue
		}

		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}

	return normalized, nil
}
