package reddit

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// Subscribe subscribes the authenticated user to a subreddit.
func (c *Client) Subscribe(subreddit string) error {
	form := url.Values{
		"action":               {"sub"},
		"sr_name":              {subreddit},
		"skip_initial_defaults": {"true"},
	}
	_, err := c.oauthPost("/api/subscribe", form)
	if err != nil {
		return fmt.Errorf("subscribing to %s: %w", subreddit, err)
	}
	return nil
}

// Unsubscribe removes the authenticated user's subscription to a subreddit.
func (c *Client) Unsubscribe(subreddit string) error {
	form := url.Values{
		"action":  {"unsub"},
		"sr_name": {subreddit},
	}
	_, err := c.oauthPost("/api/subscribe", form)
	if err != nil {
		return fmt.Errorf("unsubscribing from %s: %w", subreddit, err)
	}
	return nil
}

// MySubscriptions returns the authenticated user's subscribed subreddits.
func (c *Client) MySubscriptions(limit int) (*SubredditListing, error) {
	path := "/subreddits/mine/subscriber.json?raw_json=1&limit=" + strconv.Itoa(limit)

	body, err := c.oauthGet(path)
	if err != nil {
		return nil, err
	}

	listing, children, err := parseListing(body)
	if err != nil {
		return nil, fmt.Errorf("parsing subreddit listing: %w", err)
	}

	result := &SubredditListing{
		After:  listing.Data.After,
		Before: listing.Data.Before,
	}
	for _, child := range children {
		var rs rawSubreddit
		if err := json.Unmarshal(child.Data, &rs); err != nil {
			return nil, fmt.Errorf("unmarshaling subreddit: %w", err)
		}
		result.Subreddits = append(result.Subreddits, rs.toSubredditInfo())
	}
	return result, nil
}

// SubredditAbout returns metadata about a subreddit.
func (c *Client) SubredditAbout(name string) (*SubredditInfo, error) {
	var wrapper struct {
		Data rawSubreddit `json:"data"`
	}
	if err := c.oauthGetJSON("/r/"+name+"/about.json", &wrapper); err != nil {
		return nil, fmt.Errorf("fetching subreddit %s: %w", name, err)
	}
	info := wrapper.Data.toSubredditInfo()
	return &info, nil
}
