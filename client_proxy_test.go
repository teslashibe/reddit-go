package reddit

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestNewFromLoginPropagatesProxyURL(t *testing.T) {
	bare := New(&Options{Token: "t"})
	fromLogin := NewFromLogin(&LoginResult{
		Token:    "t",
		ProxyURL: "http://user:pass@proxy.test:80",
	})
	if fromLogin.httpClient.Transport == bare.httpClient.Transport {
		t.Fatal("NewFromLogin did not apply ProxyURL to client transport")
	}
}

func TestClientProxyEgress(t *testing.T) {
	proxyURL := strings.TrimSpace(os.Getenv("REDDIT_PROXY_URL"))
	if proxyURL == "" {
		t.Skip("REDDIT_PROXY_URL not set")
	}

	directResp, err := http.Get("https://api.ipify.org?format=text")
	if err != nil {
		t.Fatalf("direct ipify: %v", err)
	}
	directBody, _ := io.ReadAll(directResp.Body)
	directResp.Body.Close()
	directIP := strings.TrimSpace(string(directBody))

	c := New(&Options{Token: "test-token", ProxyURL: proxyURL})
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://api.ipify.org?format=text", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		t.Fatalf("Do via proxy: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	proxyIP := strings.TrimSpace(string(body))
	if proxyIP == "" {
		t.Fatal("empty egress IP from proxy client")
	}
	if proxyIP == directIP {
		t.Fatalf("proxy client egress IP %q matches direct datacenter IP %q", proxyIP, directIP)
	}
	t.Logf("direct IP=%s proxy egress IP=%s", directIP, proxyIP)
}
