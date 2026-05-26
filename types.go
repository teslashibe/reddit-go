package reddit

import (
	"encoding/json"
	"time"
)

// RateLimitState captures rate-limit information from the most recently observed
// response headers. All fields are zero-valued until a response with rate-limit
// headers is received.
type RateLimitState struct {
	Limit      int           `json:"limit"`       // max requests per window (0 = not reported)
	Remaining  int           `json:"remaining"`   // requests left in the current window
	Reset      time.Time     `json:"reset"`       // when the window resets (UTC)
	RetryAfter time.Duration `json:"retry_after"` // set to Retry-After duration after a 429
}

// IsLimited reports whether the current state indicates requests are blocked.
func (r RateLimitState) IsLimited() bool {
	if !r.Reset.IsZero() && r.Remaining == 0 && time.Now().Before(r.Reset) {
		return true
	}
	return r.RetryAfter > 0
}

// ResetIn returns how long until the rate-limit window resets.
// Returns 0 if Reset is in the past or not set.
func (r RateLimitState) ResetIn() time.Duration {
	if r.Reset.IsZero() {
		return 0
	}
	if d := time.Until(r.Reset); d > 0 {
		return d
	}
	return 0
}

// Message represents a Reddit private message or inbox item.
type Message struct {
	ID        string    `json:"id"`
	Fullname  string    `json:"name"`
	Author    string    `json:"author"`
	Dest      string    `json:"dest"`
	Subject   string    `json:"subject"`
	Body      string    `json:"body"`
	BodyHTML  string    `json:"body_html"`
	Created   time.Time `json:"created_utc"`
	IsNew     bool      `json:"new"`
	Type      string    `json:"type"`
	Context   string    `json:"context,omitempty"`
	Subreddit string    `json:"subreddit,omitempty"`
	ParentID  string    `json:"parent_id,omitempty"`

	WasComment bool `json:"was_comment"`
}

// MessageListing is a paginated list of messages.
type MessageListing struct {
	Messages []Message
	After    string
	Before   string
}

// Post represents a Reddit submission.
//
// The engagement-relevant fields (Score, UpvoteRatio, NumComments,
// ViewCount, TotalAwards, Gilded) are point-in-time snapshots from the
// most recent fetch. Reddit does not stream updates; re-fetch via
// Client.PostInfo or Client.PostsInfo to refresh them.
type Post struct {
	ID            string    `json:"id"`
	Fullname      string    `json:"name"`
	Subreddit     string    `json:"subreddit"`
	Title         string    `json:"title"`
	Author        string    `json:"author"`
	SelfText      string    `json:"selftext"`
	URL           string    `json:"url"`
	Permalink     string    `json:"permalink"`
	Domain        string    `json:"domain"`
	Score         int       `json:"score"`
	UpvoteRatio   float64   `json:"upvote_ratio"`
	NumComments   int       `json:"num_comments"`
	// ViewCount is Reddit's reported view count when available. Reddit
	// only exposes this for posts the authenticated user authored, and
	// even then it can be absent on very fresh posts. Pointer so callers
	// can distinguish "unknown" (nil) from "zero views" (*v == 0).
	ViewCount   *int `json:"view_count,omitempty"`
	TotalAwards int  `json:"total_awards_received"`
	Gilded      int  `json:"gilded"`
	Created       time.Time `json:"created_utc"`
	IsSelf        bool      `json:"is_self"`
	Over18        bool      `json:"over_18"`
	Stickied      bool      `json:"stickied"`
	Locked        bool      `json:"locked"`
	Archived      bool      `json:"archived"`
	Saved         bool      `json:"saved"`
	Hidden        bool      `json:"hidden"`
	IsVideo       bool      `json:"is_video"`
	Distinguished string    `json:"distinguished,omitempty"`
	LinkFlairText string    `json:"link_flair_text,omitempty"`
}

// PostListing is a paginated list of posts.
type PostListing struct {
	Posts  []Post
	After  string
	Before string
}

