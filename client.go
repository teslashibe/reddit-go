package reddit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	impersonate "github.com/teslashibe/impersonate-go"
)

const (
	oauthBaseURL     = "https://oauth.reddit.com"
	wwwBaseURL       = "https://www.reddit.com"
	matrixBaseURL    = "https://matrix.redditspace.com"
	defaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
	defaultTimeout   = 30 * time.Second
)

// Options configures the messenger client. Token is required.
type Options struct {
	// Token is the token_v2 cookie value from a logged-in Reddit session.
	// Get it from browser DevTools → Application → Cookies → reddit.com → token_v2.
	Token string

	// Cookies is the full cookie set from the same logged-in session,
	// keyed by cookie name (e.g. "reddit_session", "csrf_token", "loid").
	// Optional for the OAuth API — only required for endpoints that
	// reject bearer-only auth, namely the post-insights HTML pages
	// at www.reddit.com/poststats/{id}/. If you intend to call
	// PostInsights, include at minimum reddit_session and token_v2.
	Cookies map[string]string

	// UserAgent overrides the default browser-like User-Agent string.
	UserAgent string

	// RequestTimeout overrides the default 30s HTTP timeout.
	RequestTimeout time.Duration

	// MinRequestGap sets the minimum interval between requests (default 650ms).
	MinRequestGap time.Duration

	// ProxyURL, when set, routes every API request through an HTTP/S proxy
	// (e.g. a Webshare residential gateway). Use the same sticky session as
	// LoginParams.ProxyURL so login and post-login traffic egress from one IP.
	ProxyURL string
}

// Client is the main Reddit API client.
type Client struct {
	httpClient *http.Client
	token      string
	// cookieHeader is the pre-baked Cookie header value for www.reddit.com
	// requests. Computed once at construction so we don't re-stringify
	// on every PostInsights call. Empty when Options.Cookies wasn't set.
	cookieHeader string
	userAgent    string

	gapMu     sync.Mutex
	lastReqAt time.Time
	minGap    time.Duration

	rlMu    sync.Mutex
	rlState RateLimitState

	// authCookies is the cookie set this client was constructed with, kept so
	// callers (e.g. the MCP host's RefreshCredential) can persist a freshly
	// minted session back to their credential store. Set from Options.Cookies.
	authCookies map[string]string
}

// AuthSnapshot returns the client's current bearer token and cookie set, so a
// host can persist a freshly minted credential-login session for reuse.
func (m *Client) AuthSnapshot() (token string, cookies map[string]string) {
	if len(m.authCookies) == 0 {
		return m.token, nil
	}
	out := make(map[string]string, len(m.authCookies))
	for k, v := range m.authCookies {
		out[k] = v
	}
	return m.token, out
}

// NewFromLogin builds a client from a credential-login result, preserving the
// minted bearer and cookies (so AuthSnapshot can hand them back for persistence).
func NewFromLogin(res *LoginResult) *Client {
	if res == nil {
		return New(nil)
	}
	c := New(&Options{
		Token:     res.Token,
		Cookies:   res.Cookies,
		UserAgent: res.UserAgent,
		ProxyURL:  res.ProxyURL,
	})
	c.authCookies = res.Cookies
	return c
}

// New creates a Reddit API client. Options.Token is required.
func New(opts *Options) *Client {
	if opts == nil {
		opts = &Options{}
	}

	ua := defaultUserAgent
	if opts.UserAgent != "" {
		ua = opts.UserAgent
	}

	timeout := defaultTimeout
	if opts.RequestTimeout > 0 {
		timeout = opts.RequestTimeout
	}

	gap := 650 * time.Millisecond
	if opts.MinRequestGap > 0 {
		gap = opts.MinRequestGap
	}

	jar, _ := cookiejar.New(nil)
	httpClient := newHTTPClient(jar, timeout, strings.TrimSpace(opts.ProxyURL))

	// Build the Cookie header once so we don't re-allocate per request.
	// Order doesn't matter to Reddit, but we sort for deterministic
	// output (helps when grepping wire dumps in tests).
	var cookieHeader string
	if len(opts.Cookies) > 0 {
		names := make([]string, 0, len(opts.Cookies))
		for name := range opts.Cookies {
			if name == "" {
				continue
			}
			names = append(names, name)
		}
		sort.Strings(names)
		var b strings.Builder
		for i, name := range names {
			if i > 0 {
				b.WriteString("; ")
			}
			b.WriteString(name)
			b.WriteByte('=')
			b.WriteString(opts.Cookies[name])
		}
		cookieHeader = b.String()
	}

	return &Client{
		httpClient:   httpClient,
		token:        opts.Token,
		cookieHeader: cookieHeader,
		userAgent:    ua,
		minGap:       gap,
		authCookies:  opts.Cookies,
	}
}

