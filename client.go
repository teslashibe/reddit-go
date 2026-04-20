package reddit

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	oauthBaseURL     = "https://oauth.reddit.com"
	matrixBaseURL    = "https://matrix.redditspace.com"
	defaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
	defaultTimeout   = 30 * time.Second
)

// Options configures the messenger client. Token is required.
type Options struct {
	// Token is the token_v2 cookie value from a logged-in Reddit session.
	// Get it from browser DevTools → Application → Cookies → reddit.com → token_v2.
	Token string

	// UserAgent overrides the default browser-like User-Agent string.
	UserAgent string

	// RequestTimeout overrides the default 30s HTTP timeout.
	RequestTimeout time.Duration

	// MinRequestGap sets the minimum interval between requests (default 650ms).
	MinRequestGap time.Duration
}

// Client is the main Reddit API client.
type Client struct {
	httpClient *http.Client
	token      string
	userAgent  string

	mu            sync.Mutex
	remaining     float64
	resetAt       time.Time
	minRequestGap time.Duration
	lastRequestAt time.Time
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

	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
			Jar:     jar,
		},
		token:         opts.Token,
		userAgent:     ua,
		remaining:     100,
		minRequestGap: gap,
	}
}

func (m *Client) doRequest(method, rawURL string, body io.Reader, contentType string) (*http.Response, error) {
	m.waitForRateLimit()

	req, err := http.NewRequest(method, rawURL, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("User-Agent", m.userAgent)
	if m.token != "" {
		req.Header.Set("Authorization", "Bearer "+m.token)
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

	m.updateRateLimits(resp.Header)
	return resp, nil
}

func (m *Client) oauthGet(path string) ([]byte, error) {
	resp, err := m.doRequest("GET", oauthBaseURL+path, nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d from %s: %s", resp.StatusCode, path, truncate(string(body), 200))
	}

	return io.ReadAll(resp.Body)
}

func (m *Client) oauthGetJSON(path string, v interface{}) error {
	body, err := m.oauthGet(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, v)
}

func (m *Client) oauthPost(path string, form url.Values) ([]byte, error) {
	resp, err := m.doRequest("POST", oauthBaseURL+path, strings.NewReader(form.Encode()), "application/x-www-form-urlencoded")
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

func (m *Client) matrixGet(path string) ([]byte, error) {
	resp, err := m.doRequest("GET", matrixBaseURL+path, nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("matrix status %d from %s: %s", resp.StatusCode, path, truncate(string(body), 200))
	}

	return io.ReadAll(resp.Body)
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

	resp, err := m.doRequest("PUT", matrixBaseURL+path, strings.NewReader(string(data)), "application/json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("matrix status %d from PUT %s: %s", resp.StatusCode, path, truncate(string(body), 200))
	}

	return body, nil
}

func (m *Client) matrixPost(path string, payload interface{}) ([]byte, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshaling payload: %w", err)
	}

	resp, err := m.doRequest("POST", matrixBaseURL+path, strings.NewReader(string(data)), "application/json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("matrix status %d from POST %s: %s", resp.StatusCode, path, truncate(string(body), 200))
	}

	return body, nil
}

// waitForRateLimit uses a leaky-bucket reservation pattern.
// The mutex is released before sleeping so concurrent callers
// each reserve their own slot and wait independently.
func (m *Client) waitForRateLimit() {
	m.mu.Lock()

	now := time.Now()
	nextSlot := m.lastRequestAt.Add(m.minRequestGap)
	if now.After(nextSlot) {
		nextSlot = now
	}
	if m.remaining <= 2 && m.resetAt.After(nextSlot) {
		nextSlot = m.resetAt.Add(time.Second)
	}

	m.lastRequestAt = nextSlot
	m.mu.Unlock()

	if wait := time.Until(nextSlot); wait > 0 {
		time.Sleep(wait)
	}
}

func (m *Client) updateRateLimits(h http.Header) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if v := h.Get("X-Ratelimit-Remaining"); v != "" {
		if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
			m.remaining = f
		}
	}
	if v := h.Get("X-Ratelimit-Reset"); v != "" {
		if secs, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			m.resetAt = time.Now().Add(time.Duration(secs) * time.Second)
		}
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
