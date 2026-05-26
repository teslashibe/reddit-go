package reddit

import (
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"
)

// PostInsights fetches the analytics view Reddit shows the post owner
// at https://www.reddit.com/poststats/{id}/. This is the canonical
// source for view counts and shares — the public OAuth API does NOT
// expose them, even for posts the authenticated user authored
// (view_count comes back null in /api/info for most accounts).
//
// Requires Options.Cookies to include the full logged-in session
// (specifically reddit_session and token_v2). Returns an error if
// cookies aren't set or if the page comes back as the login screen.
//
// Implementation note: Reddit doesn't expose post insights via any
// JSON endpoint, so this scrapes the rendered HTML. The shreddit
// templates are stable but if Reddit re-themes the page the parsers
// here will silently start returning zero values for the affected
// sections — guard at the call site by checking PostInsights.Title
// (always populated when parse succeeded).
//
// Rate-limit footprint: one ~470KB HTML fetch per call. Caller is
// responsible for not re-polling more than every few minutes;
// reddit-go's MinRequestGap throttles back-to-back calls but doesn't
// dedupe.
func (c *Client) PostInsights(id string) (*PostInsights, error) {
	bareID := strings.TrimPrefix(normalizePostFullname(id), "t3_")
	if bareID == "" {
		return nil, fmt.Errorf("empty post id")
	}
	body, err := c.webGet("/poststats/" + bareID + "/")
	if err != nil {
		return nil, fmt.Errorf("fetching post insights for %s: %w", bareID, err)
	}
	htmlSrc := string(body)

	insights := &PostInsights{PostID: bareID}
	parsePostInsightsHTML(htmlSrc, insights)
	if insights.Title == "" && insights.Upvotes == 0 && insights.TotalViews == 0 {
		// Nothing parsed — either the page layout changed or the
		// post id was invalid (Reddit serves a generic 404 page for
		// missing posts that still renders the shreddit shell, so
		// status code alone isn't enough to detect this).
		return nil, fmt.Errorf("post insights parser returned no data for %s — page layout may have changed or post is inaccessible", bareID)
	}
	return insights, nil
}