// newHTTPClient builds the Reddit API http.Client. When proxyURL is set, every
// request tunnels through it via the Chrome-impersonating transport (same as
// Login) so post-login traffic egresses from the same residential IP as login.
func newHTTPClient(jar http.CookieJar, timeout time.Duration, proxyURL string) *http.Client {
	if proxyURL != "" {
		return impersonate.NewClient(impersonate.Options{ProxyURL: proxyURL}, jar, timeout)
	}
	return &http.Client{Timeout: timeout, Jar: jar}
}

// RateLimit returns a snapshot of the most recently observed rate-limit state.
// Use RateLimitState.IsLimited() to check if the client is currently throttled.
func (m *Client) RateLimit() RateLimitState {
	m.rlMu.Lock()
	defer m.rlMu.Unlock()
	return m.rlState
}

func (m *Client) doRequest(ctx context.Context, method, rawURL string, body io.Reader, contentType string) (*http.Response, error) {
	m.waitForGap(ctx)

	req, err := http.NewRequestWithContext(ctx, method, rawURL, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("User-Agent", m.userAgent)
	// www.reddit.com endpoints (poststats, signed-out feeds) reject
	// bearer-only requests with a login redirect — they need the
	// real cookie set instead. OAuth + matrix endpoints accept the
	// bearer alone, so we attach both opportunistically and let the
	// server pick. Stripping the bearer for www.reddit.com isn't
	// required (it just gets ignored), so we keep the code simple.
	if m.token != "" {
		req.Header.Set("Authorization", "Bearer "+m.token)
	}
	// Send session cookies to both www.reddit.com AND oauth.reddit.com.
	// Reddit's web-session token (minted via /svc/shreddit/token) is
	// session-backed: read GETs on oauth.reddit.com are accepted with the
	// bearer alone, but write POSTs (/api/comment, /api/submit, etc.) require
	// the session cookies alongside the bearer — the same behaviour as a
	// browser, which sends *.reddit.com cookies to every reddit subdomain.
	if m.cookieHeader != "" && (strings.HasPrefix(rawURL, wwwBaseURL) || strings.HasPrefix(rawURL, oauthBaseURL)) {
		req.Header.Set("Cookie", m.cookieHeader)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/json")
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request to %s: %w", rawURL, err)
	}

	m.updateRateLimit(resp.Header)

	if resp.StatusCode == http.StatusTooManyRequests {
		wait := parseRetryAfter(resp.Header.Get("Retry-After"), 60*time.Second)
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		m.rlMu.Lock()
		m.rlState.Remaining = 0
		m.rlState.RetryAfter = wait
		if m.rlState.Reset.IsZero() || time.Until(m.rlState.Reset) < wait {
			m.rlState.Reset = time.Now().Add(wait)
		}
		m.rlMu.Unlock()
		m.gapMu.Lock()
		if earliest := time.Now().Add(wait); m.lastReqAt.Before(earliest) {
			m.lastReqAt = earliest
		}
		m.gapMu.Unlock()
		return nil, fmt.Errorf("%w", ErrRateLimited)
	}

	return resp, nil
}

func (m *Client) oauthGet(path string) ([]byte, error) {
	resp, err := m.doRequest(context.Background(), "GET", oauthBaseURL+path, nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		// Only 401 means the bearer is expired/invalid. 403 is a resource-level
		// denial (private/banned subreddit, inaccessible rules) on a perfectly
		// valid token — re-minting the session won't help, so don't flag it.
		if resp.StatusCode == http.StatusUnauthorized {
			return nil, fmt.Errorf("unexpected status %d from %s: %s: %w", resp.StatusCode, path, truncate(string(body), 200), ErrUnauthorized)
		}
		return nil, fmt.Errorf("unexpected status %d from %s: %s", resp.StatusCode, path, truncate(string(body), 200))
	}

	return io.ReadAll(resp.Body)
}

func (m *Client) oauthGetJSON(path string, v interface{}) error {
	body, err := m.oauthGet(path)
	if err != nil {
		return err
	}
	if err := redditAPIBodyError(path, body); err != nil {
		return err
	}
	return json.Unmarshal(body, v)
}

// redditAPIBodyError detects Reddit's 200-with-error-body pattern, where the
// API returns HTTP 200 but the JSON payload carries a top-level "error" key
// (e.g. {"message":"Not Found","error":404}). Without this check, callers that
// unmarshal into a typed struct (e.g. a wrapper expecting a "data" key) silently
// get a zero-valued struct back instead of an explicit error.
func redditAPIBodyError(path string, body []byte) error {
	var probe struct {
		Error   json.RawMessage `json:"error"`
		Message string          `json:"message"`
	}
	if json.Unmarshal(body, &probe) == nil && len(probe.Error) > 0 && string(probe.Error) != "null" {
		return fmt.Errorf("reddit API error from %s: %s %s", path, probe.Error, probe.Message)
	}
	return nil
}

func (m *Client) oauthPost(path string, form url.Values) ([]byte, error) {
	resp, err := m.doRequest(context.Background(), "POST", oauthBaseURL+path, strings.NewReader(form.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status %d from POST %s: %s", resp.StatusCode, path, truncate(string(body), 200))
	}

	return body, nil
}

// webGet fetches a path from www.reddit.com — the user-facing site,
// not oauth.reddit.com. Used for endpoints that don't have an OAuth
// equivalent (currently just /poststats/{id}/, the post-insights HTML
// page). Sends both the bearer token and the full cookie set so the
// site recognizes the session as fully logged-in (bearer alone gets
// redirected to /login).
//
// Returns an error if Options.Cookies wasn't supplied — the page
// always 302s on bearer-only requests, so failing fast is friendlier
// than returning a login page as bytes.
//
// The Accept header is text/html so Reddit returns the rendered page
// (vs an SSE/JSON variant for clients that opt in).
func (m *Client) webGet(path string) ([]byte, error) {
	if m.cookieHeader == "" {
		return nil, fmt.Errorf("www.reddit.com request requires Options.Cookies (got empty)")
	}
	req, err := http.NewRequestWithContext(context.Background(), "GET", wwwBaseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", m.userAgent)
	req.Header.Set("Cookie", m.cookieHeader)
	if m.token != "" {
		req.Header.Set("Authorization", "Bearer "+m.token)
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9")
	// Match the rate-limit gap so we don't burst www.reddit.com — it
	// shares quota in practice with oauth.reddit.com.
	m.waitForGap(req.Context())
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request to %s: %w", path, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status %d from %s: %s", resp.StatusCode, path, truncate(string(body), 200))
	}
	// Reddit serves the login page with a 200 when the session looks
	// dead but the URL is reachable, so detect by content. The login
	// HTML always contains "Welcome to Reddit" in <title>.
	if b := body; len(b) < 200_000 && strings.Contains(string(b), "<title>Welcome to Reddit</title>") {
		return nil, fmt.Errorf("session not authenticated for %s (got login page; refresh cookies): %w", path, ErrUnauthorized)
	}
	return body, nil
}

// matrixStatusErr builds an error for a non-2xx Matrix response, wrapping
// ErrUnauthorized on a 401 or any body carrying the Matrix M_UNKNOWN_TOKEN
// error code (Reddit returns this once the bearer's chat session has lapsed).
func matrixStatusErr(status int, body []byte, what string) error {
	if status == http.StatusUnauthorized || bytes.Contains(body, []byte("M_UNKNOWN_TOKEN")) {
		return fmt.Errorf("matrix status %d from %s: %s: %w", status, what, truncate(string(body), 200), ErrUnauthorized)
	}
	return fmt.Errorf("matrix status %d from %s: %s", status, what, truncate(string(body), 200))
}

func (m *Client) matrixGet(path string) ([]byte, error) {
	resp, err := m.doRequest(context.Background(), "GET", matrixBaseURL+path, nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, matrixStatusErr(resp.StatusCode, body, path)
	}

	return io.ReadAll(resp.Body)
}

// HealthCheck verifies the bearer is live for Reddit CHAT (Matrix), not just
// REST. It hits the Matrix whoami endpoint and returns ErrUnauthorized when the
// chat session has lapsed (M_UNKNOWN_TOKEN) — the exact failure that leaves
// chat broken while oauth.reddit.com still accepts the same token. Hosts use
// this so connection status reflects chat auth.
func (m *Client) HealthCheck(ctx context.Context) error {
	resp, err := m.doRequest(ctx, "GET", matrixBaseURL+"/_matrix/client/r0/account/whoami", nil, "")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2000))
	return matrixStatusErr(resp.StatusCode, body, "/_matrix/client/r0/account/whoami")
}

