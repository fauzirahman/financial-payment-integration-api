package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/fauzirahman/financial-payment-integration-api/internal/model"
	"github.com/fauzirahman/financial-payment-integration-api/internal/service"
	"github.com/jackc/pgx/v5"
)

type WebhookHandler struct {
	service *service.PaymentService
	secret  []byte
}

func NewWebhookHandler(paymentService *service.PaymentService, secret string) *WebhookHandler {
	return &WebhookHandler{service: paymentService, secret: []byte(secret)}
}

func (h *WebhookHandler) HandlePaymentWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if len(h.secret) == 0 {
		writeError(w, http.StatusServiceUnavailable, "webhook is not configured")
		return
	}

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 64<<10))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if !validWebhookSignature(h.secret, body, r.Header.Get("X-Webhook-Signature")) {
		writeError(w, http.StatusUnauthorized, "invalid webhook signature")
		return
	}

	var event model.WebhookEvent
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&event); err != nil {
		writeError(w, http.StatusBadRequest, "invalid webhook event")
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "request body must contain a single JSON object")
		return
	}

	duplicate, err := h.service.ProcessWebhookEvent(r.Context(), event)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidWebhookEvent):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, model.ErrInvalidPaymentTransition):
			writeError(w, http.StatusConflict, "payment status cannot be changed by this event")
		case errors.Is(err, pgx.ErrNoRows):
			writeError(w, http.StatusNotFound, "payment not found")
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"received": true, "duplicate": !duplicate})
}

func validWebhookSignature(secret, body []byte, header string) bool {
	if !strings.HasPrefix(header, "sha256=") {
		return false
	}
	supplied, err := hex.DecodeString(strings.TrimPrefix(header, "sha256="))
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(body)
	return hmac.Equal(supplied, mac.Sum(nil))
}