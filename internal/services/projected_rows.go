package services

import "time"

func formatProjectedDateTimes(rows []map[string]any, fields []string, layout string) {
	if len(rows) == 0 || len(fields) == 0 {
		return
	}

	fieldSet := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		fieldSet[field] = struct{}{}
	}

	for _, field := range []string{"created_at", "updated_at"} {
		if _, ok := fieldSet[field]; !ok {
			continue
		}
		for _, row := range rows {
			switch ts := row[field].(type) {
			case time.Time:
				row[field] = ts.Format(layout)
			case *time.Time:
				if ts != nil {
					row[field] = ts.Format(layout)
				}
			}
		}
	}
}