func (m *Client) matrixGetJSON(path string, v interface{}) error {
	body, err := m.matrixGet(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, v)
}

func (m *Client) matrixPut(path string, payload interface{}) ([]byte, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshaling payload: %w", err)
	}

	resp, err := m.doRequest(context.Background(), "PUT", matrixBaseURL+path, strings.NewReader(string(data)), "application/json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, matrixStatusErr(resp.StatusCode, body, "PUT "+path)
	}

	return body, nil
}

func (m *Client) matrixPost(path string, payload interface{}) ([]byte, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshaling payload: %w", err)
	}

	resp, err := m.doRequest(context.Background(), "POST", matrixBaseURL+path, strings.NewReader(string(data)), "application/json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode/100 != 2 {
		return nil, matrixStatusErr(resp.StatusCode, body, "POST "+path)
	}

	return body, nil
}

// waitForGap uses a leaky-bucket reservation pattern with adaptive gap logic.
// The mutex is released before sleeping so concurrent callers each reserve
// their own slot and wait independently.
func (m *Client) waitForGap(ctx context.Context) {
	gap := m.adaptiveGap()

	m.gapMu.Lock()
	now := time.Now()
	next := m.lastReqAt.Add(gap)
	if now.After(next) {
		next = now
	}
	m.lastReqAt = next
	m.gapMu.Unlock()

	if wait := time.Until(next); wait > 0 {
		select {
		case <-ctx.Done():
		case <-time.After(wait):
		}
	}
	// Clear RetryAfter once we've waited past it.
	m.rlMu.Lock()
	m.rlState.RetryAfter = 0
	m.rlMu.Unlock()
}

