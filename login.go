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
	"os"
	"regexp"
	"strings"

	impersonate "github.com/teslashibe/impersonate-go"
)

// Credential login for Reddit.
//
// The OAuth client (client.go) drives oauth.reddit.com with a Bearer token
// (the token_v2 / web access token). The Social OS flow logs in with a
// username + password instead of pasting that token: we establish a web
// session via the legacy login endpoint, then read the freshly-minted bearer
// access token off the logged-in www.reddit.com page. Both the bearer and the
// session cookies are returned so New can drive oauth.reddit.com AND the
// www.reddit.com endpoints (poststats) that reject bearer-only auth.

const (
	loginUserAgent = defaultUserAgent
	oldBaseURL     = "https://old.reddit.com"
)

var (
	reLoginErrEsc = regexp.MustCompile(`"errors":\s*\[\s*\[\s*"([A-Z_]+)"`)
	reCSRFToken   = regexp.MustCompile(`name="csrf_token"\s+value="([0-9a-f]+)"`)
)

// LoginParams holds Reddit credentials. OTP is optional and only needed when
// the account has 2FA enabled (TOTP code or a code Reddit sent).
type LoginParams struct {
	Username string
	Password string
	OTP      string
	// UserAgent overrides the browser UA used for the login + minted client.
	UserAgent string
	// ProxyURL, when set, routes the login through an HTTP/S proxy (e.g. a
	// Webshare residential gateway). Reddit's edge blocks datacenter IPs and
	// non-browser requests, so a residential egress is often required.
	ProxyURL string

	// RecaptchaToken is a reCAPTCHA Enterprise token (site key RecaptchaSiteKey,
	// action "login"). Reddit's /svc/shreddit/account/login endpoint rejects any
	// POST without one (generic HTTP 400 "Something went wrong"), verified by
	// reverse-engineering the live faceplate-form. The token is browser-bound
	// and cannot be minted in pure Go, so the caller supplies it directly or via
	// RecaptchaProvider (e.g. a headless-browser sidecar, mirroring the TikTok
	// X-Gnarly signer in #259). If empty and RecaptchaProvider is nil, Login
	// returns ErrRecaptchaRequired.
	RecaptchaToken string
	// RecaptchaProvider mints a reCAPTCHA Enterprise token on demand when
	// RecaptchaToken is empty. siteKey is RecaptchaSiteKey; action is "login".
	RecaptchaProvider func(ctx context.Context, siteKey, action string) (string, error)
}

// RecaptchaSiteKey is Reddit's public reCAPTCHA Enterprise site key, scraped
// from the live login page (google.com/recaptcha/enterprise.js?render=...).
const RecaptchaSiteKey = "6LfirrMoAAAAAHZOipvza4kpp_VtTwLNuXVwURNQ"

// LoginResult is the minted session, suitable for passing straight into New
// via Options{Token, Cookies, UserAgent}.
type LoginResult struct {
	Token     string
	Cookies   map[string]string
	UserAgent string
}

