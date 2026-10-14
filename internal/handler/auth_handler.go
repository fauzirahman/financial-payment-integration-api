package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/fauzirahman/financial-payment-integration-api/internal/service"
	"github.com/jackc/pgx/v5/pgconn"
)

type AuthHandler struct { auth *service.AuthService; tokens *service.TokenManager }
func NewAuthHandler(auth *service.AuthService, tokens *service.TokenManager) *AuthHandler { return &AuthHandler{auth:auth, tokens:tokens} }

type authRequest struct { Name string `json:"name"`; Email string `json:"email"`; Password string `json:"password"` }
type authResponse struct { User any `json:"user"`; AccessToken string `json:"access_token"`; TokenType string `json:"token_type"`; ExpiresIn int64 `json:"expires_in"` }

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost { w.Header().Set("Allow", http.MethodPost); writeError(w, http.StatusMethodNotAllowed, "method not allowed"); return }
	var req authRequest
	if err := decodeAuthRequest(w, r, &req); err != nil { writeError(w, http.StatusBadRequest, "invalid request body"); return }
	user, err := h.auth.Register(r.Context(), req.Name, req.Email, req.Password)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { writeError(w, http.StatusConflict, "email already registered"); return }
		if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "valid email") || strings.Contains(err.Error(), "password") { writeError(w, http.StatusBadRequest, err.Error()); return }
		writeError(w, http.StatusInternalServerError, "internal server error"); return
	}
	h.respondWithToken(w, http.StatusCreated, user.ID, user)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost { w.Header().Set("Allow", http.MethodPost); writeError(w, http.StatusMethodNotAllowed, "method not allowed"); return }
	var req authRequest
	if err := decodeAuthRequest(w, r, &req); err != nil { writeError(w, http.StatusBadRequest, "invalid request body"); return }
	user, err := h.auth.Authenticate(r.Context(), req.Email, req.Password)
	if errors.Is(err, service.ErrInvalidCredentials) { writeError(w, http.StatusUnauthorized, "invalid email or password"); return }
	if err != nil { writeError(w, http.StatusInternalServerError, "internal server error"); return }
	h.respondWithToken(w, http.StatusOK, user.ID, user)
}

func decodeAuthRequest(w http.ResponseWriter, r *http.Request, req *authRequest) error {
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)); d.DisallowUnknownFields()
	if err := d.Decode(req); err != nil { return err }
	if err := d.Decode(&struct{}{}); !errors.Is(err, io.EOF) { return errors.New("multiple JSON values") }
	return nil
}

func (h *AuthHandler) respondWithToken(w http.ResponseWriter, status int, id string, user any) {
	token, err := h.tokens.Issue(id, time.Time{})
	if err != nil { writeError(w, http.StatusInternalServerError, "authentication is not configured"); return }
	writeJSON(w, status, authResponse{User:user, AccessToken:token, TokenType:"Bearer", ExpiresIn:3600})
}
