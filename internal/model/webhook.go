package model

type WebhookEvent struct {
	EventID   string `json:"event_id"`
	Type      string `json:"type"`
	Reference string `json:"reference"`
}