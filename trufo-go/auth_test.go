package trufo

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClientRejectsPlaintext(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected panic for plaintext non-loopback base URL")
		}
	}()
	NewClient("http://example.com")
}

func TestNewClientAllowsLoopbackHTTPAndTrimsSlash(t *testing.T) {
	c := NewClient("http://127.0.0.1:8080/")
	if c.BaseURL != "http://127.0.0.1:8080" {
		t.Errorf("BaseURL = %q, want trimmed", c.BaseURL)
	}
}

func TestDeviceFlowPolling(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc(DeviceAuthorizePath, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != "trufo-api:k" {
			t.Errorf("device authorize must use the API key, got %q", r.Header.Get("X-API-Key"))
		}
		json.NewEncoder(w).Encode(DeviceAuthResponse{
			DeviceCode: "dc", UserCode: "ABCD-EFGH",
			VerificationURI: "https://app.trufo.ai/device", VerificationURIComplete: "https://app.trufo.ai/device?user_code=ABCD-EFGH",
			ExpiresIn: 300, Interval: 1,
		})
	})
	var pollCount int
	mux.HandleFunc(DeviceTokenPath, func(w http.ResponseWriter, r *http.Request) {
		pollCount++
		switch pollCount {
		case 1:
			w.WriteHeader(400)
			json.NewEncoder(w).Encode(map[string]any{"detail": "authorization_pending"})
		case 2:
			w.WriteHeader(400)
			json.NewEncoder(w).Encode(map[string]any{"detail": "DeviceCodeNotFound"})
		case 3:
			w.WriteHeader(400)
			json.NewEncoder(w).Encode(map[string]any{"detail": "access_denied"})
		case 4:
			json.NewEncoder(w).Encode(TokenPair{AccessToken: "a", RefreshToken: "r"})
		}
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := NewClient(srv.URL, WithAPIKey("trufo-api:k"))
	ctx := context.Background()

	auth, err := c.InitiateDeviceAuth(ctx)
	if err != nil || auth.DeviceCode != "dc" {
		t.Fatalf("InitiateDeviceAuth = %+v, %v", auth, err)
	}

	if _, err := c.PollDeviceToken(ctx, "dc"); !errors.Is(err, ErrAuthorizationPending) {
		t.Errorf("poll 1: expected ErrAuthorizationPending, got %v", err)
	}
	if _, err := c.PollDeviceToken(ctx, "dc"); err == nil || err.Error() != "device code expired; please try again" {
		t.Errorf("poll 2 (DeviceCodeNotFound): got %v", err)
	}
	if _, err := c.PollDeviceToken(ctx, "dc"); err == nil || err.Error() != "authorization was denied by the user" {
		t.Errorf("poll 3 (access_denied): got %v", err)
	}
	tokens, err := c.PollDeviceToken(ctx, "dc")
	if err != nil || tokens.AccessToken != "a" || tokens.RefreshToken != "r" {
		t.Errorf("poll 4: got %+v, %v", tokens, err)
	}
}
