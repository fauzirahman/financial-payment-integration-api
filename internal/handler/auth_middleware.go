package handler

import (
	"net/http"
	"strings"

	"github.com/fauzirahman/financial-payment-integration-api/internal/service"
)

func RequireAuth(tokens *service.TokenManager, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Fields(r.Header.Get("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			w.Header().Set("WWW-Authenticate", "Bearer")
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		if _, err := tokens.Validate(parts[1]); err != nil {
			w.Header().Set("WWW-Authenticate", "Bearer")
			writeError(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func AuthenticationMiddleware(tokens *service.TokenManager, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" || r.URL.Path == "/docs" || strings.HasPrefix(r.URL.Path, "/swagger/") ||
			r.URL.Path == "/api/v1/auth/register" || r.URL.Path == "/api/v1/auth/login" ||
			r.URL.Path == "/api/v1/webhooks/payment" {
			next.ServeHTTP(w, r)
			return
		}
		RequireAuth(tokens, next).ServeHTTP(w, r)
	})
}