// Login authenticates with a username + password and returns a minted session
// (bearer + cookies). It does not require a pre-existing token.
//
// 2FA: when the account has 2FA enabled, set OTP. Reddit's legacy endpoint
// takes the second factor as "password:otp"; Login assembles that when OTP is
// set. A missing/!needed OTP surfaces ErrTwoFactorRequired so the caller can
// prompt for a code and retry.
func Login(ctx context.Context, p LoginParams) (*LoginResult, error) {
	username := strings.TrimSpace(p.Username)
	if username == "" || p.Password == "" {
		return nil, fmt.Errorf("reddit: username and password are required")
	}
	ua := p.UserAgent
	if ua == "" {
		ua = loginUserAgent
	}

	jar, _ := cookiejar.New(nil)
	// Reddit's login edge fingerprints the TLS ClientHello (JA3) and rejects
	// Go's default stack with a 403 block page. Present Chrome's ClientHello
	// via the shared impersonate transport so the login looks like a browser.
	hc := impersonate.NewClient(impersonate.Options{}, jar, defaultTimeout)

	// Reddit gates login behind reCAPTCHA Enterprise: every POST to
	// /svc/shreddit/account/login must carry a recaptcha_token or the edge
	// returns a generic HTTP 400. The token is browser-bound (Google's
	// enterprise.js fingerprints the JS environment) so it cannot be minted in
	// pure Go; resolve it from the caller or a sidecar before priming.
	recaptcha := strings.TrimSpace(p.RecaptchaToken)
	if recaptcha == "" {
		if p.RecaptchaProvider == nil {
			return nil, ErrRecaptchaRequired
		}
		tok, err := p.RecaptchaProvider(ctx, RecaptchaSiteKey, "login")
		if err != nil {
			return nil, fmt.Errorf("reddit: mint recaptcha token: %w", err)
		}
		recaptcha = tok
		if strings.TrimSpace(recaptcha) == "" {
			return nil, ErrRecaptchaRequired
		}
	}

	// Load the login page to seed cookies and read the csrf_token the modern
	// (shreddit) login endpoint requires.
	csrf, err := primeSession(ctx, hc, ua)
	if err != nil {
		return nil, fmt.Errorf("reddit: prime session: %w", err)
	}

	// Match the real shreddit web request byte-for-byte (reverse-engineered
	// from the live faceplate-form via browser dev-tools): username, password,
	// otp (empty unless 2FA), dest, csrf_token, recaptcha_token; URL-encoded,
	// partial-html Accept header. A request missing recaptcha_token returns 400;
	// a complete one returns 200 and sets the reddit_session cookie (verified
	// live: authenticated as the form's account).
	form := url.Values{
		"username":        {username},
		"password":        {p.Password},
		"otp":             {strings.TrimSpace(p.OTP)},
		"dest":            {wwwBaseURL},
		"csrf_token":      {csrf},
		"recaptcha_token": {recaptcha},
	}
	loginURL := wwwBaseURL + "/svc/shreddit/account/login"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, loginURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("reddit: build login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Origin", wwwBaseURL)
	req.Header.Set("Referer", wwwBaseURL+"/login/")
	req.Header.Set("Accept", "text/vnd.reddit.partial+html, application/json")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Dest", "empty")

	resp, err := hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("reddit: login request: %w", err)
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	resp.Body.Close()
	raw := string(body)

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("reddit: login rate limited (HTTP 429)")
	}
	if dbg := strings.TrimSpace(os.Getenv("REDDIT_LOGIN_DEBUG")); dbg != "" {
		snippet := raw
		if len(snippet) > 400 {
			snippet = snippet[:400]
		}
		fmt.Fprintf(os.Stderr, "[reddit-login-debug] status=%d body=%s\n", resp.StatusCode, snippet)
	}
	if m := reLoginErrEsc.FindStringSubmatch(raw); m != nil {
		switch m[1] {
		case "WRONG_PASSWORD":
			return nil, fmt.Errorf("reddit: wrong username or password")
		case "INCORRECT_USERNAME_PASSWORD":
			return nil, fmt.Errorf("reddit: wrong username or password")
		case "RATELIMIT":
			return nil, fmt.Errorf("reddit: login rate limited (try again later)")
		default:
			// 2FA / one-time code variants surface here.
			if strings.Contains(raw, "OTP") || strings.Contains(strings.ToLower(raw), "two") {
				return nil, ErrTwoFactorRequired
			}
			return nil, fmt.Errorf("reddit: login failed (%s)", m[1])
		}
	}

	// Session cookie must be set on success.
	cookies := collectCookies(jar)
	if cookies["reddit_session"] == "" {
		// If errors were empty but no session, the account may need 2FA.
		low := strings.ToLower(raw)
		if strings.Contains(low, "two-factor") || strings.Contains(low, "2fa") || strings.Contains(low, "verification code") {
			return nil, ErrTwoFactorRequired
		}
		if strings.Contains(low, "incorrect username or password") || strings.Contains(low, "invalid") {
			return nil, fmt.Errorf("reddit: wrong username or password")
		}
		return nil, fmt.Errorf("reddit: login did not establish a session (status %d)", resp.StatusCode)
	}

	if dbg := strings.TrimSpace(os.Getenv("REDDIT_LOGIN_DEBUG")); dbg != "" {
		names := make([]string, 0, len(cookies))
		for k := range cookies {
			names = append(names, k)
		}
		fmt.Fprintf(os.Stderr, "[reddit-login-debug] post-login cookies: %v\n", names)
	}

	// Mint the OAuth bearer. The session cookies authenticate www.reddit.com,
	// but reddit-go's API methods hit oauth.reddit.com, which needs a real
	// Bearer (the token_v2 cookie is NOT an oauth bearer). The shreddit web app
	// mints the bearer via POST /svc/shreddit/token {csrf_token} -> {token,
	// expires}; we replicate that exact call (reverse-engineered from the
	// bundle) rather than scraping HTML.
	token, err := fetchAccessToken(ctx, hc, ua, csrf)
	if err != nil {
		return nil, fmt.Errorf("reddit: mint access token: %w", err)
	}

	// Re-snapshot cookies so the www.reddit.com (poststats) path keeps a
	// complete jar too.
	cookies = collectCookies(jar)

	return &LoginResult{Token: token, Cookies: cookies, UserAgent: ua}, nil
}

