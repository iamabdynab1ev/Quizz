package apperrors

import (
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestWrapDBError_StringTooLongIncludesParsedLimit(t *testing.T) {
	err := WrapDBError(&pgconn.PgError{
		Code:    sqlStateStringDataRightTruncation,
		Message: "value too long for type character varying(500)",
	})
	if err == nil {
		t.Fatal("expected mapped error")
	}

	httpErr, ok := err.(*HttpError)
	if !ok {
		t.Fatalf("expected *HttpError, got %T", err)
	}

	if !strings.Contains(httpErr.Message, "500") {
		t.Fatalf("expected message to include parsed limit, got %q", httpErr.Message)
	}
}

func TestExtractVarcharLimit(t *testing.T) {
	maxLen, ok := extractVarcharLimit("value too long for type varchar(255)")
	if !ok {
		t.Fatal("expected varchar limit to be parsed")
	}
	if maxLen != 255 {
		t.Fatalf("expected 255, got %d", maxLen)
	}
}
