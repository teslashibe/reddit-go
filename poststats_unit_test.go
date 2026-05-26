package reddit

import (
	"os"
	"strings"
	"testing"
)

// TestParsePostInsightsHTML exercises the scraper against a real
// /poststats/{id}/ HTML response captured from r/whoop on
// 2026-04-22. The fixture has every section populated (post is
// >24h old so the hourly chart is full, has 76 comments so the top
// comments section has its full 3 entries, etc.) so this is a good
// regression smoke for any future Reddit theme refresh.
func TestParsePostInsightsHTML(t *testing.T) {
	body, err := os.ReadFile("testdata/poststats_1sf4xjz.html")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	out := &PostInsights{PostID: "1sf4xjz"}
	parsePostInsightsHTML(string(body), out)

	wantTitle := "We analyzed 2,169 posts from this sub. The #1 complaint isn't price. It's that Whoop won't tell you what to do with your data."
	if out.Title != wantTitle {
		t.Errorf("Title:\n  got: %q\n  want: %q", out.Title, wantTitle)
	}
	if out.Subreddit != "whoop" {
		t.Errorf("Subreddit: got %q, want %q", out.Subreddit, "whoop")
	}
	if !strings.HasPrefix(out.Permalink, "https://www.reddit.com/r/whoop/comments/1sf4xjz/") {
		t.Errorf("Permalink: got %q, want https://www.reddit.com/r/whoop/comments/1sf4xjz/...", out.Permalink)
	}
	if !strings.Contains(out.PersonalComparison, "#1 post of all time") {
		t.Errorf("PersonalComparison: got %q, want contains '#1 post of all time'", out.PersonalComparison)
	}

	if out.TotalViewsFormatted != "51K" {
		t.Errorf("TotalViewsFormatted: got %q, want %q", out.TotalViewsFormatted, "51K")
	}
	// Hourly sum should be in the same order of magnitude as the
	// formatted total (51K). Slight under-count is OK because the
	// chart caps at 48 hours.
	if out.TotalViews < 30_000 || out.TotalViews > 70_000 {
		t.Errorf("TotalViews from hourly sum: got %d, want in [30000, 70000]", out.TotalViews)
	}
	if len(out.HourlyViews) < 24 {
		t.Errorf("HourlyViews: got %d entries, want >= 24", len(out.HourlyViews))
	}
	// Hour 1 should exist and match the first segment in the
	// fixture (837 views).
	if len(out.HourlyViews) > 0 {
		first := out.HourlyViews[0]
		if first.HourOffset != 1 || first.Views != 837 {
			t.Errorf("HourlyViews[0]: got %+v, want {HourOffset:1 Views:837}", first)
		}
	}
	if out.ViewsChangeFormatted != "+74" {
		t.Errorf("ViewsChangeFormatted: got %q, want %q", out.ViewsChangeFormatted, "+74")
	}
	if out.ViewsChange != 74 {
		t.Errorf("ViewsChange: got %d, want 74", out.ViewsChange)
	}

	if out.Upvotes != 116 {
		t.Errorf("Upvotes: got %d, want 116", out.Upvotes)
	}
	if out.UpvoteRatio < 0.86 || out.UpvoteRatio > 0.89 {
		t.Errorf("UpvoteRatio: got %.3f, want ~0.877", out.UpvoteRatio)
	}
	if out.Comments != 76 {
		t.Errorf("Comments: got %d, want 76", out.Comments)
	}
	if out.Shares != 63 {
		t.Errorf("Shares: got %d, want 63", out.Shares)
	}
	if out.Crossposts != 1 {
		t.Errorf("Crossposts: got %d, want 1", out.Crossposts)
	}
	if out.Awards != 0 {
		t.Errorf("Awards: got %d, want 0", out.Awards)
	}

	if len(out.TopComments) != 3 {
		t.Fatalf("TopComments: got %d, want 3", len(out.TopComments))
	}
	wantTop := []struct {
		author string
		score  int
		bodyHas string
	}{
		{"PLR1972", 32, "tracks accurately"},
		{"cliffr39", 26, "Whoop. $240/yr"},
		{"AllenLovis", 22, "I would find the most helpful"},
	}
	for i, w := range wantTop {
		got := out.TopComments[i]
		if got.Author != w.author {
			t.Errorf("TopComments[%d].Author: got %q, want %q", i, got.Author, w.author)
		}
		if got.Score != w.score {
			t.Errorf("TopComments[%d].Score: got %d, want %d", i, got.Score, w.score)
		}
		if !strings.Contains(got.Body, w.bodyHas) {
			t.Errorf("TopComments[%d].Body: got %q, want substring %q", i, got.Body, w.bodyHas)
		}
		if !strings.HasPrefix(got.Permalink, "https://www.reddit.com/r/whoop/comments/1sf4xjz/") {
			t.Errorf("TopComments[%d].Permalink: got %q, want under r/whoop/comments/1sf4xjz/", i, got.Permalink)
		}
	}
}

// TestParseAbbreviatedCount sanity-checks the K/M parser used as a
// fallback when the hourly chart isn't in the page (fresh posts).
func TestParseAbbreviatedCount(t *testing.T) {
	cases := map[string]int{
		"":      0,
		"847":   847,
		"51K":   51_000,
		"1.2M":  1_200_000,
		"3.5k":  3_500,
		"12.7m": 12_700_000,
		"junk":  0,
	}
	for in, want := range cases {
		if got := parseAbbreviatedCount(in); got != want {
			t.Errorf("parseAbbreviatedCount(%q) = %d, want %d", in, got, want)
		}
	}
}
