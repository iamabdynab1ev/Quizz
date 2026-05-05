package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	nethttp "net/http"
	"strings"

	"lms-arvand-backend/internal/domain"
)

func decodeJSON(w nethttp.ResponseWriter, r *nethttp.Request, dst any, maxBytes int64) error {
	r.Body = nethttp.MaxBytesReader(w, r.Body, maxBytes)
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var maxBytesError *nethttp.MaxBytesError

		switch {
		case errors.Is(err, io.EOF):
			return domain.BadRequestError("invalid_json", "Тело запроса не должно быть пустым",
				domain.ValidationField("body", "required", "Передайте JSON-объект в теле запроса"))
		case errors.As(err, &syntaxError):
			return domain.BadRequestError("invalid_json", "Некорректный JSON",
				domain.ValidationField("body", "invalid_json", fmt.Sprintf("Ошибка синтаксиса JSON около позиции %d", syntaxError.Offset)))
		case errors.As(err, &unmarshalTypeError):
			field := unmarshalTypeError.Field
			if field == "" {
				field = "body"
			}
			return domain.BadRequestError("invalid_json", "Некорректный тип поля",
				domain.ValidationField(field, "wrong_type", fmt.Sprintf("Поле должно иметь тип %s", unmarshalTypeError.Type.String())))
		case errors.As(err, &maxBytesError):
			return domain.BadRequestError("invalid_json", "Тело запроса слишком большое",
				domain.ValidationField("body", "too_large", "Уменьшите размер JSON-запроса"))
		default:
			if field := unknownJSONField(err); field != "" {
				return domain.BadRequestError("invalid_json", "В запросе есть неизвестное поле",
					domain.ValidationField(field, "unknown_field", "Это поле не поддерживается API"))
			}
			return domain.BadRequestError("invalid_json", "Некорректное тело запроса",
				domain.ValidationField("body", "invalid_json", err.Error()))
		}
	}

	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return domain.BadRequestError("invalid_json", "Некорректное тело запроса",
			domain.ValidationField("body", "multiple_json_values", "Передайте только один JSON-объект"))
	}

	return nil
}

func writeDecodeError(w nethttp.ResponseWriter, err error) {
	var appErr *domain.AppError
	if errors.As(err, &appErr) {
		writeAppError(w, appErr)
		return
	}

	writeError(w, nethttp.StatusBadRequest, "invalid_json", "Некорректное тело запроса")
}

func unknownJSONField(err error) string {
	const prefix = `json: unknown field "`
	message := err.Error()
	if !strings.HasPrefix(message, prefix) {
		return ""
	}

	field := strings.TrimPrefix(message, prefix)
	field = strings.TrimSuffix(field, `"`)
	return strings.TrimSpace(field)
}