// ErrTwoFactorRequired is returned by Login when the account has 2FA enabled
// and no (or an incorrect) OTP was supplied. Retry Login with LoginParams.OTP.
var ErrTwoFactorRequired = errTwoFactor

// primeSession loads the login page (seeding cookies) and returns the
// csrf_token the shreddit login endpoint requires. www.reddit.com fronts the
// page with a JavaScript "please wait for verification" interstitial; we solve
// it (see challenge.go) so the real login page — and its csrf_token cookie —
// is served.
func primeSession(ctx context.Context, hc *http.Client, ua string) (string, error) {
	debug := strings.TrimSpace(os.Getenv("REDDIT_LOGIN_DEBUG")) != ""

	pageURL := wwwBaseURL + "/login/"
	body, err := getPage(ctx, hc, pageURL, ua)
	if err != nil {
		return "", err
	}

	// Clear up to a few rounds of the JS interstitial before giving up.
	for i := 0; i < 3 && isJSChallenge(body); i++ {
		nextURL, serr := solveJSChallenge(pageURL, body)
		if serr != nil {
			return "", fmt.Errorf("solve verification challenge: %w", serr)
		}
		if debug {
			fmt.Fprintf(os.Stderr, "[reddit-login-debug] solved JS challenge round %d -> %s\n", i+1, nextURL)
		}
		pageURL = nextURL
		if body, err = getPage(ctx, hc, pageURL, ua); err != nil {
			return "", err
		}
	}
	if isJSChallenge(body) {
		return "", fmt.Errorf("verification challenge not cleared after retries")
	}

	// The modern login page delivers csrf_token as a cookie (not embedded in
	// the HTML). Prefer the cookie; keep the legacy HTML scrape as a fallback.
	if u, err := url.Parse(wwwBaseURL); err == nil {
		for _, ck := range hc.Jar.Cookies(u) {
			if ck.Name == "csrf_token" && ck.Value != "" {
				return ck.Value, nil
			}
		}
	}
	if m := reCSRFToken.FindStringSubmatch(body); m != nil {
		return m[1], nil
	}
	if debug {
		fmt.Fprintf(os.Stderr, "[reddit-login-debug] no csrf cookie or input (body len=%d)\n", len(body))
	}
	return "", fmt.Errorf("csrf_token not found after clearing interstitial")
}

// getPage issues a browser-like GET and returns the response body.
func getPage(ctx context.Context, hc *http.Client, target, ua string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	resp, err := hc.Do(req)
	if err != nil {
		return "", err
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
	resp.Body.Close()
	return string(body), nil
}

// fetchAccessToken mints the OAuth bearer the way the shreddit web app does:
// POST /svc/shreddit/token with the csrf_token, which returns {token, expires}
// for an authenticated session. Reverse-engineered from the shreddit bundle:
//
//	fetch("/svc/shreddit/token",{method:"POST",
//	  headers:{Accept:"application/json","Content-Type":"application/json"},
//	  body:JSON.stringify({csrf_token:csrf})})
func fetchAccessToken(ctx context.Context, hc *http.Client, ua, csrf string) (string, error) {
	payload, _ := json.Marshal(map[string]string{"csrf_token": csrf})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, wwwBaseURL+"/svc/shreddit/token", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", wwwBaseURL)
	req.Header.Set("Referer", wwwBaseURL+"/")
	resp, err := hc.Do(req)
	if err != nil {
		return "", err
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token endpoint status %d (session may not be logged in)", resp.StatusCode)
	}
	var out struct {
		Token   string `json:"token"`
		Expires int64  `json:"expires"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}
	if out.Token == "" {
		return "", fmt.Errorf("token endpoint returned empty token")
	}
	return out.Token, nil
}

func collectCookies(jar http.CookieJar) map[string]string {
	out := map[string]string{}
	for _, host := range []string{wwwBaseURL, "https://reddit.com", oauthBaseURL} {
		u, err := url.Parse(host)
		if err != nil {
			continue
		}
		for _, ck := range jar.Cookies(u) {
			if ck.Value != "" {
				out[ck.Name] = ck.Value
			}
		}
	}
	return out
}
