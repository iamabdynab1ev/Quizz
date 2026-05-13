package recaptcha

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type verifyResponse struct {
	Success    bool     `json:"success"`
	Score      float64  `json:"score"`
	Action     string   `json:"action"`
	Hostname   string   `json:"hostname"`
	ErrorCodes []string `json:"error-codes"`
}

type Verifier struct {
	client         *http.Client
	secretKey      string
	minScore       float64
	enabled        bool
	expectedAction string
	verifyURL      string
}

func NewVerifier(secretKey string, minScore float64, enabled bool) *Verifier {
	return &Verifier{
		client:         &http.Client{Timeout: 5 * time.Second},
		secretKey:      strings.TrimSpace(secretKey),
		minScore:       minScore,
		enabled:        enabled,
		expectedAction: "register",
		verifyURL:      "https://www.google.com/recaptcha/api/siteverify",
	}
}

func (v *Verifier) Verify(ctx context.Context, token string, remoteIP string) error {
	if !v.enabled {
		return nil
	}

	if v.secretKey == "" {
		return fmt.Errorf("recaptcha secret key is empty")
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return fmt.Errorf("recaptcha token is empty")
	}

	form := url.Values{}
	form.Set("secret", v.secretKey)
	form.Set("response", token)
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.verifyURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("recaptcha build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := v.client.Do(req)
	if err != nil {
		return fmt.Errorf("recaptcha request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("recaptcha unexpected status: %d", resp.StatusCode)
	}

	var result verifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("recaptcha decode response: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("recaptcha failed (success=false errors=%s)", strings.Join(result.ErrorCodes, ","))
	}

	if v.expectedAction != "" && result.Action != v.expectedAction {
		return fmt.Errorf("recaptcha action mismatch (got=%q want=%q)", result.Action, v.expectedAction)
	}

	if result.Score < v.minScore {
		return fmt.Errorf("recaptcha score too low (score=%.2f min=%.2f)", result.Score, v.minScore)
	}

	return nil
}
