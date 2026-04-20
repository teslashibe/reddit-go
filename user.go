package reddit

import (
	"encoding/json"
	"fmt"
	"strconv"
)

func (c *Client) fetchPosts(endpoint string, limit int, after string) (*PostListing, error) {
	path := endpoint + "?raw_json=1&limit=" + strconv.Itoa(limit)
	if after != "" {
		path += "&after=" + after
	}

	body, err := c.oauthGet(path)
	if err != nil {
		return nil, err
	}

	listing, children, err := parseListing(body)
	if err != nil {
		return nil, fmt.Errorf("parsing post listing: %w", err)
	}

	result := &PostListing{
		After:  listing.Data.After,
		Before: listing.Data.Before,
	}
	for _, child := range children {
		var rp rawPost
		if err := json.Unmarshal(child.Data, &rp); err != nil {
			return nil, fmt.Errorf("unmarshaling post: %w", err)
		}
		result.Posts = append(result.Posts, rp.toPost())
	}
	return result, nil
}

func (c *Client) fetchComments(endpoint string, limit int, after string) (*CommentListing, error) {
	path := endpoint + "?raw_json=1&limit=" + strconv.Itoa(limit)
	if after != "" {
		path += "&after=" + after
	}

	body, err := c.oauthGet(path)
	if err != nil {
		return nil, err
	}

	listing, children, err := parseListing(body)
	if err != nil {
		return nil, fmt.Errorf("parsing comment listing: %w", err)
	}

	result := &CommentListing{
		After:  listing.Data.After,
		Before: listing.Data.Before,
	}
	for _, child := range children {
		var rc rawComment
		if err := json.Unmarshal(child.Data, &rc); err != nil {
			return nil, fmt.Errorf("unmarshaling comment: %w", err)
		}
		result.Comments = append(result.Comments, rc.toComment())
	}
	return result, nil
}

func (c *Client) fetchOverview(endpoint string, limit int, after string) (*OverviewListing, error) {
	path := endpoint + "?raw_json=1&limit=" + strconv.Itoa(limit)
	if after != "" {
		path += "&after=" + after
	}

	body, err := c.oauthGet(path)
	if err != nil {
		return nil, err
	}

	listing, children, err := parseListing(body)
	if err != nil {
		return nil, fmt.Errorf("parsing overview listing: %w", err)
	}

	result := &OverviewListing{
		After:  listing.Data.After,
		Before: listing.Data.Before,
	}
	for _, child := range children {
		switch child.Kind {
		case "t3":
			var rp rawPost
			if err := json.Unmarshal(child.Data, &rp); err != nil {
				return nil, fmt.Errorf("unmarshaling post in overview: %w", err)
			}
			p := rp.toPost()
			result.Items = append(result.Items, OverviewItem{Kind: "t3", Post: &p})
		case "t1":
			var rc rawComment
			if err := json.Unmarshal(child.Data, &rc); err != nil {
				return nil, fmt.Errorf("unmarshaling comment in overview: %w", err)
			}
			cm := rc.toComment()
			result.Items = append(result.Items, OverviewItem{Kind: "t1", Comment: &cm})
		}
	}
	return result, nil
}

// MyPosts returns the authenticated user's submitted posts.
func (c *Client) MyPosts(limit int) (*PostListing, error) {
	me, err := c.Me()
	if err != nil {
		return nil, fmt.Errorf("getting identity: %w", err)
	}
	return c.fetchPosts("/user/"+me.Name+"/submitted.json", limit, "")
}

// MyComments returns the authenticated user's comments.
func (c *Client) MyComments(limit int) (*CommentListing, error) {
	me, err := c.Me()
	if err != nil {
		return nil, fmt.Errorf("getting identity: %w", err)
	}
	return c.fetchComments("/user/"+me.Name+"/comments.json", limit, "")
}

// MyOverview returns the authenticated user's mixed post and comment feed.
func (c *Client) MyOverview(limit int) (*OverviewListing, error) {
	me, err := c.Me()
	if err != nil {
		return nil, fmt.Errorf("getting identity: %w", err)
	}
	return c.fetchOverview("/user/"+me.Name+"/overview.json", limit, "")
}

// Saved returns the authenticated user's saved items.
func (c *Client) Saved(limit int) (*OverviewListing, error) {
	me, err := c.Me()
	if err != nil {
		return nil, fmt.Errorf("getting identity: %w", err)
	}
	return c.fetchOverview("/user/"+me.Name+"/saved.json", limit, "")
}

// Upvoted returns the authenticated user's upvoted posts.
func (c *Client) Upvoted(limit int) (*PostListing, error) {
	me, err := c.Me()
	if err != nil {
		return nil, fmt.Errorf("getting identity: %w", err)
	}
	return c.fetchPosts("/user/"+me.Name+"/upvoted.json", limit, "")
}

// UserAbout returns public information about a Reddit user.
func (c *Client) UserAbout(username string) (*User, error) {
	var wrapper struct {
		Data rawUser `json:"data"`
	}
	if err := c.oauthGetJSON("/user/"+username+"/about.json", &wrapper); err != nil {
		return nil, fmt.Errorf("fetching user %s: %w", username, err)
	}
	u := wrapper.Data.toUser()
	return &u, nil
}
