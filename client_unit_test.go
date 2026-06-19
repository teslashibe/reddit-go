package reddit

import "testing"

// Unit tests for pure client helpers — no REDDIT_TOKEN required.

func TestRedditAPIBodyError(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{"normal data response", `{"data":{"id":"abc","display_name":"golang"}}`, false},
		{"listing response", `{"kind":"Listing","data":{"children":[]}}`, false},
		{"no error key", `{"rules":[],"site_rules":[]}`, false},
		{"null error key", `{"error":null,"message":""}`, false},
		{"integer error code", `{"message":"Not Found","error":404}`, true},
		{"string error code", `{"message":"Forbidden","error":"forbidden"}`, true},
		{"rate-limit error body", `{"message":"Too Many Requests","error":429}`, true},
		{"empty body", `{}`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := redditAPIBodyError("/r/test/about.json", []byte(tc.body))
			if (err != nil) != tc.wantErr {
				t.Errorf("redditAPIBodyError() = %v, wantErr=%v", err, tc.wantErr)
			}
		})
	}
}
