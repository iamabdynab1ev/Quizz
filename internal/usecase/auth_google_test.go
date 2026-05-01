package usecase

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGoogleTokenInfoVerifierVerifyIDToken(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("id_token"); got != "valid-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"aud":"test-client-id",
			"sub":"google-subject-123",
			"email":"student@example.com",
			"email_verified":"true",
			"given_name":"Ali",
			"family_name":"Karimov",
			"picture":"https://example.com/avatar.jpg",
			"iss":"https://accounts.google.com",
			"exp":"4102444800"
		}`))
	}))
	defer server.Close()

	verifier := &GoogleTokenInfoVerifier{
		client:    server.Client(),
		clientID:  "test-client-id",
		tokenInfo: server.URL,
	}

	identity, err := verifier.VerifyIDToken(context.Background(), "valid-token")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if identity.Subject != "google-subject-123" {
		t.Fatalf("unexpected subject: %s", identity.Subject)
	}

	if identity.Email != "student@example.com" {
		t.Fatalf("unexpected email: %s", identity.Email)
	}

	if !identity.EmailVerified {
		t.Fatalf("expected email to be verified")
	}
}

func TestGoogleTokenInfoVerifierRejectsWrongAudience(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"aud":"other-client-id",
			"sub":"google-subject-123",
			"email":"student@example.com",
			"email_verified":"true",
			"iss":"accounts.google.com",
			"exp":"4102444800"
		}`))
	}))
	defer server.Close()

	verifier := &GoogleTokenInfoVerifier{
		client:    server.Client(),
		clientID:  "test-client-id",
		tokenInfo: server.URL,
	}

	if _, err := verifier.VerifyIDToken(context.Background(), "valid-token"); err == nil {
		t.Fatalf("expected audience mismatch error")
	}
}
