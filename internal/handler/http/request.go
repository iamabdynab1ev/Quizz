package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	nethttp "net/http"
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
			return fmt.Errorf("handler http decode json: empty body")
		case errors.As(err, &syntaxError):
			return fmt.Errorf("handler http decode json syntax at offset %d: %w", syntaxError.Offset, err)
		case errors.As(err, &unmarshalTypeError):
			return fmt.Errorf("handler http decode json wrong type for field %s: %w", unmarshalTypeError.Field, err)
		case errors.As(err, &maxBytesError):
			return fmt.Errorf("handler http decode json max bytes: %w", err)
		default:
			return fmt.Errorf("handler http decode json: %w", err)
		}
	}

	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("handler http decode json: multiple JSON values are not allowed")
	}

	return nil
}