// Comment represents a Reddit comment.
type Comment struct {
	ID            string    `json:"id"`
	Fullname      string    `json:"name"`
	Author        string    `json:"author"`
	Body          string    `json:"body"`
	BodyHTML      string    `json:"body_html"`
	Subreddit     string    `json:"subreddit"`
	Score         int       `json:"score"`
	Created       time.Time `json:"created_utc"`
	ParentID      string    `json:"parent_id"`
	LinkID        string    `json:"link_id"`
	LinkTitle     string    `json:"link_title"`
	Permalink     string    `json:"permalink"`
	Depth         int       `json:"depth"`
	IsSubmitter   bool      `json:"is_submitter"`
	Stickied      bool      `json:"stickied"`
	Edited        bool      `json:"edited"`
	Distinguished string    `json:"distinguished,omitempty"`
}

// CommentListing is a paginated list of comments.
type CommentListing struct {
	Comments []Comment
	After    string
	Before   string
}

// OverviewItem is either a Post or Comment from the overview feed.
type OverviewItem struct {
	Kind    string   // "t3" for post, "t1" for comment
	Post    *Post    // set when Kind == "t3"
	Comment *Comment // set when Kind == "t1"
}

// OverviewListing is a paginated list of overview items.
type OverviewListing struct {
	Items  []OverviewItem
	After  string
	Before string
}

// User holds public info about a Reddit user.
type User struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Created      time.Time `json:"created_utc"`
	LinkKarma    int       `json:"link_karma"`
	CommentKarma int       `json:"comment_karma"`
	TotalKarma   int       `json:"total_karma"`
	IsGold       bool      `json:"is_gold"`
	IsMod        bool      `json:"is_mod"`
	Verified     bool      `json:"verified"`
	HasVerifiedEmail bool  `json:"has_verified_email"`
	IconImg      string    `json:"icon_img"`
}

// SubredditInfo holds metadata about a subreddit.
//
// Rules and SiteRules are populated by SubredditAbout (which makes a
// second request to /r/{name}/about/rules under the hood) and by
// SubredditRules. They are nil — not empty slices — when the rules
// endpoint hasn't been queried yet, so callers can distinguish
// "no rules" from "rules not loaded".
type SubredditInfo struct {
	ID                string    `json:"id"`
	Name              string    `json:"display_name"`
	Title             string    `json:"title"`
	Description       string    `json:"public_description"`
	Subscribers       int       `json:"subscribers"`
	ActiveUsers       int       `json:"accounts_active"`
	Created           time.Time `json:"created_utc"`
	Over18            bool      `json:"over_18"`
	SubredditType     string    `json:"subreddit_type"`
	Rules             []SubredditRule `json:"rules,omitempty"`
	SiteRules         []string        `json:"site_rules,omitempty"`
}

// SubredditRule is one moderator-defined posting rule for a subreddit.
//
// Kind is one of "all", "link", or "comment" and tells you whether the
// rule applies to posts, links specifically, comments specifically, or
// everything. ShortName is the bold headline (e.g. "No self-promotion"),
// Description is the longer Markdown body shown on the rules page.
type SubredditRule struct {
	Kind            string    `json:"kind"`
	ShortName       string    `json:"short_name"`
	Description     string    `json:"description"`
	ViolationReason string    `json:"violation_reason,omitempty"`
	Priority        int       `json:"priority"`
	Created         time.Time `json:"created_utc"`
}

// SubredditRules is the response from /r/{name}/about/rules — moderator
// rules plus the site-wide rules Reddit applies on top of every sub.
type SubredditRules struct {
	Rules     []SubredditRule `json:"rules"`
	SiteRules []string        `json:"site_rules"`
}

// SubredditListing is a paginated list of subreddits.
type SubredditListing struct {
	Subreddits []SubredditInfo
	After      string
	Before     string
}