// Pre-compiled regexes used by parsePostInsightsHTML. Compiled once
// at package init since each PostInsights call re-uses them and these
// regexes aren't cheap (some have alternation + counted capture
// groups).
var (
	// Title sits in an <h2> with this exact class set in the
	// post-card section. Class order is stable in the rendered
	// output (template, not user-editable) so we anchor on it.
	rxInsightsTitle = regexp.MustCompile(
		`(?s)<h2 class="text-16 font-semibold line-clamp-2 m-0 mt-2xs">\s*(.+?)\s*</h2>`,
	)
	// Sub name comes from the post-card aria-label "... from r/{sub}".
	rxInsightsSubreddit = regexp.MustCompile(`from r/([A-Za-z0-9_]+)`)
	// Permalink to the actual post — first <a href> inside the
	// post-card, anchored on the /comments/ path.
	rxInsightsPermalink = regexp.MustCompile(`href="(/r/[^"]+/comments/[^"]+/)"`)
	// Personal-comparison ribbon — the text right after the
	// testid="personal-comparison-section" container. The text is
	// inside the same <div>, after the closing </ac-track> tag.
	rxInsightsRibbon = regexp.MustCompile(
		`(?s)data-testid="personal-comparison-section"[^>]*>\s*<ac-track[^>]*></ac-track>\s*([^<]+?)\s*</div>`,
	)

	// Reach — total views.
	// "51K views" or "1.2M views" or "847 views". The aria-label is
	// reliable; the inline <faceplate-number> nearby has the raw
	// integer for ranges where Reddit shows the exact value (under
	// 10K) but for larger numbers Reddit only renders the
	// abbreviated form on the page, so we sum the hourly chart for
	// the canonical total when needed.
	rxInsightsViewsAria = regexp.MustCompile(
		`aria-label="([0-9]+(?:\.[0-9]+)?[KMm]?) views"`,
	)
	// Views change ("+74", "-12", or absent for fresh posts).
	// Reddit puts it in a span with this exact testid.
	rxInsightsViewsChange = regexp.MustCompile(
		`data-testid="stat-row-value-change">\s*([+-]?[0-9,]+)\s*</span>`,
	)
	// Per-hour view counts — pulled from the screen-reader content
	// rather than the SVG <rect> elements because the SR text
	// orders them chronologically with explicit "At hour N" labels,
	// vs the SVG which is rotated and indexed by x-coordinate.
	rxInsightsHourly = regexp.MustCompile(
		`At hour (\d+): ([0-9,]+) views`,
	)

	// Engagement — each row is an aria-label on a "stat row" div
	// in the Engagement section. The page renders each stat
	// (upvotes, comments, shares, crossposts, awards) at the
	// post-level FIRST, followed by per-comment counts inside the
	// Top Comments section. We rely on regex returning the first
	// match (which is always the post-level one) instead of
	// anchoring on the "Engagement" h2 — anchoring is fragile if
	// Reddit renumbers their tailwind classes.
	rxInsightsUpvotes = regexp.MustCompile(
		`aria-label="([0-9,]+) upvotes?"`,
	)
	rxInsightsUpvoteRatio = regexp.MustCompile(
		`aria-label="([0-9.]+)% upvote ratio"`,
	)
	rxInsightsComments = regexp.MustCompile(
		`aria-label="([0-9,]+) comments?"`,
	)
	rxInsightsShares = regexp.MustCompile(
		`aria-label="([0-9,]+) shares?"`,
	)
	rxInsightsCrossposts = regexp.MustCompile(
		`aria-label="([0-9,]+) crossposts?"`,
	)
	rxInsightsAwards = regexp.MustCompile(
		`aria-label="([0-9,]+) awards?"`,
	)

	// Top comments — Reddit puts each in a card with three
	// useful regions: an aria-label encoding author/age/score
	// (truncated body), an <a> with the canonical permalink, and
	// an inner <div id="-post-rtjson-content">…<p>full body</p>
	// </div>. We capture all three in one match so the indexes
	// stay aligned across comments. The aria-label body is
	// truncated at ~300 chars (Reddit appends "…" or just cuts);
	// we prefer the rtjson body which is always the un-truncated
	// markdown render.
	rxInsightsTopComment = regexp.MustCompile(
		`(?s)aria-label="Comment by ([^,"]+), [^,"]+, Number of votes: (\d+)\. [^"]*"[\s\S]*?<a href="(/r/[^"]+)"[\s\S]*?id="-post-rtjson-content"[^>]*>\s*<p>\s*(.+?)\s*</p>`,
	)
)

