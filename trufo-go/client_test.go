package trufo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIKeyHeaderSent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-API-Key"); got != "gproduct-admin:secret" {
			t.Errorf("X-API-Key = %q", got)
		}
		if got := r.Header.Get("Authorization"); got != "" {
			t.Errorf("Authorization should be empty, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, WithAPIKey("gproduct-admin:secret"))
	var out statusResponse
	if err := c.post(context.Background(), "/anything", map[string]any{}, &out); err != nil {
		t.Fatal(err)
	}
	if out.Status != "ok" {
		t.Errorf("status = %q", out.Status)
	}
}

func TestBearerHeaderSent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer tok123" {
			t.Errorf("Authorization = %q", got)
		}
		if got := r.Header.Get("X-API-Key"); got != "" {
			t.Errorf("X-API-Key should be empty, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, WithTokens("tok123", "ref456"))
	var out statusResponse
	if err := c.post(context.Background(), "/anything", map[string]any{}, &out); err != nil {
		t.Fatal(err)
	}
}

func TestErrorEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-ID", "req-abc")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]any{"detail": "InsufficientPermissionForAction", "action": "create_apikey"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, WithAPIKey("gproduct-admin:secret"))
	err := c.post(context.Background(), "/x", map[string]any{}, nil)
	var ae *APIError
	if !errors.As(err, &ae) {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if ae.StatusCode != 403 || ae.Detail != "InsufficientPermissionForAction" || ae.Action != "create_apikey" {
		t.Errorf("unexpected error: %+v", ae)
	}
	if ae.RequestID != "req-abc" {
		t.Errorf("RequestID = %q", ae.RequestID)
	}
	if !Is(err, 403, "InsufficientPermissionForAction") {
		t.Error("Is() should match")
	}
	if Is(err, 403, "Other") || Is(err, 404, "") {
		t.Error("Is() should not match")
	}
}

func TestFieldErrors422(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(`{"detail":[{"type":"missing","loc":["body","gp_id"],"msg":"Field required"},{"type":"missing","loc":["body","name"],"msg":"Field required"}]}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, WithAPIKey("gproduct-admin:secret"))
	err := c.post(context.Background(), "/x", map[string]any{}, nil)
	var ae *APIError
	if !errors.As(err, &ae) {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if len(ae.FieldErrors) != 2 || fmt.Sprint(ae.FieldErrors[0].Loc[1]) != "gp_id" || ae.FieldErrors[0].Msg != "Field required" {
		t.Errorf("unexpected field errors: %+v", ae.FieldErrors)
	}
	if !Is(err, 422, "") {
		t.Error("Is() should match 422 with any detail")
	}
}

func TestRefreshOn401(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case AccountRefreshPath:
			body, _ := io.ReadAll(r.Body)
			var b map[string]any
			json.Unmarshal(body, &b)
			if b["refresh_token"] != "old-refresh" {
				t.Errorf("refresh body = %s", body)
			}
			if r.Header.Get("Authorization") != "" || r.Header.Get("X-API-Key") != "" {
				t.Errorf("refresh must carry no credential header")
			}
			json.NewEncoder(w).Encode(TokenPair{AccessToken: "new-access", RefreshToken: "new-refresh"})
		case "/data":
			calls++
			if r.Header.Get("Authorization") == "Bearer new-access" {
				json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
				return
			}
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]any{"detail": "TokenExpired"})
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	var refreshed []string
	c := NewClient(srv.URL, WithTokens("old-access", "old-refresh"))
	c.OnTokenRefresh = func(a, r string) { refreshed = []string{a, r} }

	var out statusResponse
	if err := c.post(context.Background(), "/data", map[string]any{}, &out); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Errorf("expected 2 calls (initial 401 + retry), got %d", calls)
	}
	if c.AccessToken != "new-access" || c.RefreshToken != "new-refresh" {
		t.Errorf("tokens not updated: %q / %q", c.AccessToken, c.RefreshToken)
	}
	if len(refreshed) != 2 || refreshed[0] != "new-access" {
		t.Errorf("OnTokenRefresh not called: %v", refreshed)
	}
}

func TestNoCredentialNoHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" || r.Header.Get("X-API-Key") != "" {
			t.Errorf("unexpected credential header")
		}
		json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
	}))
	defer srv.Close()
	c := NewClient(srv.URL)
	var out statusResponse
	if err := c.post(context.Background(), "/x", map[string]any{}, &out); err != nil {
		t.Fatal(err)
	}
}

func TestAPIKeyScope(t *testing.T) {
	c := NewClient(APIURL, WithAPIKey("trufo-api:abc"))
	if c.APIKeyScope() != "trufo-api" {
		t.Errorf("scope = %q", c.APIKeyScope())
	}
	if NewClient(APIURL).APIKeyScope() != "" {
		t.Error("empty key should have empty scope")
	}
}

func TestTestHostAuthURL(t *testing.T) {
	c := NewClient(APIURLTest)
	if c.AuthURL != APIURL {
		t.Errorf("test host AuthURL should be %q, got %q", APIURL, c.AuthURL)
	}
	if NewClient(APIURLEurope).AuthURL != APIURLEurope {
		t.Error("eu host AuthURL should be itself")
	}
}
