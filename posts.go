package reddit

import (
	"encoding/json"
	"fmt"
	"net/url"
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
			Data struct {
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
