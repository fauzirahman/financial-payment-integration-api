package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fauzirahman/financial-payment-integration-api/internal/model"
	"github.com/fauzirahman/financial-payment-integration-api/internal/repository"
	"github.com/fauzirahman/financial-payment-integration-api/internal/service"
	"github.com/jackc/pgx/v5"
)

type inMemoryUserRepository struct {
	users map[string]*model.AuthUser
}

var _ repository.UserRepository = (*inMemoryUserRepository)(nil)

func newInMemoryUserRepository() *inMemoryUserRepository {
	return &inMemoryUserRepository{users: make(map[string]*model.AuthUser)}
}

func (r *inMemoryUserRepository) Create(_ context.Context, user *model.AuthUser) error {
	if _, exists := r.users[user.Email]; exists {
		return errors.New("duplicate email")
	}
	user.ID = "user-1"
	user.CreatedAt = time.Now().UTC()
	user.UpdatedAt = user.CreatedAt
	copyOfUser := *user
	r.users[user.Email] = &copyOfUser
	return nil
}

func (r *inMemoryUserRepository) FindByEmail(_ context.Context, email string) (*model.AuthUser, error) {
	user, exists := r.users[email]
	if !exists {
		return nil, pgx.ErrNoRows
	}
	copyOfUser := *user
	return &copyOfUser, nil
}

func TestAuthenticationEndpoints(t *testing.T) {
	const secret = "test-auth-token-secret-with-at-least-32-bytes"
	users := newInMemoryUserRepository()
	authHandler := NewAuthHandler(
		service.NewAuthService(users),
		service.NewTokenManager(secret, time.Hour),
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/register", authHandler.Register)
	mux.HandleFunc("/api/v1/auth/login", authHandler.Login)
	mux.HandleFunc("/api/v1/private", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	server := AuthenticationMiddleware(service.NewTokenManager(secret, time.Hour), mux)

	registerResponse := performAuthRequest(t, server, http.MethodPost, "/api/v1/auth/register",
		`{"name":"Demo User","email":"demo@example.com","password":"password-123"}`, "")
	if registerResponse.Code != http.StatusCreated {
		t.Fatalf("register status = %d, want %d; body: %s", registerResponse.Code, http.StatusCreated, registerResponse.Body)
	}

	var registered authResponse
	if err := json.Unmarshal(registerResponse.Body.Bytes(), &registered); err != nil {
		t.Fatalf("decode register response: %v", err)
	}
	if registered.AccessToken == "" || registered.TokenType != "Bearer" || registered.ExpiresIn != 3600 {
		t.Fatalf("unexpected register token response: %+v", registered)
	}

	protectedResponse := performAuthRequest(t, server, http.MethodGet, "/api/v1/private", "", "Bearer "+registered.AccessToken)
	if protectedResponse.Code != http.StatusNoContent {
		t.Fatalf("protected endpoint status with register token = %d, want %d; body: %s",
			protectedResponse.Code, http.StatusNoContent, protectedResponse.Body)
	}

	loginResponse := performAuthRequest(t, server, http.MethodPost, "/api/v1/auth/login",
		`{"email":"DEMO@example.com","password":"password-123"}`, "")
	if loginResponse.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d; body: %s", loginResponse.Code, http.StatusOK, loginResponse.Body)
	}

	var loggedIn authResponse
	if err := json.Unmarshal(loginResponse.Body.Bytes(), &loggedIn); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if loggedIn.AccessToken == "" || loggedIn.TokenType != "Bearer" {
		t.Fatalf("unexpected login token response: %+v", loggedIn)
	}

	badCredentialsResponse := performAuthRequest(t, server, http.MethodPost, "/api/v1/auth/login",
		`{"email":"demo@example.com","password":"wrong-password"}`, "")
	if badCredentialsResponse.Code != http.StatusUnauthorized {
		t.Fatalf("bad credentials status = %d, want %d; body: %s",
			badCredentialsResponse.Code, http.StatusUnauthorized, badCredentialsResponse.Body)
	}
}

func TestAuthenticationMiddlewareRejectsMissingAndInvalidTokens(t *testing.T) {
	const secret = "test-auth-token-secret-with-at-least-32-bytes"
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	server := AuthenticationMiddleware(service.NewTokenManager(secret, time.Hour), next)

	for _, test := range []struct {
		name          string
		authorization string
		wantError     string
	}{
		{name: "missing token", wantError: "authentication required"},
		{name: "invalid token", authorization: "Bearer invalid", wantError: "invalid or expired token"},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := performAuthRequest(t, server, http.MethodGet, "/api/v1/private", "", test.authorization)
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusUnauthorized, response.Body)
			}
			if !strings.Contains(response.Body.String(), test.wantError) {
				t.Fatalf("response body %q does not contain %q", response.Body.String(), test.wantError)
			}
		})
	}
}

func performAuthRequest(t *testing.T, server http.Handler, method, path, body, authorization string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	if authorization != "" {
		request.Header.Set("Authorization", authorization)
	}
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	return response
}
