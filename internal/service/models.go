package service

import "github.com/google/uuid"

type WebhookRequest struct {
	NewIP  string    `json:"new_ip"`
	UserID uuid.UUID `json:"guid"`
	Ts     int64     `json:"ts"`
}
