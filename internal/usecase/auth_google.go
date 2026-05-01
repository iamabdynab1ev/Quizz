package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"lms-arvand-backend/internal/domain"
)

type googleTokenVerifier interface {
	VerifyIDToken(ctx context.Context, idToken string) (GoogleIdentity, error)
}

type GoogleIdentity struct {
	Subject       string
	Email         string
	EmailVerified bool
	GivenName     string
	FamilyName    string
	Picture       string
	Audience      string
}

type GoogleTokenInfoVerifier struct {
	client    *http.Client
	clientID  string
	tokenInfo string
}

func NewGoogleTokenInfoVerifier(clientID string) *GoogleTokenInfoVerifier {
	return &GoogleTokenInfoVerifier{
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		clientID:  strings.TrimSpace(clientID),
		tokenInfo: "https://oauth2.googleapis.com/tokeninfo",
	}
}

func (v *GoogleTokenInfoVerifier) VerifyIDToken(ctx context.Context, idToken string) (GoogleIdentity, error) {
	idToken = strings.TrimSpace(idToken)
	if idToken == "" {
		return GoogleIdentity{}, domain.ValidationError("id_token is required")
	}

	if v.clientID == "" {
		return GoogleIdentity{}, domain.UnavailableError("google sign-in is not configured")
	}

	tokenInfoURL, err := url.Parse(v.tokenInfo)
	if err != nil {
		return GoogleIdentity{}, fmt.Errorf("usecase google verify parse tokeninfo url: %w", err)
	}

	query := tokenInfoURL.Query()
	query.Set("id_token", idToken)
	tokenInfoURL.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, tokenInfoURL.String(), nil)
	if err != nil {
		return GoogleIdentity{}, fmt.Errorf("usecase google verify create request: %w", err)
	}

	response, err := v.client.Do(request)
	if err != nil {
		return GoogleIdentity{}, fmt.Errorf("usecase google verify request tokeninfo: %w", domain.UnavailableError("google token verification is unavailable"))
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return GoogleIdentity{}, domain.UnauthorizedError("google id token is invalid")
	}

	var payload struct {
		Audience      string      `json:"aud"`
		Subject       string      `json:"sub"`
		Email         string      `json:"email"`
		EmailVerified interface{} `json:"email_verified"`
		GivenName     string      `json:"given_name"`
		FamilyName    string      `json:"family_name"`
		Picture       string      `json:"picture"`
		Issuer        string      `json:"iss"`
		ExpiresAt     string      `json:"exp"`
	}

	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return GoogleIdentity{}, fmt.Errorf("usecase google verify decode tokeninfo: %w", err)
	}

	if payload.Audience != v.clientID {
		return GoogleIdentity{}, domain.UnauthorizedError("google id token audience mismatch")
	}

	if payload.Issuer != "accounts.google.com" && payload.Issuer != "https://accounts.google.com" {
		return GoogleIdentity{}, domain.UnauthorizedError("google id token issuer is invalid")
	}

	if strings.TrimSpace(payload.Subject) == "" {
		return GoogleIdentity{}, domain.UnauthorizedError("google id token subject is missing")
	}

	if strings.TrimSpace(payload.Email) == "" {
		return GoogleIdentity{}, domain.UnauthorizedError("google id token email is missing")
	}

	emailVerified, err := parseGoogleVerified(payload.EmailVerified)
	if err != nil {
		return GoogleIdentity{}, domain.UnauthorizedError("google id token email_verified is invalid")
	}

	if !emailVerified {
		return GoogleIdentity{}, domain.UnauthorizedError("google account email is not verified")
	}

	if payload.ExpiresAt != "" {
		expiresAtUnix, err := strconv.ParseInt(payload.ExpiresAt, 10, 64)
		if err != nil {
			return GoogleIdentity{}, domain.UnauthorizedError("google id token exp is invalid")
		}

		if time.Unix(expiresAtUnix, 0).Before(time.Now().UTC()) {
			return GoogleIdentity{}, domain.UnauthorizedError("google id token is expired")
		}
	}

	return GoogleIdentity{
		Subject:       strings.TrimSpace(payload.Subject),
		Email:         strings.TrimSpace(payload.Email),
		EmailVerified: emailVerified,
		GivenName:     strings.TrimSpace(payload.GivenName),
		FamilyName:    strings.TrimSpace(payload.FamilyName),
		Picture:       strings.TrimSpace(payload.Picture),
		Audience:      strings.TrimSpace(payload.Audience),
	}, nil
}

func parseGoogleVerified(value interface{}) (bool, error) {
	switch typed := value.(type) {
	case bool:
		return typed, nil
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(typed))
		if err != nil {
			return false, err
		}

		return parsed, nil
	default:
		return false, fmt.Errorf("unsupported type %T", value)
	}
}
