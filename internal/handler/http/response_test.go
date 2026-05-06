package http

import (
	"encoding/json"
	nethttp "net/http"
	"net/http/httptest"
	"testing"

	"lms-arvand-backend/internal/domain"
)

func TestWriteAppErrorSingleFieldUsesFlatResponse(t *testing.T) {
	recorder := httptest.NewRecorder()

	writeAppError(recorder, domain.FieldValidationError("Проверьте поля формы",
		domain.ValidationField("passing_points", "too_high", "Баллы для прохождения не могут быть больше максимального балла теста"),
	))

	if recorder.Code != nethttp.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, nethttp.StatusBadRequest)
	}

	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if body["field"] != "passing_points" {
		t.Fatalf("field = %v, want passing_points", body["field"])
	}
	if body["code"] != "too_high" {
		t.Fatalf("code = %v, want too_high", body["code"])
	}
	if body["message"] == "" {
		t.Fatalf("message must not be empty")
	}
	if _, exists := body["fields"]; exists {
		t.Fatalf("single-field validation response must not contain fields envelope: %s", recorder.Body.String())
	}
	if _, exists := body["error"]; exists {
		t.Fatalf("single-field validation response must not contain error envelope: %s", recorder.Body.String())
	}
}
