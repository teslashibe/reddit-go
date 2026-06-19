package reddit

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// Submit creates a new self-post in the given subreddit.
func (c *Client) Submit(subreddit, title, body string) (*Post, error) {
	form := url.Values{
		"api_type": {"json"},
		"kind":     {"self"},
		"sr":       {subreddit},
		"title":    {title},
		"text":     {body},
	}
	resp, err := c.oauthPost("/api/submit", form)
	if err != nil {
		return nil, fmt.Errorf("submitting post: %w", err)
	}
	return parseSubmitResponse(resp)
}

// SubmitLink creates a new link post in the given subreddit.
func (c *Client) SubmitLink(subreddit, title, linkURL string) (*Post, error) {
	form := url.Values{
		"api_type": {"json"},
		"kind":     {"link"},
		"sr":       {subreddit},
		"title":    {title},
		"url":      {linkURL},
	}
	resp, err := c.oauthPost("/api/submit", form)
	if err != nil {
		return nil, fmt.Errorf("submitting link: %w", err)
	}
	return parseSubmitResponse(resp)
}

func parseSubmitResponse(resp []byte) (*Post, error) {
	var result struct {
		JSON struct {
			Data struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				URL  string `json:"url"`
			} `json:"data"`
		} `json:"json"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("parsing submit response: %w", err)
	}
	return &Post{
		ID:       result.JSON.Data.ID,
		Fullname: result.JSON.Data.Name,
		URL:      result.JSON.Data.URL,
	}, nil
}

// Reply posts a comment reply to the given parent (post or comment fullname).
// Returns the new comment's ID.
func (c *Client) Reply(parentID, body string) (string, error) {
	form := url.Values{
		"api_type": {"json"},
		"thing_id": {parentID},
		"text":     {body},
	}
	resp, err := c.oauthPost("/api/comment", form)
	if err != nil {
		return "", fmt.Errorf("posting comment: %w", err)
	}

	var result struct {
		JSON struct {
			// Errors carries Reddit's application-level error list (e.g.
			// QUOTA_FILLED, RATELIMIT). Each entry is a 3-element slice:
			// [code, message, field]. The HTTP status is still 200, so
			// checking here is the only way to surface these failures.
			Errors [][]string `json:"errors"`
			Data   struct {
				Things []struct {
					Data struct {
						ID string `json:"id"`
					} `json:"data"`
				} `json:"things"`
			} `json:"data"`
		} `json:"json"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", fmt.Errorf("parsing comment response: %w", err)
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

// Edit updates the text of a self-post or comment identified by its fullname.
func (c *Client) Edit(fullname, newText string) error {
	form := url.Values{
		"api_type": {"json"},
		"thing_id": {fullname},
		"text":     {newText},
	}
	_, err := c.oauthPost("/api/editusertext", form)
	if err != nil {
		return fmt.Errorf("editing %s: %w", fullname, err)
	}
	return nil
}

// Delete removes a post or comment identified by its fullname.
func (c *Client) Delete(fullname string) error {
	form := url.Values{
		"id": {fullname},
	}
	_, err := c.oauthPost("/api/del", form)
	if err != nil {
		return fmt.Errorf("deleting %s: %w", fullname, err)
	}
	return nil
}

// PostInfo fetches the live state of a single post by ID, returning the
// current Score, UpvoteRatio, NumComments, and (for posts the
// authenticated user authored) ViewCount. Use this to refresh metrics
// on a post that's already been published.
//
// id may be either the bare post ID (e.g. "abc123") or the fullname
// (e.g. "t3_abc123") — both are normalized.
//
// For looking up multiple posts at once, prefer PostsInfo (one HTTP
// call vs N).
func (c *Client) PostInfo(id string) (*Post, error) {
	posts, err := c.PostsInfo([]string{id})
	if err != nil {
		return nil, err
	}
	if len(posts) == 0 {
		return nil, fmt.Errorf("post %s not found (deleted, removed, or invalid id)", id)
	}
	return &posts[0], nil
}

// PostsInfo fetches the live state of multiple posts in a single
// /api/info request. Reddit caps each /api/info call at 100 IDs; this
// method automatically chunks larger requests and returns a flat slice
// in the order Reddit returned them (which roughly matches input order
// but isn't strictly guaranteed).
//
// IDs missing from the response (deleted/removed posts, malformed IDs)
// are silently dropped; callers can detect them by comparing input
// length to output length.
//
// Each input id may be either a bare ID ("abc123") or a fullname
// ("t3_abc123") — both are normalized to the t3_ form Reddit expects.
func (c *Client) PostsInfo(ids []string) ([]Post, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	const maxPerRequest = 100
	out := make([]Post, 0, len(ids))
	for start := 0; start < len(ids); start += maxPerRequest {
		end := start + maxPerRequest
		if end > len(ids) {
			end = len(ids)
		}
		fullnames := make([]string, 0, end-start)
		for _, id := range ids[start:end] {
			fullnames = append(fullnames, normalizePostFullname(id))
		}
		path := "/api/info?id=" + url.QueryEscape(strings.Join(fullnames, ",")) + "&raw_json=1"
		body, err := c.oauthGet(path)
		if err != nil {
			return nil, fmt.Errorf("fetching post info: %w", err)
		}
		_, children, err := parseListing(body)
		if err != nil {
			return nil, fmt.Errorf("parsing post info: %w", err)
		}
		for _, child := range children {
			if child.Kind != "t3" {
				continue
			}
			var rp rawPost
			if err := json.Unmarshal(child.Data, &rp); err != nil {
				return nil, fmt.Errorf("unmarshaling post: %w", err)
			}
			out = append(out, rp.toPost())
		}
	}
	return out, nil
}

// PostCommentsOptions tunes a PostComments call. Zero values are sane
// defaults.
type PostCommentsOptions struct {
	// Sort is one of "confidence" (default), "top", "new", "controversial",
	// "old", "qa". Maps to Reddit's `sort` query param.
	Sort string
	// Limit caps the number of comments returned at the top level.
	// Reddit's max is 500. Zero defers to Reddit's default (~200).
	Limit int
	// Depth caps the comment-tree depth. Zero defers to Reddit's
	// default (full tree). Set to 1 if you only want top-level replies.
	Depth int
}

// PostComments fetches the post and its comment tree by post ID. The
// returned Post has the same engagement fields as PostInfo would
// return, and the comment slice is depth-first ordered (parents before
// their children) so callers can rebuild a tree by following ParentID.
//
// id may be the bare post ID or the fullname; the t3_ prefix is stripped.
//
// Reddit's comment endpoint returns "more" placeholders for very deep
// or very wide trees. PostComments quietly drops them — if you need
// every comment, fetch incrementally yourself or raise Limit.
func (c *Client) PostComments(id string, opts *PostCommentsOptions) (*Post, []Comment, error) {
	bareID := strings.TrimPrefix(normalizePostFullname(id), "t3_")
	if bareID == "" {
		return nil, nil, fmt.Errorf("empty post id")
	}
	if opts == nil {
		opts = &PostCommentsOptions{}
	}
	q := url.Values{}
	q.Set("raw_json", "1")
	if opts.Sort != "" {
		q.Set("sort", opts.Sort)
	}
	if opts.Limit > 0 {
		q.Set("limit", strconv.Itoa(opts.Limit))
	}
	if opts.Depth > 0 {
		q.Set("depth", strconv.Itoa(opts.Depth))
	}
	path := "/comments/" + bareID + ".json?" + q.Encode()
	body, err := c.oauthGet(path)
	if err != nil {
		return nil, nil, fmt.Errorf("fetching comments: %w", err)
	}
	// /comments returns a 2-element JSON array: [postListing, commentListing].
	var raw [2]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, nil, fmt.Errorf("parsing comments envelope: %w", err)
	}
	_, postChildren, err := parseListing(raw[0])
	if err != nil {
		return nil, nil, fmt.Errorf("parsing post listing: %w", err)
	}
	if len(postChildren) == 0 || postChildren[0].Kind != "t3" {
		return nil, nil, fmt.Errorf("post not found")
	}
	var rp rawPost
	if err := json.Unmarshal(postChildren[0].Data, &rp); err != nil {
		return nil, nil, fmt.Errorf("unmarshaling post: %w", err)
	}
	post := rp.toPost()

	_, commentChildren, err := parseListing(raw[1])
	if err != nil {
		return nil, nil, fmt.Errorf("parsing comment listing: %w", err)
	}
	comments := make([]Comment, 0, len(commentChildren))
	flattenComments(commentChildren, &comments)
	return &post, comments, nil
}

