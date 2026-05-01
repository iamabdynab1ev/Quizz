package usecase

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"lms-arvand-backend/internal/domain"
)

type webhookDeliveryRepository interface {
	ListActiveByEvent(ctx context.Context, eventType domain.AppEventType) ([]domain.Webhook, error)
	RecordDeliveryResult(ctx context.Context, webhookID string, attemptedAt time.Time, statusCode *int, lastError *string, delivered bool) error
}

type WebhookDispatcher struct {
	logger     *slog.Logger
	repository webhookDeliveryRepository
	client     *http.Client
}

type webhookDeliveryPayload struct {
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	At      time.Time       `json:"at"`
	ActorID *string         `json:"actor_id,omitempty"`
	Payload json.RawMessage `json:"payload"`
}

func NewWebhookDispatcher(logger *slog.Logger, repository webhookDeliveryRepository) *WebhookDispatcher {
	if logger == nil {
		logger = slog.Default()
	}

	return &WebhookDispatcher{
		logger:     logger,
		repository: repository,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (d *WebhookDispatcher) Dispatch(ctx context.Context, auditLog domain.AuditLog) error {
	if d == nil || d.repository == nil {
		return nil
	}

	webhooks, err := d.repository.ListActiveByEvent(ctx, auditLog.Type)
	if err != nil {
		return fmt.Errorf("webhook dispatcher list active by event: %w", err)
	}

	if len(webhooks) == 0 {
		return nil
	}

	payload, err := json.Marshal(webhookDeliveryPayload{
		ID:      auditLog.ID,
		Type:    string(auditLog.Type),
		At:      auditLog.At,
		ActorID: auditLog.ActorID,
		Payload: auditLog.Payload,
	})
	if err != nil {
		return fmt.Errorf("webhook dispatcher marshal payload: %w", err)
	}

	for _, webhook := range webhooks {
		if err := d.deliverWithRetry(ctx, webhook, auditLog.Type, auditLog.ID, payload); err != nil {
			d.logger.ErrorContext(ctx, "webhook delivery failed",
				slog.String("webhook_id", webhook.ID),
				slog.String("event_type", string(auditLog.Type)),
				slog.String("error", err.Error()),
			)
		}
	}

	return nil
}

func (d *WebhookDispatcher) deliverWithRetry(ctx context.Context, webhook domain.Webhook, eventType domain.AppEventType, deliveryID string, payload []byte) error {
	var lastStatusCode *int
	var lastError *string
	attemptedAt := time.Now().UTC()

	for attempt := 0; attempt < 3; attempt++ {
		statusCode, err := d.deliverOnce(ctx, webhook, eventType, deliveryID, payload)
		lastStatusCode = statusCode
		lastError = errorMessage(err)

		if err == nil {
			if repoErr := d.repository.RecordDeliveryResult(ctx, webhook.ID, attemptedAt, lastStatusCode, nil, true); repoErr != nil {
				return fmt.Errorf("webhook dispatcher persist success result: %w", repoErr)
			}
			return nil
		}

		if attempt < 2 {
			backoff := time.Duration(1<<attempt) * time.Second
			select {
			case <-ctx.Done():
				if lastError == nil {
					lastError = errorMessage(ctx.Err())
				}
				attempt = 3
			case <-time.After(backoff):
			}
		}
	}

	if repoErr := d.repository.RecordDeliveryResult(ctx, webhook.ID, attemptedAt, lastStatusCode, lastError, false); repoErr != nil {
		return fmt.Errorf("webhook dispatcher persist failure result: %w", repoErr)
	}

	if lastError == nil {
		return fmt.Errorf("webhook dispatcher delivery failed")
	}

	return fmt.Errorf("webhook dispatcher delivery failed: %s", *lastError)
}

func (d *WebhookDispatcher) deliverOnce(ctx context.Context, webhook domain.Webhook, eventType domain.AppEventType, deliveryID string, payload []byte) (*int, error) {
	timestamp := time.Now().UTC().Format(time.RFC3339)
	signature := signWebhookPayload(webhook.Secret, timestamp, payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhook.URL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-LMS-Event", string(eventType))
	req.Header.Set("X-LMS-Delivery-ID", deliveryID)
	req.Header.Set("X-LMS-Timestamp", timestamp)
	req.Header.Set("X-LMS-Signature", "sha256="+signature)

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	statusCode := resp.StatusCode
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return &statusCode, nil
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	bodyText := strings.TrimSpace(string(body))
	if bodyText == "" {
		bodyText = resp.Status
	}

	return &statusCode, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, bodyText)
}

func signWebhookPayload(secret, timestamp string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

func errorMessage(err error) *string {
	if err == nil {
		return nil
	}

	message := err.Error()
	return &message
}
