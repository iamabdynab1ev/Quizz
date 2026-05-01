package telegram

import (
	"context"
	"fmt"
	"strings"

	"request-system/pkg/types"
)

type projectedUsersRepository interface {
	GetUsersProjected(ctx context.Context, filter types.Filter, fields []string) ([]map[string]any, uint64, error)
}

type executorCandidate struct {
	ID  uint64
	Fio string
}

func (c *TelegramController) listExecutorCandidates(ctx context.Context, filter types.Filter) ([]executorCandidate, error) {
	if projectedRepo, ok := c.userRepo.(projectedUsersRepository); ok {
		projected, _, err := projectedRepo.GetUsersProjected(ctx, filter, []string{"id", "fio"})
		if err != nil {
			return nil, err
		}

		candidates := make([]executorCandidate, 0, len(projected))
		for _, item := range projected {
			id, ok := anyToUint64(item["id"])
			if !ok || id == 0 {
				continue
			}

			fio, ok := item["fio"].(string)
			if !ok || strings.TrimSpace(fio) == "" {
				continue
			}

			candidates = append(candidates, executorCandidate{
				ID:  id,
				Fio: strings.TrimSpace(fio),
			})
		}

		return candidates, nil
	}

	users, _, err := c.userRepo.GetUsers(ctx, filter)
	if err != nil {
		return nil, err
	}

	candidates := make([]executorCandidate, 0, len(users))
	for _, user := range users {
		if strings.TrimSpace(user.Fio) == "" {
			continue
		}
		candidates = append(candidates, executorCandidate{
			ID:  user.ID,
			Fio: strings.TrimSpace(user.Fio),
		})
	}

	return candidates, nil
}

func anyToUint64(value any) (uint64, bool) {
	switch v := value.(type) {
	case uint64:
		return v, true
	case uint32:
		return uint64(v), true
	case uint16:
		return uint64(v), true
	case uint8:
		return uint64(v), true
	case uint:
		return uint64(v), true
	case int64:
		if v < 0 {
			return 0, false
		}
		return uint64(v), true
	case int32:
		if v < 0 {
			return 0, false
		}
		return uint64(v), true
	case int:
		if v < 0 {
			return 0, false
		}
		return uint64(v), true
	case float64:
		if v < 0 {
			return 0, false
		}
		return uint64(v), true
	case float32:
		if v < 0 {
			return 0, false
		}
		return uint64(v), true
	default:
		return 0, false
	}
}

func buildExecutorSearchFilter(scope executorSelectionScope, search string, limit int) types.Filter {
	if limit <= 0 {
		limit = 10
	}

	return types.Filter{
		Search:         strings.TrimSpace(search),
		Filter:         scope.filter,
		Sort:           map[string]string{"fio": "asc"},
		Limit:          limit,
		Page:           1,
		Offset:         0,
		WithPagination: false,
	}
}

func buildExecutorCallback(userID uint64) string {
	return fmt.Sprintf(`{"action":"set_executor","user_id":%d}`, userID)
}
