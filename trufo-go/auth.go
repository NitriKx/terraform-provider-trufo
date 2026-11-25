package trufo

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

// ErrNoBrowser signals that the loopback flow cannot run because no local
// browser could be launched or no loopback port could be bound (the headless
// signal, as in trufo-py). Callers should fall back to the device flow.
var ErrNoBrowser = errors.New("no local browser available")

// TokenPair holds an access + refresh token pair.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// DeviceAuthResponse is the response of POST /account/device/authorize.
type DeviceAuthResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

// ErrAuthorizationPending is returned by PollDeviceToken when the user has
// not approved yet; keep polling.
var ErrAuthorizationPending = errors.New("authorization_pending")

// InitiateDeviceAuth begins the RFC 8628 device authorization flow. It
// requires an API key with the trufo-api scope.
func (c *Client) InitiateDeviceAuth(ctx context.Context) (*DeviceAuthResponse, error) {
	var out DeviceAuthResponse
	// The device endpoints authenticate with the API key, not a Bearer token.
	if err := c.postWithAPIKey(ctx, c.AuthURL+DeviceAuthorizePath, map[string]any{}, &out); err != nil {
		return nil, fmt.Errorf("device authorize failed: %w", err)
	}
	return &out, nil
}

// PollDeviceToken polls POST /account/device/token once. It returns
// ErrAuthorizationPending while the user has not approved; callers should
// sleep resp.Interval and retry until resp.ExpiresIn elapses.
func (c *Client) PollDeviceToken(ctx context.Context, deviceCode string) (*TokenPair, error) {
	var out TokenPair
	err := c.postWithAPIKey(ctx, c.AuthURL+DeviceTokenPath, map[string]any{"device_code": deviceCode}, &out)
	if err != nil {
		var ae *APIError
		if errors.As(err, &ae) && ae.StatusCode == 400 && ae.Detail == "authorization_pending" {
			return nil, ErrAuthorizationPending
		}
		if errors.As(err, &ae) && ae.StatusCode == 400 && ae.Detail == "access_denied" {
			return nil, errors.New("authorization was denied by the user")
		}
		if errors.As(err, &ae) && ae.StatusCode == 400 && (ae.Detail == "expired_token" || ae.Detail == "DeviceCodeNotFound") {
			return nil, errors.New("device code expired; please try again")
		}
		return nil, fmt.Errorf("device token poll failed: %w", err)
	}
	return &out, nil
}

// DeviceFlow runs the full device authorization flow, calling prompt with the
// verification URL and user code so the caller can display them, and polling
// until approval or expiry. On success the client's tokens are set.
func (c *Client) DeviceFlow(ctx context.Context, prompt func(verificationURIComplete, userCode string)) (*TokenPair, error) {
	if c.APIKey == "" || c.APIKeyScope() != "trufo-api" {
		return nil, errors.New("device flow requires an API key with the trufo-api scope")
	}
	auth, err := c.InitiateDeviceAuth(ctx)
	if err != nil {
		return nil, err
	}
	if prompt != nil {
		prompt(auth.VerificationURIComplete, auth.UserCode)
	}
	interval := time.Duration(auth.Interval) * time.Second
	if interval <= 0 {
		interval = 5 * time.Second
	}
	deadline := time.Now().Add(time.Duration(auth.ExpiresIn) * time.Second)
	for {
		tokens, err := c.PollDeviceToken(ctx, auth.DeviceCode)
		if err == nil {
			c.SetTokens(tokens.AccessToken, tokens.RefreshToken)
			if c.OnTokenRefresh != nil {
				c.OnTokenRefresh(tokens.AccessToken, tokens.RefreshToken)
			}
			return tokens, nil
		}
		if !errors.Is(err, ErrAuthorizationPending) {
			return nil, err
		}
		if time.Now().Add(interval).After(deadline) {
			return nil, errors.New("timed out waiting for device authorization")
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
		}
	}
}

// Refresh exchanges the client's refresh token for a new pair (POST
// /account/refresh). Refresh tokens are single-use and rotated on every call.
// On success the client's tokens are updated.
func (c *Client) Refresh(ctx context.Context) error {
	_, refresh := c.tokens()
	if refresh == "" {
		return errors.New("no refresh token available")
	}
	// /account/refresh takes no credential header.
	body, err := json.Marshal(map[string]any{"refresh_token": refresh})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.AuthURL+AccountRefreshPath, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("X-TF-Version", SDKVersion)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("token refresh failed: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return parseError(resp.StatusCode, resp.Header.Get("X-Request-ID"), respBody)
	}
	var out TokenPair
	if err := decodeInto(respBody, &out); err != nil {
		return err
	}
	c.SetTokens(out.AccessToken, out.RefreshToken)
	if c.OnTokenRefresh != nil {
		c.OnTokenRefresh(out.AccessToken, out.RefreshToken)
	}
	return nil
}

// Logout revokes the client's refresh token (idempotent; the access token
// stays valid until it expires). Auth: access token. Like Refresh, it goes to
// AuthURL: the test host has no account routes and accepts tokens issued by
// the global host.
func (c *Client) Logout(ctx context.Context) error {
	access, refresh := c.tokens()
	if access == "" {
		return errors.New("no access token")
	}
	b, err := json.Marshal(map[string]any{"refresh_token": refresh})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.AuthURL+AccountLogoutPath, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("X-TF-Version", SDKVersion)
	req.Header.Set("Authorization", "Bearer "+access)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("logout failed: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return parseError(resp.StatusCode, resp.Header.Get("X-Request-ID"), respBody)
	}
	return nil
}