// adaptiveGap returns the delay before the next request based on observed
// rate-limit state. Spreads requests across the window when quota is low;
// waits for reset when quota is exhausted.
func (m *Client) adaptiveGap() time.Duration {
	m.rlMu.Lock()
	rs := m.rlState
	m.rlMu.Unlock()

	// Quota exhausted — wait for the window to reset.
	if rs.Remaining == 0 && !rs.Reset.IsZero() {
		if d := time.Until(rs.Reset); d > 0 {
			return d + 50*time.Millisecond
		}
	}
	// Spread remaining quota evenly across the reset window (90% safety margin).
	if rs.Remaining > 0 && !rs.Reset.IsZero() {
		if d := time.Until(rs.Reset); d > 0 {
			spread := d / time.Duration(float64(rs.Remaining)*0.9)
			if spread > m.minGap {
				return spread
			}
		}
	}
	return m.minGap
}

// updateRateLimit reads standard rate-limit headers from a response and updates
// the client's tracked state. Call on every HTTP response.
func (m *Client) updateRateLimit(h http.Header) {
	m.rlMu.Lock()
	defer m.rlMu.Unlock()
	if v := rlHeader(h, "Limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			m.rlState.Limit = n
		}
	}
	if v := rlHeader(h, "Remaining"); v != "" {
		// Reddit sends X-Ratelimit-Remaining as a float like "99.0".
		if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
			m.rlState.Remaining = int(f)
		}
	}
	if v := rlHeader(h, "Reset"); v != "" {
		if ts, err := strconv.ParseInt(v, 10, 64); err == nil {
			if ts > 1_000_000_000 {
				m.rlState.Reset = time.Unix(ts, 0) // Unix epoch
			} else {
				m.rlState.Reset = time.Now().Add(time.Duration(ts) * time.Second) // relative seconds (Reddit style)
			}
		}
	}
}

// rlHeader returns the trimmed value of a rate-limit header, checking the four
// most common prefix variants.
func rlHeader(h http.Header, suffix string) string {
	for _, p := range []string{"X-RateLimit-", "X-Rate-Limit-", "X-Ratelimit-", "RateLimit-"} {
		if v := strings.TrimSpace(h.Get(p + suffix)); v != "" {
			return v
		}
	}
	return ""
}

// parseRetryAfter parses rate-limit headers. Handles three formats:
// - Seconds integer (Retry-After: 60)
// - Unix epoch timestamp (X-Rate-Limit-Reset: 1716000000)
// - HTTP-date (Retry-After: Mon, 01 Jan 2024 00:00:00 GMT)
func parseRetryAfter(val string, fallback time.Duration) time.Duration {
	if val == "" {
		return fallback
	}
	trimmed := strings.TrimSpace(val)
	if n, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
		if n > 1_000_000_000 {
			if d := time.Until(time.Unix(n, 0)); d > 0 {
				return d
			}
			return fallback
		}
		return time.Duration(n) * time.Second
	}
	if t, err := http.ParseTime(trimmed); err == nil {
		if d := time.Until(t); d > 0 {
			return d
		}
	}
	return fallback
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