// flattenComments walks the children depth-first, appending each comment
// (kind == "t1") in encounter order. Reddit nests replies inside each
// comment's `replies` field as another listing — we recurse into it.
// "more" placeholders (kind == "more") are dropped: surfacing them up
// would force every caller to handle a sentinel type, and the agent
// callers we care about would just ignore them anyway.
func flattenComments(children []redditChild, out *[]Comment) {
	for _, child := range children {
		if child.Kind != "t1" {
			continue
		}
		// Comment data has both flat fields and an embedded `replies`
		// listing. Decode twice: once into rawComment for the flat
		// fields, once into a struct that pulls out the replies.
		var rc rawComment
		if err := json.Unmarshal(child.Data, &rc); err != nil {
			continue
		}
		*out = append(*out, rc.toComment())

		var replyEnvelope struct {
			Replies json.RawMessage `json:"replies"`
		}
		if err := json.Unmarshal(child.Data, &replyEnvelope); err != nil {
			continue
		}
		// Reddit returns "" (string) when there are no replies. Skip.
		if len(replyEnvelope.Replies) == 0 || string(replyEnvelope.Replies) == `""` {
			continue
		}
		_, grandchildren, err := parseListing(replyEnvelope.Replies)
		if err != nil {
			continue
		}
		flattenComments(grandchildren, out)
	}
}

// normalizePostFullname accepts either a bare post id ("abc123") or a
// fullname ("t3_abc123") and returns the canonical fullname Reddit
// expects in /api/info. Returns the input unchanged if it already has
// a known kind prefix.
func normalizePostFullname(id string) string {
	id = strings.TrimSpace(id)
	if strings.HasPrefix(id, "t3_") || strings.HasPrefix(id, "t1_") {
		return id
	}
	return "t3_" + id
}