// PostInsights captures the metrics shown on the /poststats/{id}/ page —
// the analytics view Reddit only renders for the post's author. This is
// strictly richer than what /api/info returns: hourly view breakdown,
// share count, top comments, etc. Fields can be zero/empty when the
// section is missing on the rendered page (e.g. very fresh posts have
// no hourly chart yet) so callers should treat zero as "unknown" not
// "definitely zero" for the chart-only fields (HourlyViews).
type PostInsights struct {
	// PostID is the bare post id ("1sf4xjz", no t3_ prefix) the
	// insights are for. Echoed back from the input so callers
	// don't have to re-pass it.
	PostID string `json:"post_id"`
	// Title and Subreddit are the post's display headline and home
	// sub, scraped from the post-card section at the top of the
	// insights page.
	Title     string `json:"title,omitempty"`
	Subreddit string `json:"subreddit,omitempty"`
	// Permalink is the full URL to the post (always /r/.../comments/...).
	// Useful to render a "View on Reddit" link alongside the metrics.
	Permalink string `json:"permalink,omitempty"`
	// PersonalComparison is the headline ribbon at the top of the
	// page, e.g. "🥇 Incredible! This is your #1 post of all time"
	// or "Top 5% of your posts". Empty when the page has no ribbon.
	PersonalComparison string `json:"personal_comparison,omitempty"`

	// Reach.
	TotalViews          int    `json:"total_views"`
	TotalViewsFormatted string `json:"total_views_formatted,omitempty"`
	// ViewsChange is the 24-hour delta Reddit shows next to the
	// total ("+74"). Negative numbers are possible if the post is
	// losing visibility. Stored as the raw integer; the formatted
	// "+74" is in ViewsChangeFormatted.
	ViewsChange          int    `json:"views_change"`
	ViewsChangeFormatted string `json:"views_change_formatted,omitempty"`
	// HourlyViews is the per-hour view chart for the first 48
	// hours after posting. Index 0 == hour 1. Empty for very fresh
	// posts (the chart needs at least 1h of data) and for posts
	// older than ~30 days (Reddit drops the chart eventually).
	HourlyViews []HourlyViews `json:"hourly_views,omitempty"`

	// Engagement.
	Upvotes     int     `json:"upvotes"`
	UpvoteRatio float64 `json:"upvote_ratio"`
	Comments    int     `json:"comments"`
	Shares      int     `json:"shares"`
	Crossposts  int     `json:"crossposts"`
	Awards      int     `json:"awards"`

	// TopComments is the leading comments Reddit highlights on the
	// insights page (always 3 in our observation, but treat as
	// "up to 3"). Ordered by Reddit's pick (usually highest score).
	TopComments []InsightTopComment `json:"top_comments,omitempty"`
}

// HourlyViews is one bar of the 48-hour view chart. HourOffset is
// 1-indexed (matches Reddit's "Hour 1" label) and Views is the count
// for that hour.
type HourlyViews struct {
	HourOffset int `json:"hour_offset"`
	Views      int `json:"views"`
}

// InsightTopComment is one of the "top comments" Reddit highlights at
// the bottom of the insights page. Body is the un-truncated comment
// text (the page sometimes truncates with "…" but Reddit re-renders
// the full text in the screen-reader metadata, which we prefer).
type InsightTopComment struct {
	Author    string `json:"author"`
	Body      string `json:"body"`
	Score     int    `json:"score"`
	Permalink string `json:"permalink"`
}

// Trophy represents a Reddit trophy/award.
type Trophy struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	AwardID     string `json:"award_id"`
	IconURL     string `json:"icon_70"`
}

// Friend represents a Reddit friend.
type Friend struct {
	Name string    `json:"name"`
	ID   string    `json:"id"`
	Date time.Time `json:"date"`
}

// --- internal raw types for JSON decoding ---

type redditListing struct {
	Kind string `json:"kind"`
	Data struct {
		After    string          `json:"after"`
		Before   string          `json:"before"`
		Children json.RawMessage `json:"children"`
	} `json:"data"`
}

type redditChild struct {
	Kind string          `json:"kind"`
	Data json.RawMessage `json:"data"`
}

type rawMessage struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Author     string  `json:"author"`
	Dest       string  `json:"dest"`
	Subject    string  `json:"subject"`
	Body       string  `json:"body"`
	BodyHTML   string  `json:"body_html"`
	CreatedUTC float64 `json:"created_utc"`
	New        bool    `json:"new"`
	Type       string  `json:"type"`
	Context    string  `json:"context"`
	Subreddit  string  `json:"subreddit"`
	ParentID   string  `json:"parent_id"`
	WasComment bool    `json:"was_comment"`
}