// parsePostInsightsHTML mutates the passed-in PostInsights struct
// with whatever fields it can extract from the HTML. Missing fields
// stay at their zero value — we don't return an error on partial
// parses because Reddit silently drops sections (e.g. "Reach" before
// the first hour passes, "Top comments" on posts with <3 comments)
// and we want callers to still get the engagement numbers in those
// cases.
func parsePostInsightsHTML(src string, out *PostInsights) {
	if m := rxInsightsTitle.FindStringSubmatch(src); len(m) > 1 {
		out.Title = strings.TrimSpace(html.UnescapeString(m[1]))
	}
	if m := rxInsightsSubreddit.FindStringSubmatch(src); len(m) > 1 {
		out.Subreddit = m[1]
	}
	if m := rxInsightsPermalink.FindStringSubmatch(src); len(m) > 1 {
		out.Permalink = "https://www.reddit.com" + m[1]
	}
	if m := rxInsightsRibbon.FindStringSubmatch(src); len(m) > 1 {
		out.PersonalComparison = strings.TrimSpace(html.UnescapeString(m[1]))
	}

	if m := rxInsightsViewsAria.FindStringSubmatch(src); len(m) > 1 {
		out.TotalViewsFormatted = m[1]
	}
	if m := rxInsightsViewsChange.FindStringSubmatch(src); len(m) > 1 {
		raw := strings.TrimSpace(m[1])
		out.ViewsChangeFormatted = raw
		if n, err := strconv.Atoi(strings.ReplaceAll(strings.TrimPrefix(raw, "+"), ",", "")); err == nil {
			out.ViewsChange = n
		}
	}

	if hourly := rxInsightsHourly.FindAllStringSubmatch(src, -1); len(hourly) > 0 {
		out.HourlyViews = make([]HourlyViews, 0, len(hourly))
		var sum int
		for _, h := range hourly {
			hour, _ := strconv.Atoi(h[1])
			views, err := strconv.Atoi(strings.ReplaceAll(h[2], ",", ""))
			if err != nil {
				continue
			}
			out.HourlyViews = append(out.HourlyViews, HourlyViews{HourOffset: hour, Views: views})
			sum += views
		}
		// Sort by HourOffset so callers don't have to (regex
		// matches in document order, which Reddit already orders
		// chronologically — but defensive sort costs nothing).
		sortHourly(out.HourlyViews)
		// Use the hourly sum as the canonical TotalViews when
		// it's larger than the parsed view_count from /api/info
		// (which is null for most accounts) — but only when the
		// formatted total agrees with sign. The hourly chart caps
		// at 48 hours, so for posts older than 2 days the sum is
		// a lower bound, not the true total. We still set
		// TotalViews to the sum since it's better than zero, and
		// flag the truth via TotalViewsFormatted (which is always
		// the true total per Reddit).
		out.TotalViews = sum
	}

	// If the page rendered the formatted "51K" but no chart, parse
	// the formatted string as a fallback. We tolerate "K"/"M"
	// suffixes and decimals, returning the rounded integer.
	if out.TotalViews == 0 && out.TotalViewsFormatted != "" {
		out.TotalViews = parseAbbreviatedCount(out.TotalViewsFormatted)
	}

	if m := rxInsightsUpvotes.FindStringSubmatch(src); len(m) > 1 {
		out.Upvotes = parseIntCommas(m[1])
	}
	if m := rxInsightsUpvoteRatio.FindStringSubmatch(src); len(m) > 1 {
		if f, err := strconv.ParseFloat(m[1], 64); err == nil {
			out.UpvoteRatio = f / 100.0
		}
	}
	if m := rxInsightsComments.FindStringSubmatch(src); len(m) > 1 {
		out.Comments = parseIntCommas(m[1])
	}
	if m := rxInsightsShares.FindStringSubmatch(src); len(m) > 1 {
		out.Shares = parseIntCommas(m[1])
	}
	if m := rxInsightsCrossposts.FindStringSubmatch(src); len(m) > 1 {
		out.Crossposts = parseIntCommas(m[1])
	}
	if m := rxInsightsAwards.FindStringSubmatch(src); len(m) > 1 {
		out.Awards = parseIntCommas(m[1])
	}

	if matches := rxInsightsTopComment.FindAllStringSubmatch(src, -1); len(matches) > 0 {
		out.TopComments = make([]InsightTopComment, 0, len(matches))
		for _, m := range matches {
			score, _ := strconv.Atoi(m[2])
			out.TopComments = append(out.TopComments, InsightTopComment{
				Author:    strings.TrimSpace(m[1]),
				Score:     score,
				Permalink: "https://www.reddit.com" + m[3],
				Body:      strings.TrimSpace(html.UnescapeString(m[4])),
			})
		}
	}
}

// parseAbbreviatedCount turns Reddit's "51K"/"1.2M"/"847" view-count
// shorthand into an integer. Returns 0 on parse failure (callers fall
// back to the hourly-sum estimate).
func parseAbbreviatedCount(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	mult := 1
	switch {
	case strings.HasSuffix(s, "K"), strings.HasSuffix(s, "k"):
		mult = 1_000
		s = s[:len(s)-1]
	case strings.HasSuffix(s, "M"), strings.HasSuffix(s, "m"):
		mult = 1_000_000
		s = s[:len(s)-1]
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return int(f * float64(mult))
}

// parseIntCommas strips thousands-separator commas before parsing.
// Reddit renders numbers with commas in some places ("1,234 upvotes")
// and bare in others; this normalizes both.
func parseIntCommas(s string) int {
	n, err := strconv.Atoi(strings.ReplaceAll(strings.TrimSpace(s), ",", ""))
	if err != nil {
		return 0
	}
	return n
}

// sortHourly orders the slice in-place by HourOffset ascending.
// Implemented inline (no `sort` import) since the slice is always
// <= 48 elements and a tiny insertion sort is just as fast as
// sort.Slice's overhead.
func sortHourly(v []HourlyViews) {
	for i := 1; i < len(v); i++ {
		x := v[i]
		j := i - 1
		for j >= 0 && v[j].HourOffset > x.HourOffset {
			v[j+1] = v[j]
			j--
		}
		v[j+1] = x
	}
}
