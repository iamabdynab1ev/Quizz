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
		domain.ValidationField("quiz_pass_percent", "out_of_range", "Значение проходного процента должно быть от 0 до 100"),
	))

	if recorder.Code != nethttp.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, nethttp.StatusBadRequest)
	}

	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if body["field"] != "quiz_pass_percent" {
		t.Fatalf("field = %v, want quiz_pass_percent", body["field"])
	}
	if body["code"] != "out_of_range" {
		t.Fatalf("code = %v, want out_of_range", body["code"])
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
