package reddit

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// Unit tests in this file run without REDDIT_TOKEN — they exercise the
// pure parsing/normalization helpers added alongside the post-analytics
// MCP tools. Integration tests for the live API live in reddit_test.go
// and are gated by REDDIT_TOKEN.

func TestNormalizePostFullname(t *testing.T) {
	cases := map[string]string{
		"abc123":       "t3_abc123",
		"t3_abc123":    "t3_abc123",
		"t1_xyz":       "t1_xyz", // already a comment fullname — leave it
		"  abc  ":      "t3_abc",
		"t3_":          "t3_", // edge: trust the caller, don't double-prefix
	}
	for in, want := range cases {
		if got := normalizePostFullname(in); got != want {
			t.Errorf("normalizePostFullname(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestReplyJSONErrors(t *testing.T) {
	// replyResult mirrors the struct inside Client.Reply so we can test the
	// json.errors branch without a live HTTP round-trip.
	type replyResult struct {
		JSON struct {
			Errors [][]string `json:"errors"`
			Data   struct {
				Things []struct {
					Data struct{ ID string `json:"id"` } `json:"data"`
				} `json:"things"`
			} `json:"data"`
		} `json:"json"`
	}

	parse := func(body string) (string, error) {
		var result replyResult
		if err := json.Unmarshal([]byte(body), &result); err != nil {
			return "", err
		}
		if len(result.JSON.Errors) > 0 {
			e := result.JSON.Errors[0]
			msg := "reddit error posting comment"
			if len(e) >= 2 {
				msg = fmt.Sprintf("reddit error: %s — %s", e[0], e[1])
			} else if len(e) == 1 {
				msg = fmt.Sprintf("reddit error: %s", e[0])
			}
			return "", fmt.Errorf("%s", msg)
		}
		if len(result.JSON.Data.Things) == 0 {
			return "", fmt.Errorf("no comment returned in response")
		}
		return result.JSON.Data.Things[0].Data.ID, nil
	}

	cases := []struct {
		name    string
		body    string
		wantID  string
		wantErr string
	}{
		{
			name:   "success",
			body:   `{"json":{"errors":[],"data":{"things":[{"kind":"t1","data":{"id":"abc123"}}]}}}`,
			wantID: "abc123",
		},
		{
			name:    "quota filled",
			body:    `{"json":{"errors":[["QUOTA_FILLED","you are doing that too much. try again in 8 minutes.","ratelimit"]],"data":{}}}`,
			wantErr: "QUOTA_FILLED",
		},
		{
			name:    "ratelimit",
			body:    `{"json":{"errors":[["RATELIMIT","take a break","ratelimit"]],"data":{}}}`,
			wantErr: "RATELIMIT",
		},
		{
			name:    "single-element error",
			body:    `{"json":{"errors":[["SUBREDDIT_NOEXIST"]],"data":{}}}`,
			wantErr: "SUBREDDIT_NOEXIST",
		},
		{
			name:    "empty things no errors",
			body:    `{"json":{"errors":[],"data":{"things":[]}}}`,
			wantErr: "no comment returned in response",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			id, err := parse(tc.body)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("error %q does not contain %q", err.Error(), tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if id != tc.wantID {
				t.Errorf("id = %q, want %q", id, tc.wantID)
			}
		})
	}
}

func TestRawSubredditRulesToSubredditRules(t *testing.T) {
	body := []byte(`{
		"rules": [
			{
				"kind": "all",
				"short_name": "Be civil",
				"description": "No personal attacks.",
				"violation_reason": "incivility",
				"priority": 0,
				"created_utc": 1600000000
			},
			{
				"kind": "link",
				"short_name": "No surveys",
				"description": "Surveys belong in r/SampleSize.",
				"priority": 1,
				"created_utc": 1610000000
			}
		],
		"site_rules": ["No personal info", "No spam"],
		"site_rules_flow": [{"reasonTextToShow": "ignored"}]
	}`)
	var raw rawSubredditRules
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	got := raw.toSubredditRules()
	if len(got.Rules) != 2 {
		t.Fatalf("want 2 rules, got %d", len(got.Rules))
	}
	if got.Rules[0].ShortName != "Be civil" || got.Rules[0].Kind != "all" {
		t.Errorf("rule[0] = %+v", got.Rules[0])
	}
	if got.Rules[1].ShortName != "No surveys" || got.Rules[1].Kind != "link" {
		t.Errorf("rule[1] = %+v", got.Rules[1])
	}
	if !reflect.DeepEqual(got.SiteRules, []string{"No personal info", "No spam"}) {
		t.Errorf("site_rules drift: %v", got.SiteRules)
	}
	// site_rules_flow is intentionally not surfaced — guard against
	// accidental future inclusion.
	if got.Rules[0].Created.IsZero() {
		t.Errorf("created_utc not parsed")
	}
}

func TestFlattenCommentsHandlesNestedReplies(t *testing.T) {
	// Mimics the shape Reddit returns for /comments/{id}: a parent
	// comment with one nested reply. The nested reply lives in
	// `replies` as a redditListing wrapper.
	body := []byte(`[
		{
			"kind": "t1",
			"data": {
				"id": "p",
				"name": "t1_p",
				"author": "alice",
				"body": "parent",
				"score": 5,
				"created_utc": 1700000000,
				"replies": {
					"kind": "Listing",
					"data": {
						"after": null,
						"before": null,
						"children": [
							{
								"kind": "t1",
								"data": {
									"id": "c",
									"name": "t1_c",
									"author": "bob",
									"body": "child",
									"score": 2,
									"created_utc": 1700000100,
									"replies": ""
								}
							},
							{
								"kind": "more",
								"data": {"id": "_", "children": []}
							}
						]
					}
				}
			}
		}
	]`)
	var children []redditChild
	if err := json.Unmarshal(body, &children); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	var out []Comment
	flattenComments(children, &out)
	if len(out) != 2 {
		t.Fatalf("want 2 comments (parent + child), got %d", len(out))
	}
	if out[0].ID != "p" || out[1].ID != "c" {
		t.Errorf("order drift: %+v", out)
	}
	if out[1].Body != "child" {
		t.Errorf("nested body lost: %q", out[1].Body)
	}
}

func TestRawPostViewCountIsPointer(t *testing.T) {
	// Reddit returns null for view_count on posts the authenticated
	// user did not author. Make sure that round-trips as a nil pointer
	// rather than silently becoming 0.
	body := []byte(`{
		"id": "abc",
		"name": "t3_abc",
		"score": 10,
		"view_count": null
	}`)
	var rp rawPost
	if err := json.Unmarshal(body, &rp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if rp.ViewCount != nil {
		t.Errorf("expected nil view_count, got %d", *rp.ViewCount)
	}

	body = []byte(`{"id":"abc","name":"t3_abc","score":10,"view_count":42}`)
	if err := json.Unmarshal(body, &rp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if rp.ViewCount == nil || *rp.ViewCount != 42 {
		t.Errorf("expected 42 view_count, got %v", rp.ViewCount)
	}
	post := rp.toPost()
	if post.ViewCount == nil || *post.ViewCount != 42 {
		t.Errorf("toPost dropped view_count: %v", post.ViewCount)
	}
}
