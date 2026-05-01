package middleware

import (
	"net/http"
	"strings"
)

func CORS(allowedOrigins []string) func(next http.Handler) http.Handler {
	normalized := normalizeAllowedOrigins(allowedOrigins)
	allowAny := len(normalized) == 0 || containsWildcard(normalized)
	allowed := make(map[string]struct{}, len(normalized))
	for _, origin := range normalized {
		allowed[origin] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := strings.TrimSpace(r.Header.Get("Origin"))
			if origin != "" {
				switch {
				case allowAny:
					w.Header().Set("Access-Control-Allow-Origin", "*")
				case isOriginAllowed(origin, allowed):
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Add("Vary", "Origin")
				}
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Expose-Headers", "Content-Type")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func normalizeAllowedOrigins(origins []string) []string {
	if len(origins) == 0 {
		return []string{"*"}
	}

	result := make([]string, 0, len(origins))
	for _, origin := range origins {
		trimmed := strings.TrimSpace(origin)
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
	}

	if len(result) == 0 {
		return []string{"*"}
	}

	return result
}

func containsWildcard(origins []string) bool {
	for _, origin := range origins {
		if origin == "*" {
			return true
		}
	}

	return false
}

func isOriginAllowed(origin string, allowed map[string]struct{}) bool {
	_, ok := allowed[origin]
	return ok
}
