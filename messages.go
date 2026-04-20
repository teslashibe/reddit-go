package redditmessenger

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

func (m *Messenger) fetchMessages(endpoint string, limit int, after string) (*MessageListing, error) {
	path := endpoint + "?raw_json=1"
	if limit > 0 {
		path += "&limit=" + strconv.Itoa(limit)
	}
	if after != "" {
		path += "&after=" + after
	}

	body, err := m.oauthGet(path)
	if err != nil {
		return nil, err
	}

	var listing redditListing
	if err := json.Unmarshal(body, &listing); err != nil {
		return nil, fmt.Errorf("decoding message listing: %w", err)
	}

	result := &MessageListing{
		After:  listing.Data.After,
		Before: listing.Data.Before,
	}
	for _, child := range listing.Data.Children {
		result.Messages = append(result.Messages, child.Data.toMessage())
	}
	return result, nil
}

// Inbox returns the user's inbox (PMs + comment replies).
// Limit of 0 returns Reddit's default (25). Max is 100.
func (m *Messenger) Inbox(limit int) (*MessageListing, error) {
	return m.fetchMessages("/message/inbox.json", limit, "")
}

// InboxAfter returns the next page of inbox messages after the given cursor.
func (m *Messenger) InboxAfter(limit int, after string) (*MessageListing, error) {
	return m.fetchMessages("/message/inbox.json", limit, after)
}

// Messages returns only private messages (no comment replies).
func (m *Messenger) Messages(limit int) (*MessageListing, error) {
	return m.fetchMessages("/message/messages.json", limit, "")
}

// MessagesAfter returns the next page of PMs after the given cursor.
func (m *Messenger) MessagesAfter(limit int, after string) (*MessageListing, error) {
	return m.fetchMessages("/message/messages.json", limit, after)
}

// Sent returns messages sent by the authenticated user.
func (m *Messenger) Sent(limit int) (*MessageListing, error) {
	return m.fetchMessages("/message/sent.json", limit, "")
}

// SentAfter returns the next page of sent messages after the given cursor.
func (m *Messenger) SentAfter(limit int, after string) (*MessageListing, error) {
	return m.fetchMessages("/message/sent.json", limit, after)
}

// Unread returns unread inbox items.
func (m *Messenger) Unread(limit int) (*MessageListing, error) {
	return m.fetchMessages("/message/unread.json", limit, "")
}

// Compose sends a private message to a Reddit user.
func (m *Messenger) Compose(to, subject, body string) error {
	form := url.Values{
		"api_type": {"json"},
		"to":       {to},
		"subject":  {subject},
		"text":     {body},
	}

	respBody, err := m.oauthPost("/api/compose", form)
	if err != nil {
		return err
	}

	var result struct {
		JSON struct {
			Errors [][]string `json:"errors"`
		} `json:"json"`
	}
	if err := json.Unmarshal(respBody, &result); err == nil && len(result.JSON.Errors) > 0 {
		return fmt.Errorf("compose failed: %v", result.JSON.Errors)
	}

	return nil
}

// MarkRead marks one or more messages as read.
// Pass fullnames like "t1_abc123" or "t4_abc123".
func (m *Messenger) MarkRead(fullnames ...string) error {
	if len(fullnames) == 0 {
		return nil
	}
	form := url.Values{
		"id": {joinComma(fullnames)},
	}
	_, err := m.oauthPost("/api/read_message", form)
	return err
}

// MarkUnread marks one or more messages as unread.
func (m *Messenger) MarkUnread(fullnames ...string) error {
	if len(fullnames) == 0 {
		return nil
	}
	form := url.Values{
		"id": {joinComma(fullnames)},
	}
	_, err := m.oauthPost("/api/unread_message", form)
	return err
}

func joinComma(ss []string) string {
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += ","
		}
		result += s
	}
	return result
}
