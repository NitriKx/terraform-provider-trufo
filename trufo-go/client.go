package trufo

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	// DefaultTimeout is the per-request timeout, matching trufo-py's 30s.
	DefaultTimeout = 30 * time.Second
	// SDKVersion identifies this client in the X-TF-Version header (same
	// header name as trufo-py's headers.py). The version headers are
	// self-reported telemetry, not authentication.
	SDKVersion = "trufo-go/0.1.0"
)

// Client is an HTTP client for the Trufo API.
//
// Every request carries exactly one credential, as the API requires: a Bearer
// access token when AccessToken is set, otherwise the API key in X-API-Key.
// Sending both is rejected server-side with 400 MixedCredentials.
type Client struct {
	// BaseURL is the API host, e.g. APIURL, APIURLEurope or APIURLTest.
	BaseURL string
	// AuthURL is the host used for sign-in and token refresh. The test host
	// has no sign-in routes and accepts tokens issued by APIURL, so this
	// defaults to APIURL when BaseURL is APIURLTest.
	AuthURL string
	// APIKey, when non-empty, is sent as X-API-Key ({scope}:{key}).
	APIKey string
	// AccessToken, when non-empty, is sent as Authorization: Bearer.
	AccessToken string
	// RefreshToken is used to obtain a new AccessToken on a 401.
	// Access via tokens()/SetTokens: the provider shares one Client across
	// concurrent CRUD calls, and a 401-triggered refresh mutates the pair.
	RefreshToken string
	// HTTPClient is the underlying HTTP client; nil uses one with
	// DefaultTimeout.
	HTTPClient *http.Client
	// UserAgent is sent as User-Agent; defaults to SDKVersion.
	UserAgent string

	// mu guards AccessToken/RefreshToken against concurrent refreshes.
	mu sync.Mutex

	// OnTokenRefresh, when set, is called after a successful token refresh
	// so callers can persist the rotated pair.
	OnTokenRefresh func(accessToken, refreshToken string)
}

// Option configures a Client.
type Option func(*Client)

func WithBaseURL(url string) Option { return func(c *Client) { c.BaseURL = url } }
func WithAPIKey(key string) Option  { return func(c *Client) { c.APIKey = key } }
func WithTokens(access, refresh string) Option {
	return func(c *Client) { c.AccessToken, c.RefreshToken = access, refresh }
}
func WithHTTPClient(hc *http.Client) Option { return func(c *Client) { c.HTTPClient = hc } }

// NewClient returns a Client for the given API base URL (default APIURL).
// The Trufo API requires TLS 1.3+; the default transport enforces it. Plain
// http:// URLs are only accepted for loopback test servers.
func NewClient(baseURL string, opts ...Option) *Client {
	if baseURL == "" {
		baseURL = APIURL
	}
	c := &Client{BaseURL: baseURL}
	for _, opt := range opts {
		opt(c)
	}
	c.BaseURL = strings.TrimRight(c.BaseURL, "/")
	if c.AuthURL == "" {
		c.AuthURL = c.BaseURL
		if c.BaseURL == APIURLTest {
			// The test host has no sign-in or refresh routes; it accepts
			// tokens issued by the global host.
			c.AuthURL = APIURL
		}
	}
	c.AuthURL = strings.TrimRight(c.AuthURL, "/")
	// Validate after options are applied, so WithBaseURL cannot bypass the
	// plaintext-HTTP guard.
	for _, u := range []string{c.BaseURL, c.AuthURL} {
		if p, err := url.Parse(u); err == nil && p.Scheme == "http" && p.Hostname() != "127.0.0.1" && p.Hostname() != "localhost" && p.Hostname() != "::1" {
			panic("trufo: plaintext HTTP base URL rejected (TLS 1.3+ required): " + u)
		}
	}
	if c.HTTPClient == nil {
		c.HTTPClient = &http.Client{
			Timeout: DefaultTimeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS13},
			},
		}
	}
	if c.UserAgent == "" {
		c.UserAgent = SDKVersion
	}
	return c
}

// apiKeyScope returns the scope prefix of an API key ({scope}:{key}).
func apiKeyScope(key string) string {
	for i := 0; i < len(key); i++ {
		if key[i] == ':' {
			return key[:i]
		}
	}
	return ""
}

// APIKeyScope returns the scope of the configured API key, or "" if none.
func (c *Client) APIKeyScope() string { return apiKeyScope(c.APIKey) }

// SetTokens replaces the access/refresh token pair (e.g. after a refresh or a
// login flow). Safe for concurrent use.
func (c *Client) SetTokens(access, refresh string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.AccessToken = access
	c.RefreshToken = refresh
}

// tokens returns the current access/refresh pair. Safe for concurrent use.
func (c *Client) tokens() (access, refresh string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.AccessToken, c.RefreshToken
}

// do performs one HTTP request against the API. method is usually POST; the
// only GET endpoint is /tasks/{id}. body may be nil. On success the response
// JSON is decoded into out (which may be nil).
func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	req, err := c.newRequest(ctx, c.BaseURL+path, method, body)
	if err != nil {
		return err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("trufo API request failed: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return fmt.Errorf("reading trufo API response: %w", err)
	}

	if resp.StatusCode >= 400 {
		apiErr := parseError(resp.StatusCode, resp.Header.Get("X-Request-ID"), respBody)
		// Retry once with a refreshed token on 401, like TrufoSession.
		_, refresh := c.tokens()
		if resp.StatusCode == http.StatusUnauthorized && refresh != "" {
			if refreshErr := c.Refresh(ctx); refreshErr != nil {
				// Surface both: the original 401 and why the refresh failed
				// (e.g. the refresh token was already rotated/revoked).
				return fmt.Errorf("%w (token refresh failed: %v)", apiErr, refreshErr)
			}
			req2, err := c.newRequest(ctx, c.BaseURL+path, method, body)
			if err != nil {
				return err
			}
			resp2, err := c.HTTPClient.Do(req2)
			if err != nil {
				return fmt.Errorf("trufo API request failed: %w", err)
			}
			defer resp2.Body.Close()
			respBody2, err := io.ReadAll(io.LimitReader(resp2.Body, 32<<20))
			if err != nil {
				return fmt.Errorf("reading trufo API response: %w", err)
			}
			if resp2.StatusCode < 400 {
				return decodeInto(respBody2, out)
			}
			return parseError(resp2.StatusCode, resp2.Header.Get("X-Request-ID"), respBody2)
		}
		return apiErr
	}
	return decodeInto(respBody, out)
}

func decodeInto(body []byte, out any) error {
	if out == nil || len(body) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decoding trufo API response: %w (body: %.200s)", err, body)
	}
	return nil
}

// newRequest builds a request with the JSON body and the single credential
// header the API requires.
func (c *Client) newRequest(ctx context.Context, url, method string, body any) (*http.Request, error) {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("encoding request body: %w", err)
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, rdr)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("X-TF-Version", SDKVersion)
	access, _ := c.tokens()
	switch {
	case access != "":
		req.Header.Set("Authorization", "Bearer "+access)
	case c.APIKey != "":
		req.Header.Set("X-API-Key", c.APIKey)
	}
	return req, nil
}

// post is the common case: POST a JSON body and decode the JSON response.
func (c *Client) post(ctx context.Context, path string, body, out any) error {
	return c.do(ctx, http.MethodPost, path, body, out)
}

// get is for the single GET endpoint, /tasks/{id}.
func (c *Client) get(ctx context.Context, path string, out any) error {
	return c.do(ctx, http.MethodGet, path, nil, out)
}
