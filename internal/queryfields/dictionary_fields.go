package queryfields

import (
	"fmt"
	"strings"
)

var (
	userListFields = map[string]struct{}{
		"id":                   {},
		"fio":                  {},
		"email":                {},
		"phone_number":         {},
		"username":             {},
		"position_id":          {},
		"status_id":            {},
		"status_code":          {},
		"branch_id":            {},
		"department_id":        {},
		"office_id":            {},
		"otdel_id":             {},
		"branch_name":          {},
		"department_name":      {},
		"position_name":        {},
		"otdel_name":           {},
		"office_name":          {},
		"photo_url":            {},
		"must_change_password": {},
		"is_head":              {},
		"created_at":           {},
		"updated_at":           {},
	}
	positionListFields = map[string]struct{}{
		"id":            {},
		"name":          {},
		"department_id": {},
		"otdel_id":      {},
		"branch_id":     {},
		"office_id":     {},
		"type":          {},
		"status_id":     {},
		"created_at":    {},
		"updated_at":    {},
	}
	equipmentListFields = map[string]struct{}{
		"id":                {},
		"name":              {},
		"address":           {},
		"branch_id":         {},
		"office_id":         {},
		"equipment_type_id": {},
		"status_id":         {},
		"created_at":        {},
		"updated_at":        {},
	}
)

func NormalizeUserListFields(fields []string) ([]string, error) {
	return normalizeFields(fields, userListFields)
}

func NormalizePositionListFields(fields []string) ([]string, error) {
	return normalizeFields(fields, positionListFields)
}

func NormalizeEquipmentListFields(fields []string) ([]string, error) {
	return normalizeFields(fields, equipmentListFields)
}

func normalizeFields(fields []string, allowed map[string]struct{}) ([]string, error) {
	if len(fields) == 0 {
		return nil, nil
	}

	normalized := make([]string, 0, len(fields)+1)
	seen := make(map[string]struct{}, len(fields)+1)

	if _, ok := allowed["id"]; ok {
		normalized = append(normalized, "id")
		seen["id"] = struct{}{}
	}

	for _, field := range fields {
		field = strings.TrimSpace(strings.ToLower(field))
		if field == "" {
			continue
		}
		if _, ok := allowed[field]; !ok {
			return nil, fmt.Errorf("поле '%s' не поддерживается в fields", field)
		}
		if _, ok := seen[field]; ok {
			continue
		}
		normalized = append(normalized, field)
		seen[field] = struct{}{}
	}

	return normalized, nil
}
