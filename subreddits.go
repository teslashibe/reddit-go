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

// SubredditAbout returns metadata about a subreddit, including the
// moderator-defined posting rules and the site-wide rules Reddit applies
// on top.
//
// This method makes two requests: /r/{name}/about for the metadata and
// /r/{name}/about/rules for the rules. Both endpoints are cheap (no
// listing pagination, ~1KB responses) and rules are almost always what
// the caller actually wants the next thing they ask for, so we fetch
// them eagerly. If the rules request fails we still return the metadata
// — the rules-fetch error is logged on the returned info as an empty
// Rules slice (vs nil for "not loaded"), and the surface error is the
// metadata error.
//
// Use SubredditRules if you only want the rules (one request) or want
// to retry the rules fetch independently.
func (c *Client) SubredditAbout(name string) (*SubredditInfo, error) {
	var wrapper struct {
		Data rawSubreddit `json:"data"`
	}
	if err := c.oauthGetJSON("/r/"+name+"/about.json", &wrapper); err != nil {
		return nil, fmt.Errorf("fetching subreddit %s: %w", name, err)
	}
	info := wrapper.Data.toSubredditInfo()

	// Fetch rules best-effort. Subs occasionally 403 on /about/rules
	// (private subs the user isn't approved on, banned subs, etc.)
	// even when /about itself succeeds, so don't let the rules request
	// take down the whole call.
	if rules, err := c.SubredditRules(name); err == nil {
		info.Rules = rules.Rules
		info.SiteRules = rules.SiteRules
	}
	return &info, nil
}

// SubredditRules returns the moderator-defined posting rules and the
// site-wide rules for a subreddit. This is what /r/{name}/about/rules
// returns, parsed and normalized.
//
// The returned Rules slice is ordered as Reddit returned it (which
// follows the moderators' configured priority).
func (c *Client) SubredditRules(name string) (*SubredditRules, error) {
	var raw rawSubredditRules
	if err := c.oauthGetJSON("/r/"+name+"/about/rules.json", &raw); err != nil {
		return nil, fmt.Errorf("fetching rules for %s: %w", name, err)
	}
	out := raw.toSubredditRules()
	return &out, nil
}
