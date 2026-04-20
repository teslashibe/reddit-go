package reddit

import (
	"encoding/json"
	"fmt"
	"time"
)

// Preferences returns the authenticated user's account preferences as a raw map.
func (c *Client) Preferences() (map[string]interface{}, error) {
	var prefs map[string]interface{}
	if err := c.oauthGetJSON("/api/v1/me/prefs", &prefs); err != nil {
		return nil, fmt.Errorf("fetching preferences: %w", err)
	}
	return prefs, nil
}

// Trophies returns the authenticated user's trophy list.
func (c *Client) Trophies() ([]Trophy, error) {
	body, err := c.oauthGet("/api/v1/me/trophies")
	if err != nil {
		return nil, fmt.Errorf("fetching trophies: %w", err)
	}

	var wrapper struct {
		Data struct {
			Trophies []struct {
				Data Trophy `json:"data"`
			} `json:"trophies"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &wrapper); err != nil {
		return nil, fmt.Errorf("parsing trophies: %w", err)
	}

	trophies := make([]Trophy, len(wrapper.Data.Trophies))
	for i, t := range wrapper.Data.Trophies {
		trophies[i] = t.Data
	}
	return trophies, nil
}

// Friends returns the authenticated user's friend list.
func (c *Client) Friends() ([]Friend, error) {
	body, err := c.oauthGet("/api/v1/me/friends")
	if err != nil {
		return nil, fmt.Errorf("fetching friends: %w", err)
	}

	var wrapper struct {
		Data struct {
			Children []struct {
				Name string  `json:"name"`
				ID   string  `json:"id"`
				Date float64 `json:"date"`
			} `json:"children"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &wrapper); err != nil {
		return nil, fmt.Errorf("parsing friends: %w", err)
	}

	friends := make([]Friend, len(wrapper.Data.Children))
	for i, f := range wrapper.Data.Children {
		friends[i] = Friend{
			Name: f.Name,
			ID:   f.ID,
			Date: time.Unix(int64(f.Date), 0),
		}
	}
	return friends, nil
}