func (r *rawMessage) toMessage() Message {
	return Message{
		ID:         r.ID,
		Fullname:   r.Name,
		Author:     r.Author,
		Dest:       r.Dest,
		Subject:    r.Subject,
		Body:       r.Body,
		BodyHTML:   r.BodyHTML,
		Created:    time.Unix(int64(r.CreatedUTC), 0),
		IsNew:      r.New,
		Type:       r.Type,
		Context:    r.Context,
		Subreddit:  r.Subreddit,
		ParentID:   r.ParentID,
		WasComment: r.WasComment,
	}
}

type rawPost struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Subreddit     string  `json:"subreddit"`
	Title         string  `json:"title"`
	Author        string  `json:"author"`
	SelfText      string  `json:"selftext"`
	URL           string  `json:"url"`
	Permalink     string  `json:"permalink"`
	Domain        string  `json:"domain"`
	Score         int     `json:"score"`
	UpvoteRatio   float64 `json:"upvote_ratio"`
	NumComments   int     `json:"num_comments"`
	// ViewCount is *int because Reddit returns null for posts the
	// authenticated user did not author. Without the pointer we'd
	// silently treat unknown views as zero.
	ViewCount     *int    `json:"view_count"`
	TotalAwards   int     `json:"total_awards_received"`
	Gilded        int     `json:"gilded"`
	CreatedUTC    float64 `json:"created_utc"`
	IsSelf        bool    `json:"is_self"`
	Over18        bool    `json:"over_18"`
	Stickied      bool    `json:"stickied"`
	Locked        bool    `json:"locked"`
	Archived      bool    `json:"archived"`
	Saved         bool    `json:"saved"`
	Hidden        bool    `json:"hidden"`
	IsVideo       bool    `json:"is_video"`
	Distinguished string  `json:"distinguished"`
	LinkFlairText string  `json:"link_flair_text"`
}

func (r *rawPost) toPost() Post {
	return Post{
		ID:            r.ID,
		Fullname:      r.Name,
		Subreddit:     r.Subreddit,
		Title:         r.Title,
		Author:        r.Author,
		SelfText:      r.SelfText,
		URL:           r.URL,
		Permalink:     r.Permalink,
		Domain:        r.Domain,
		Score:         r.Score,
		UpvoteRatio:   r.UpvoteRatio,
		NumComments:   r.NumComments,
		ViewCount:     r.ViewCount,
		TotalAwards:   r.TotalAwards,
		Gilded:        r.Gilded,
		Created:       time.Unix(int64(r.CreatedUTC), 0),
		IsSelf:        r.IsSelf,
		Over18:        r.Over18,
		Stickied:      r.Stickied,
		Locked:        r.Locked,
		Archived:      r.Archived,
		Saved:         r.Saved,
		Hidden:        r.Hidden,
		IsVideo:       r.IsVideo,
		Distinguished: r.Distinguished,
		LinkFlairText: r.LinkFlairText,
	}
}

type rawComment struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Author        string  `json:"author"`
	Body          string  `json:"body"`
	BodyHTML      string  `json:"body_html"`
	Subreddit     string  `json:"subreddit"`
	Score         int     `json:"score"`
	CreatedUTC    float64 `json:"created_utc"`
	ParentID      string  `json:"parent_id"`
	LinkID        string  `json:"link_id"`
	LinkTitle     string  `json:"link_title"`
	Permalink     string  `json:"permalink"`
	Depth         int     `json:"depth"`
	IsSubmitter   bool    `json:"is_submitter"`
	Stickied      bool    `json:"stickied"`
	Edited        bool    `json:"edited"`
	Distinguished string  `json:"distinguished"`
}

func (r *rawComment) toComment() Comment {
	return Comment{
		ID:            r.ID,
		Fullname:      r.Name,
		Author:        r.Author,
		Body:          r.Body,
		BodyHTML:      r.BodyHTML,
		Subreddit:     r.Subreddit,
		Score:         r.Score,
		Created:       time.Unix(int64(r.CreatedUTC), 0),
		ParentID:      r.ParentID,
		LinkID:        r.LinkID,
		LinkTitle:     r.LinkTitle,
		Permalink:     r.Permalink,
		Depth:         r.Depth,
		IsSubmitter:   r.IsSubmitter,
		Stickied:      r.Stickied,
		Edited:        r.Edited,
		Distinguished: r.Distinguished,
	}
}

