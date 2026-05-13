package recaptcha

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestVerifierSkipsCheckWhenDisabled(t *testing.T) {
	t.Parallel()

	verifier := NewVerifier("", 0.5, false)
	if err := verifier.Verify(context.Background(), "", ""); err != nil {
		t.Fatalf("Verify() error = %v, want nil", err)
	}
}

func TestVerifierRequiresSecretWhenEnabled(t *testing.T) {
	t.Parallel()

	verifier := NewVerifier("", 0.5, true)
	err := verifier.Verify(context.Background(), "token", "")
	if err == nil || !strings.Contains(err.Error(), "secret key") {
		t.Fatalf("Verify() error = %v, want secret key error", err)
	}
}

func TestVerifierRejectsActionMismatch(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm() error = %v", err)
		}
		if got := r.Form.Get("secret"); got != "secret" {
			t.Fatalf("secret = %q, want secret", got)
		}
		if got := r.Form.Get("response"); got != "token" {
			t.Fatalf("response = %q, want token", got)
		}
		fmt.Fprint(w, `{"success":true,"score":0.9,"action":"login"}`)
	}))
	defer server.Close()

	verifier := NewVerifier("secret", 0.5, true)
	verifier.verifyURL = server.URL

	err := verifier.Verify(context.Background(), "token", "")
	if err == nil || !strings.Contains(err.Error(), "action mismatch") {
		t.Fatalf("Verify() error = %v, want action mismatch", err)
	}
}

func TestVerifierAcceptsValidRegisterToken(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"success":true,"score":0.9,"action":"register"}`)
	}))
	defer server.Close()

	verifier := NewVerifier("secret", 0.5, true)
	verifier.verifyURL = server.URL

	if err := verifier.Verify(context.Background(), "token", "127.0.0.1"); err != nil {
		t.Fatalf("Verify() error = %v, want nil", err)
	}
}
