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
}

// SubredditListing is a paginated list of subreddits.
type SubredditListing struct {
	Subreddits []SubredditInfo
	After      string
	Before     string
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
