package domain

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
)

var (
	_ sql.Scanner   = (*MultiLangText)(nil)
	_ driver.Valuer = MultiLangText{}
)

type MultiLangText struct {
	RU string `json:"ru"`
	TJ string `json:"tj"`
}

func (m MultiLangText) Value() (driver.Value, error) {
	payload, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("domain multilangtext value marshal: %w", err)
	}

	return payload, nil
}

func (m *MultiLangText) Scan(src any) error {
	if m == nil {
		return fmt.Errorf("domain multilangtext scan: nil receiver")
	}

	switch value := src.(type) {
	case nil:
		*m = MultiLangText{}
		return nil
	case []byte:
		return m.unmarshal(value)
	case string:
		return m.unmarshal([]byte(value))
	case MultiLangText:
		*m = value
		return nil
	default:
		return fmt.Errorf("domain multilangtext scan: unsupported source type %T", src)
	}
}

func (m MultiLangText) IsZero() bool {
	return strings.TrimSpace(m.RU) == "" && strings.TrimSpace(m.TJ) == ""
}

func (m MultiLangText) ValidateRequired() error {
	if strings.TrimSpace(m.RU) == "" {
		return fmt.Errorf("domain multilangtext validate required: ru is empty")
	}

	if strings.TrimSpace(m.TJ) == "" {
		return fmt.Errorf("domain multilangtext validate required: tj is empty")
	}

	if m.IsZero() {
		return fmt.Errorf("domain multilangtext validate required: ru and tj are empty")
	}

	return nil
}

func (m *MultiLangText) unmarshal(data []byte) error {
	var candidate MultiLangText

	if err := json.Unmarshal(data, &candidate); err != nil {
		return fmt.Errorf("domain multilangtext unmarshal: %w", err)
	}

	*m = candidate
	return nil
}
