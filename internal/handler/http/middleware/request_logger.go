package middleware

import (
	"log/slog"
	nethttp "net/http"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

type responseWriter struct {
	nethttp.ResponseWriter
	statusCode int
	bytes      int
}

func RequestLogger(logger *slog.Logger) func(next nethttp.Handler) nethttp.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next nethttp.Handler) nethttp.Handler {
		return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
			startedAt := time.Now()
			writer := &responseWriter{
				ResponseWriter: w,
				statusCode:     nethttp.StatusOK,
			}

			next.ServeHTTP(writer, r)

			logger.InfoContext(r.Context(), "http request completed",
				slog.String("request_id", chimiddleware.GetReqID(r.Context())),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("query", r.URL.RawQuery),
				slog.String("remote_addr", r.RemoteAddr),
				slog.Int("status_code", writer.statusCode),
				slog.Int("bytes", writer.bytes),
				slog.Duration("duration", time.Since(startedAt)),
			)
		})
	}
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseWriter) Write(body []byte) (int, error) {
	size, err := w.ResponseWriter.Write(body)
	w.bytes += size
	return size, err
}
