package http

import (
	"encoding/json"
	"fmt"
	nethttp "net/http"
)

type errorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

type PagedResponse[T any] struct {
	Data   []T `json:"data"`
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

func writeJSON(w nethttp.ResponseWriter, statusCode int, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("handler http write json marshal: %w", err)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)

	if _, err := w.Write(body); err != nil {
		return fmt.Errorf("handler http write json body: %w", err)
	}

	return nil
}

func writePagedJSON[T any](w nethttp.ResponseWriter, statusCode int, data []T, total, limit, offset int) error {
	return writeJSON(w, statusCode, PagedResponse[T]{
		Data:   data,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

func writeError(w nethttp.ResponseWriter, statusCode int, errorCode, message string) {
	if err := writeJSON(w, statusCode, errorResponse{
		Error:   errorCode,
		Message: message,
	}); err != nil {
		nethttp.Error(w, nethttp.StatusText(statusCode), statusCode)
	}
}
