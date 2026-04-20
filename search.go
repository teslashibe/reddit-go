package reddit

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// Search searches all of Reddit for posts matching the query.
func (c *Client) Search(query string, limit int) (*PostListing, error) {
	path := "/search.json?raw_json=1&q=" + url.QueryEscape(query) + "&limit=" + strconv.Itoa(limit)
	return c.fetchSearchPosts(path)
}

// SearchInSubreddit searches within a specific subreddit for posts matching the query.
func (c *Client) SearchInSubreddit(subreddit, query string, limit int) (*PostListing, error) {
	path := "/r/" + subreddit + "/search.json?raw_json=1&restrict_sr=on&q=" + url.QueryEscape(query) + "&limit=" + strconv.Itoa(limit)
	return c.fetchSearchPosts(path)
}

func (c *Client) fetchSearchPosts(path string) (*PostListing, error) {
	body, err := c.oauthGet(path)
	if err != nil {
		return nil, err
	}

	listing, children, err := parseListing(body)
	if err != nil {
		return nil, fmt.Errorf("parsing search listing: %w", err)
	}

	result := &PostListing{
		After:  listing.Data.After,
		Before: listing.Data.Before,
	}
	for _, child := range children {
		var rp rawPost
		if err := json.Unmarshal(child.Data, &rp); err != nil {
			return nil, fmt.Errorf("unmarshaling search result: %w", err)
		}
		result.Posts = append(result.Posts, rp.toPost())
	}
	return result, nil
}
