// Command reddit-login-probe smoke-tests Reddit username/password login end to
// end: it logs in, mints a bearer, builds a client, and calls Me() for
// liveness. Mirrors titlepro247-login-probe. Credentials come from env:
//
//	REDDIT_USERNAME, REDDIT_PASSWORD, REDDIT_OTP (optional 2FA code)
//
// Exit 0 on success (prints the authenticated username), non-zero otherwise.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	reddit "github.com/teslashibe/reddit-go"
)

// recaptchaSidecar POSTs {site_key, action} to a token-minting sidecar and
// returns the token field. The sidecar runs a headless browser (or equivalent)
// to execute Google's enterprise.js, which pure Go cannot.
func recaptchaSidecar(ctx context.Context, endpoint, siteKey, action string) (string, error) {
	payload, _ := json.Marshal(map[string]string{"site_key": siteKey, "action": action})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("recaptcha sidecar status %d", resp.StatusCode)
	}
	var out struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	return out.Token, nil
}

func main() {
	user := strings.TrimSpace(os.Getenv("REDDIT_USERNAME"))
	pass := os.Getenv("REDDIT_PASSWORD")
	otp := strings.TrimSpace(os.Getenv("REDDIT_OTP"))
	proxy := strings.TrimSpace(os.Getenv("REDDIT_PROXY_URL"))
	// reCAPTCHA Enterprise token. Reddit's login endpoint requires one; it is
	// browser-bound and cannot be minted in pure Go. Supply a pre-minted token
	// (REDDIT_RECAPTCHA_TOKEN) or a sidecar URL (REDDIT_RECAPTCHA_URL) that
	// returns {"token":"..."} for {site_key, action} — mirrors the TikTok
	// X-Gnarly signer sidecar (#259).
	recaptcha := strings.TrimSpace(os.Getenv("REDDIT_RECAPTCHA_TOKEN"))
	sidecar := strings.TrimSpace(os.Getenv("REDDIT_RECAPTCHA_URL"))
	if user == "" || pass == "" {
		fmt.Fprintln(os.Stderr, "reddit-login-probe: set REDDIT_USERNAME and REDDIT_PASSWORD")
		os.Exit(2)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	params := reddit.LoginParams{Username: user, Password: pass, OTP: otp, ProxyURL: proxy, RecaptchaToken: recaptcha}
	if recaptcha == "" && sidecar != "" {
		params.RecaptchaProvider = func(ctx context.Context, siteKey, action string) (string, error) {
			return recaptchaSidecar(ctx, sidecar, siteKey, action)
		}
	}

	res, err := reddit.Login(ctx, params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "login: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("login ok: token_len=%d cookies=%d\n", len(res.Token), len(res.Cookies))

	// Liveness: the reverse-engineered web session authenticates the cookie
	// jar, so verify against the same /api/me.json endpoint shreddit calls.
	name, err := cookieLiveness(ctx, res.Cookies, res.UserAgent)
	if err != nil {
		fmt.Fprintf(os.Stderr, "liveness: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("liveness ok: u/%s\n", name)

	// Best-effort: confirm the OAuth bearer path too (non-fatal if the web
	// token_v2 isn't accepted verbatim by oauth.reddit.com).
	c := reddit.New(&reddit.Options{Token: res.Token, Cookies: res.Cookies, UserAgent: res.UserAgent})
	if me, err := c.Me(); err != nil {
		fmt.Fprintf(os.Stderr, "note: oauth Me() failed (bearer path): %v\n", err)
	} else if me.Name == "" {
		fmt.Fprintln(os.Stderr, "note: oauth Me() returned empty identity (token_v2 not an oauth bearer)")
	} else {
		fmt.Printf("oauth liveness ok: u/%s (id=%s)\n", me.Name, me.ID)
	}
}

func cookieLiveness(ctx context.Context, cookies map[string]string, ua string) (string, error) {
	var b strings.Builder
	for k, v := range cookies {
		if b.Len() > 0 {
			b.WriteString("; ")
		}
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(v)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.reddit.com/api/me.json", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Cookie", b.String())
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("me.json status %d", resp.StatusCode)
	}
	var out struct {
		Data struct {
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if out.Data.Name == "" {
		return "", fmt.Errorf("me.json returned no user (session not authenticated)")
	}
	return out.Data.Name, nil
}
