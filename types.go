package redditmessenger

import "time"

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

	// WasComment is true when this inbox item is a comment reply,
	// not a private message.
	WasComment bool `json:"was_comment"`
}

// MessageListing is a paginated list of messages from the Reddit API.
type MessageListing struct {
	Messages []Message
	After    string
	Before   string
}

// redditListing is the raw Reddit API listing wrapper.
type redditListing struct {
	Kind string `json:"kind"`
	Data struct {
		After    string        `json:"after"`
		Before   string        `json:"before"`
		Children []redditChild `json:"children"`
	} `json:"data"`
}

type redditChild struct {
	Kind string      `json:"kind"`
	Data rawMessage  `json:"data"`
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
		BodyHTML:    r.BodyHTML,
		Created:    time.Unix(int64(r.CreatedUTC), 0),
		IsNew:      r.New,
		Type:       r.Type,
		Context:    r.Context,
		Subreddit:  r.Subreddit,
		ParentID:   r.ParentID,
		WasComment: r.WasComment,
	}
}
