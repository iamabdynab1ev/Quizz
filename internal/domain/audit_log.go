package domain

import (
	"encoding/json"
	"time"
)

type AuditLog struct {
	ID                       string          `json:"id"`
	Type                     AppEventType    `json:"type"`
	At                       time.Time       `json:"at"`
	ActorID                  *string         `json:"actor_id,omitempty"`
	Payload                  json.RawMessage `json:"payload"`
	WebhookDispatchedAt      *time.Time      `json:"-"`
	WebhookProcessingAt      *time.Time      `json:"-"`
	WebhookDispatchAttempts  int             `json:"-"`
	WebhookNextDispatchAt    *time.Time      `json:"-"`
	WebhookLastDispatchError *string         `json:"-"`
}

type CreateAuditLogParams struct {
	Type    AppEventType    `json:"type"`
	ActorID *string         `json:"actor_id,omitempty"`
	Payload json.RawMessage `json:"payload"`
}

type AuditLogListFilter struct {
	Type    *AppEventType
	ActorID *string
	Limit   int
	Offset  int
}