// postWithAPIKey posts with the X-API-Key header even when a Bearer token is
// set (the device and loopback endpoints require the trufo-api key).
func (c *Client) postWithAPIKey(ctx context.Context, url string, body, out any) error {
	if c.APIKey == "" {
		return errors.New("no API key configured")
	}
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("X-TF-Version", SDKVersion)
	req.Header.Set("X-API-Key", c.APIKey)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return parseError(resp.StatusCode, resp.Header.Get("X-Request-ID"), respBody)
	}
	return decodeInto(respBody, out)
}

// --- loopback flow (RFC 8252 + PKCE) ---

// LoopbackLogin runs the loopback sign-in: it binds a one-shot listener on
// 127.0.0.1, opens the Trufo webapp in the local browser, waits for the
// redirect, and exchanges the code at POST /account/loopback/token. It
// requires an API key with the trufo-api scope. Use DeviceFlow on headless
// machines, where the browser cannot reach the local listener.
func (c *Client) LoopbackLogin(ctx context.Context, appURL string, timeout time.Duration) (*TokenPair, error) {
	if c.APIKey == "" || c.APIKeyScope() != "trufo-api" {
		return nil, errors.New("loopback flow requires an API key with the trufo-api scope")
	}
	if appURL == "" {
		appURL = AppURL
	}
	appURL = strings.TrimRight(appURL, "/")
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}

	verifier, challenge, err := generatePKCE()
	if err != nil {
		return nil, err
	}
	state, err := randomURLSafe(32)
	if err != nil {
		return nil, err
	}

	// Bind 127.0.0.1 specifically, never 0.0.0.0: binding all interfaces
	// would expose the callback — and the authorization code — to the network.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("%w: binding loopback listener: %v", ErrNoBrowser, err)
	}
	defer ln.Close()
	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", ln.Addr().(*net.TCPAddr).Port)

	q := url.Values{}
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	q.Set("state", state)
	q.Set("redirect_uri", redirectURI)
	authorizeURL := appURL + LoopbackAuthPath + "?" + q.Encode()
	if !openBrowser(authorizeURL) {
		return nil, fmt.Errorf("%w; use the device flow instead", ErrNoBrowser)
	}

	type callbackResult struct {
		code  string
		state string
	}
	resCh := make(chan callbackResult, 1)
	var once sync.Once
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/callback" {
			// Ignore anything that is not our callback (a favicon prefetch,
			// a port scanner, a second callback) so it cannot consume the
			// one-shot listener.
			http.NotFound(w, r)
			return
		}
		// Only a callback carrying a code consumes the one-shot wait; a
		// codeless probe on the right path keeps us waiting, like trufo-py.
		if code := r.URL.Query().Get("code"); code != "" {
			once.Do(func() {
				resCh <- callbackResult{code: code, state: r.URL.Query().Get("state")}
			})
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, "<html><body style='font-family:sans-serif;text-align:center;padding:60px'><h2>Signed in</h2><p>You can close this window and return to your terminal.</p></body></html>")
	})}
	go srv.Serve(ln) //nolint:errcheck // server closed via defer below
	defer srv.Close()

	var res callbackResult
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(timeout):
		return nil, errors.New("timed out waiting for browser sign-in")
	case res = <-resCh:
	}
	// resCh only carries non-empty codes (the handler filters codeless probes).
	// Constant-time comparison: the state check stops a third party from
	// feeding us a code that belongs to a flow we did not start.
	if subtle.ConstantTimeCompare([]byte(res.state), []byte(state)) != 1 {
		return nil, errors.New("state mismatch on loopback callback; sign-in aborted")
	}

	var out TokenPair
	err = c.postWithAPIKey(ctx, c.AuthURL+LoopbackTokenPath, map[string]any{
		"code":          res.code,
		"code_verifier": verifier,
		"redirect_uri":  redirectURI,
	}, &out)
	if err != nil {
		if ae, ok := err.(*APIError); ok && ae.Detail == "invalid_grant" {
			return nil, errors.New("sign-in code was rejected or expired; please try again")
		}
		return nil, fmt.Errorf("loopback token exchange failed: %w", err)
	}
	c.SetTokens(out.AccessToken, out.RefreshToken)
	if c.OnTokenRefresh != nil {
		c.OnTokenRefresh(out.AccessToken, out.RefreshToken)
	}
	return &out, nil
}

func generatePKCE() (verifier, challenge string, err error) {
	verifier, err = randomURLSafe(32) // RFC 7636 §4.1 allows 43-128 chars
	if err != nil {
		return "", "", err
	}
	digest := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(digest[:])
	return verifier, challenge, nil
}

func randomURLSafe(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// openBrowser tries to open url in the system browser; it returns false when
// no browser can be launched (the headless signal, as in trufo-py's
// webbrowser.open returning False). The command is run with a short timeout:
// a present-but-broken opener (e.g. xdg-open without a browser) exits non-zero
// and counts as "no browser" so the caller can fall back to the device flow.
func openBrowser(rawURL string) bool {
	var name string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		name, args = "open", []string{rawURL}
	case "windows":
		name, args = "rundll32", []string{"url.dll,FileProtocolHandler", rawURL}
	default:
		name, args = "xdg-open", []string{rawURL}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, name, args...).Run() == nil
}