type rawUser struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	CreatedUTC       float64 `json:"created_utc"`
	LinkKarma        int     `json:"link_karma"`
	CommentKarma     int     `json:"comment_karma"`
	TotalKarma       int     `json:"total_karma"`
	IsGold           bool    `json:"is_gold"`
	IsMod            bool    `json:"is_mod"`
	Verified         bool    `json:"verified"`
	HasVerifiedEmail bool    `json:"has_verified_email"`
	IconImg          string  `json:"icon_img"`
}

func (r *rawUser) toUser() User {
	return User{
		ID:               r.ID,
		Name:             r.Name,
		Created:          time.Unix(int64(r.CreatedUTC), 0),
		LinkKarma:        r.LinkKarma,
		CommentKarma:     r.CommentKarma,
		TotalKarma:       r.TotalKarma,
		IsGold:           r.IsGold,
		IsMod:            r.IsMod,
		Verified:         r.Verified,
		HasVerifiedEmail: r.HasVerifiedEmail,
		IconImg:          r.IconImg,
	}
}

type rawSubreddit struct {
	ID             string  `json:"id"`
	DisplayName    string  `json:"display_name"`
	Title          string  `json:"title"`
	PublicDesc     string  `json:"public_description"`
	Subscribers    int     `json:"subscribers"`
	AccountsActive int    `json:"accounts_active"`
	CreatedUTC     float64 `json:"created_utc"`
	Over18         bool    `json:"over_18"`
	SubredditType  string  `json:"subreddit_type"`
}

func (r *rawSubreddit) toSubredditInfo() SubredditInfo {
	return SubredditInfo{
		ID:            r.ID,
		Name:          r.DisplayName,
		Title:         r.Title,
		Description:   r.PublicDesc,
		Subscribers:   r.Subscribers,
		ActiveUsers:   r.AccountsActive,
		Created:       time.Unix(int64(r.CreatedUTC), 0),
		Over18:        r.Over18,
		SubredditType: r.SubredditType,
	}
}

// rawSubredditRules mirrors the JSON shape returned by
// /r/{name}/about/rules. Reddit returns site-wide rules in two flavors —
// the legacy `site_rules` (flat list of strings) and the newer
// `site_rules_flow` (richer per-rule objects). We only surface
// `site_rules` because the simple list is what the agent actually needs
// for "summarise the rules before posting", and the flow shape adds a
// lot of bytes that aren't worth their schema cost.
type rawSubredditRules struct {
	Rules []struct {
		Kind            string  `json:"kind"`
		ShortName       string  `json:"short_name"`
		Description     string  `json:"description"`
		ViolationReason string  `json:"violation_reason"`
		Priority        int     `json:"priority"`
		CreatedUTC      float64 `json:"created_utc"`
	} `json:"rules"`
	SiteRules []string `json:"site_rules"`
}

func (r *rawSubredditRules) toSubredditRules() SubredditRules {
	out := SubredditRules{
		Rules:     make([]SubredditRule, 0, len(r.Rules)),
		SiteRules: append([]string(nil), r.SiteRules...),
	}
	for _, raw := range r.Rules {
		out.Rules = append(out.Rules, SubredditRule{
			Kind:            raw.Kind,
			ShortName:       raw.ShortName,
			Description:     raw.Description,
			ViolationReason: raw.ViolationReason,
			Priority:        raw.Priority,
			Created:         time.Unix(int64(raw.CreatedUTC), 0),
		})
	}
	return out
}

// parseListing is a generic listing parser that handles the Reddit listing
// wrapper and dispatches children based on kind.
func parseListing(body []byte) (*redditListing, []redditChild, error) {
	var listing redditListing
	if err := json.Unmarshal(body, &listing); err != nil {
		return nil, nil, err
	}
	var children []redditChild
	if err := json.Unmarshal(listing.Data.Children, &children); err != nil {
		return nil, nil, err
	}
	return &listing, children, nil
}
