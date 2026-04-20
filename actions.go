package reddit

import (
	"fmt"
	"net/url"
	"strconv"
)

func (c *Client) vote(fullname string, dir int) error {
	form := url.Values{
		"id":  {fullname},
		"dir": {strconv.Itoa(dir)},
	}
	_, err := c.oauthPost("/api/vote", form)
	if err != nil {
		return fmt.Errorf("voting on %s: %w", fullname, err)
	}
	return nil
}

// Upvote casts an upvote on a post or comment.
func (c *Client) Upvote(fullname string) error {
	return c.vote(fullname, 1)
}

// Downvote casts a downvote on a post or comment.
func (c *Client) Downvote(fullname string) error {
	return c.vote(fullname, -1)
}

// Unvote removes any existing vote on a post or comment.
func (c *Client) Unvote(fullname string) error {
	return c.vote(fullname, 0)
}

// Save bookmarks a post or comment.
func (c *Client) Save(fullname string) error {
	form := url.Values{"id": {fullname}}
	_, err := c.oauthPost("/api/save", form)
	if err != nil {
		return fmt.Errorf("saving %s: %w", fullname, err)
	}
	return nil
}

// Unsave removes a post or comment from saved items.
func (c *Client) Unsave(fullname string) error {
	form := url.Values{"id": {fullname}}
	_, err := c.oauthPost("/api/unsave", form)
	if err != nil {
		return fmt.Errorf("unsaving %s: %w", fullname, err)
	}
	return nil
}

// Hide hides a post from the authenticated user's feed.
func (c *Client) Hide(fullname string) error {
	form := url.Values{"id": {fullname}}
	_, err := c.oauthPost("/api/hide", form)
	if err != nil {
		return fmt.Errorf("hiding %s: %w", fullname, err)
	}
	return nil
}

// Unhide reverses a previous hide action on a post.
func (c *Client) Unhide(fullname string) error {
	form := url.Values{"id": {fullname}}
	_, err := c.oauthPost("/api/unhide", form)
	if err != nil {
		return fmt.Errorf("unhiding %s: %w", fullname, err)
	}
	return nil
}

// Report reports a post or comment for the given reason.
func (c *Client) Report(fullname, reason string) error {
	form := url.Values{
		"thing_id": {fullname},
		"reason":   {reason},
	}
	_, err := c.oauthPost("/api/report", form)
	if err != nil {
		return fmt.Errorf("reporting %s: %w", fullname, err)
	}
	return nil
}
