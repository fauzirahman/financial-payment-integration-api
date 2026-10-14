package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type TokenManager struct { secret []byte; ttl time.Duration }
type tokenClaims struct { Subject string `json:"sub"`; Expires int64 `json:"exp"`; Issued int64 `json:"iat"` }

func NewTokenManager(secret string, ttl time.Duration) *TokenManager {
	return &TokenManager{secret: []byte(secret), ttl: ttl}
}

func (m *TokenManager) Issue(userID string, _ time.Time) (string, error) {
	if len(m.secret) < 32 { return "", errors.New("AUTH_TOKEN_SECRET must contain at least 32 bytes") }
	now := time.Now().UTC()
	header, _ := json.Marshal(map[string]string{"alg":"HS256", "typ":"JWT"})
	claims, _ := json.Marshal(tokenClaims{Subject:userID, Issued:now.Unix(), Expires:now.Add(m.ttl).Unix()})
	unsigned := base64.RawURLEncoding.EncodeToString(header)+"."+base64.RawURLEncoding.EncodeToString(claims)
	mac := hmac.New(sha256.New, m.secret); _, _ = mac.Write([]byte(unsigned))
	return unsigned+"."+base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func (m *TokenManager) Validate(token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 || len(m.secret) < 32 { return "", errors.New("invalid token") }
	unsigned := parts[0]+"."+parts[1]
	sig, err := base64.RawURLEncoding.DecodeString(parts[2]); if err != nil { return "", errors.New("invalid token") }
	mac := hmac.New(sha256.New, m.secret); _, _ = mac.Write([]byte(unsigned))
	if !hmac.Equal(sig, mac.Sum(nil)) { return "", errors.New("invalid token") }
	var header map[string]string
	b, err := base64.RawURLEncoding.DecodeString(parts[0]); if err != nil || json.Unmarshal(b, &header) != nil || header["alg"] != "HS256" { return "", errors.New("invalid token") }
	b, err = base64.RawURLEncoding.DecodeString(parts[1]); if err != nil { return "", errors.New("invalid token") }
	var claims tokenClaims
	if json.Unmarshal(b, &claims) != nil || claims.Subject == "" || claims.Expires <= time.Now().Unix() { return "", errors.New("invalid token") }
	return claims.Subject, nil
}
