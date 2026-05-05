package domain

import "errors"

var (
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict")
	ErrValidation   = errors.New("validation")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrUnavailable  = errors.New("unavailable")
	ErrTooMany      = errors.New("too many requests")
)

type AppError struct {
	Code    string       `json:"error"`
	Message string       `json:"message"`
	Fields  []FieldError `json:"fields,omitempty"`
	Status  int          `json:"-"`
	Err     error        `json:"-"`
}

type FieldError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *AppError) Error() string {
	if e == nil {
		return ""
	}

	if e.Message != "" {
		return e.Message
	}

	if e.Err != nil {
		return e.Err.Error()
	}

	return e.Code
}

func (e *AppError) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.Err
}

func ValidationError(message string) *AppError {
	return &AppError{
		Code:    "validation_error",
		Message: message,
		Status:  400,
		Err:     ErrValidation,
	}
}

func ValidationField(field, code, message string) FieldError {
	return FieldError{
		Field:   field,
		Code:    code,
		Message: message,
	}
}

func FieldValidationError(message string, fields ...FieldError) *AppError {
	if message == "" {
		message = "Проверьте данные запроса"
	}

	return &AppError{
		Code:    "validation_error",
		Message: message,
		Fields:  fields,
		Status:  400,
		Err:     ErrValidation,
	}
}

func BadRequestError(code, message string, fields ...FieldError) *AppError {
	if code == "" {
		code = "bad_request"
	}
	if message == "" {
		message = "Некорректный запрос"
	}

	return &AppError{
		Code:    code,
		Message: message,
		Fields:  fields,
		Status:  400,
		Err:     ErrValidation,
	}
}

func UnauthorizedError(message string) *AppError {
	return &AppError{
		Code:    "unauthorized",
		Message: message,
		Status:  401,
		Err:     ErrUnauthorized,
	}
}

func ForbiddenError(message string) *AppError {
	return &AppError{
		Code:    "forbidden",
		Message: message,
		Status:  403,
		Err:     ErrForbidden,
	}
}

func NotFoundError(message string) *AppError {
	return &AppError{
		Code:    "not_found",
		Message: message,
		Status:  404,
		Err:     ErrNotFound,
	}
}

func ConflictError(message string) *AppError {
	return &AppError{
		Code:    "conflict",
		Message: message,
		Status:  409,
		Err:     ErrConflict,
	}
}

func UnavailableError(message string) *AppError {
	return &AppError{
		Code:    "service_unavailable",
		Message: message,
		Status:  503,
		Err:     ErrUnavailable,
	}
}

func TooManyRequestsError(code string, message string) *AppError {
	if code == "" {
		code = "too_many_requests"
	}

	return &AppError{
		Code:    code,
		Message: message,
		Status:  429,
		Err:     ErrTooMany,
	}
}
