package mcp

import (
	"context"

	reddit "github.com/teslashibe/reddit-go"
	"github.com/teslashibe/mcptool"
)

// MessageListInput is the typed input for the read-only message list tools
// that don't take a pagination cursor (Inbox, Messages, Sent, Unread,
// CommentReplies, PostReplies).
type MessageListInput struct {
	Limit int `json:"limit,omitempty" jsonschema:"description=max items to return,minimum=1,maximum=100,default=25"`
}

// MessageListAfterInput is the typed input for the paginated message list
// tools (InboxAfter, MessagesAfter, SentAfter).
type MessageListAfterInput struct {
	After string `json:"after" jsonschema:"description=pagination cursor returned by the previous page (the After field on a MessageListing),required"`
	Limit int    `json:"limit,omitempty" jsonschema:"description=max items to return,minimum=1,maximum=100,default=25"`
}

func messageLimit(in MessageListInput) int {
	if in.Limit <= 0 {
		return 25
	}
	return in.Limit
}

func messageLimitAfter(in MessageListAfterInput) int {
	if in.Limit <= 0 {
		return 25
	}
	return in.Limit
}

func inbox(_ context.Context, c *reddit.Client, in MessageListInput) (any, error) {
	limit := messageLimit(in)
	res, err := c.Inbox(limit)
	if err != nil {
		return nil, err
	}
	return mcptool.PageOf(res.Messages, res.After, limit), nil
}

func inboxAfter(_ context.Context, c *reddit.Client, in MessageListAfterInput) (any, error) {
	limit := messageLimitAfter(in)
	res, err := c.InboxAfter(limit, in.After)
	if err != nil {
		return nil, err
	}
	return mcptool.PageOf(res.Messages, res.After, limit), nil
}

func messages(_ context.Context, c *reddit.Client, in MessageListInput) (any, error) {
	limit := messageLimit(in)
	res, err := c.Messages(limit)
	if err != nil {
		return nil, err
	}
	return mcptool.PageOf(res.Messages, res.After, limit), nil
}

func messagesAfter(_ context.Context, c *reddit.Client, in MessageListAfterInput) (any, error) {
	limit := messageLimitAfter(in)
	res, err := c.MessagesAfter(limit, in.After)
	if err != nil {
		return nil, err
	}
	return mcptool.PageOf(res.Messages, res.After, limit), nil
}

func sent(_ context.Context, c *reddit.Client, in MessageListInput) (any, error) {
	limit := messageLimit(in)
	res, err := c.Sent(limit)
	if err != nil {
		return nil, err
	}
	return mcptool.PageOf(res.Messages, res.After, limit), nil
}

func sentAfter(_ context.Context, c *reddit.Client, in MessageListAfterInput) (any, error) {
	limit := messageLimitAfter(in)
	res, err := c.SentAfter(limit, in.After)
	if err != nil {
		return nil, err
	}
	return mcptool.PageOf(res.Messages, res.After, limit), nil
}

func unread(_ context.Context, c *reddit.Client, in MessageListInput) (any, error) {
	limit := messageLimit(in)
	res, err := c.Unread(limit)
	if err != nil {
		return nil, err
	}
	return mcptool.PageOf(res.Messages, res.After, limit), nil
}

func commentReplies(_ context.Context, c *reddit.Client, in MessageListInput) (any, error) {
	limit := messageLimit(in)
	res, err := c.CommentReplies(limit)
	if err != nil {
		return nil, err
	}
	return mcptool.PageOf(res.Messages, res.After, limit), nil
}

func postReplies(_ context.Context, c *reddit.Client, in MessageListInput) (any, error) {
	limit := messageLimit(in)
	res, err := c.PostReplies(limit)
	if err != nil {
		return nil, err
	}
	return mcptool.PageOf(res.Messages, res.After, limit), nil
}

// ComposeInput is the typed input for reddit_compose.
type ComposeInput struct {
	To      string `json:"to" jsonschema:"description=recipient Reddit username (no u/ prefix),required"`
	Subject string `json:"subject" jsonschema:"description=message subject line,required"`
	Body    string `json:"body" jsonschema:"description=plain-text message body (Markdown is supported),required"`
}

func compose(_ context.Context, c *reddit.Client, in ComposeInput) (any, error) {
	if err := c.Compose(in.To, in.Subject, in.Body); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "to": in.To, "subject": in.Subject}, nil
}

// MarkReadInput is the typed input for reddit_mark_read and reddit_mark_unread.
type MarkReadInput struct {
	Fullnames []string `json:"fullnames" jsonschema:"description=Reddit fullnames of inbox items (e.g. t1_xxx for comment replies, t4_xxx for PMs),required"`
}

func markRead(_ context.Context, c *reddit.Client, in MarkReadInput) (any, error) {
	if err := c.MarkRead(in.Fullnames...); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "fullnames": in.Fullnames, "read": true}, nil
}

func markUnread(_ context.Context, c *reddit.Client, in MarkReadInput) (any, error) {
	if err := c.MarkUnread(in.Fullnames...); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "fullnames": in.Fullnames, "read": false}, nil
}

var messageTools = []mcptool.Tool{
	mcptool.Define[*reddit.Client, MessageListInput](
		"reddit_inbox",
		"List the authenticated user's inbox (private messages and comment replies)",
		"Inbox",
		inbox,
	),
	mcptool.Define[*reddit.Client, MessageListAfterInput](
		"reddit_inbox_after",
		"Fetch the next page of inbox items using a pagination cursor",
		"InboxAfter",
		inboxAfter,
	),
	mcptool.Define[*reddit.Client, MessageListInput](
		"reddit_messages",
		"List private messages only (excluding comment replies) for the authenticated user",
		"Messages",
		messages,
	),
	mcptool.Define[*reddit.Client, MessageListAfterInput](
		"reddit_messages_after",
		"Fetch the next page of private messages using a pagination cursor",
		"MessagesAfter",
		messagesAfter,
	),
	mcptool.Define[*reddit.Client, MessageListInput](
		"reddit_sent",
		"List messages sent by the authenticated user",
		"Sent",
		sent,
	),
	mcptool.Define[*reddit.Client, MessageListAfterInput](
		"reddit_sent_after",
		"Fetch the next page of sent messages using a pagination cursor",
		"SentAfter",
		sentAfter,
	),
	mcptool.Define[*reddit.Client, MessageListInput](
		"reddit_unread",
		"List unread inbox items for the authenticated user",
		"Unread",
		unread,
	),
	mcptool.Define[*reddit.Client, MessageListInput](
		"reddit_comment_replies",
		"List replies to the authenticated user's comments",
		"CommentReplies",
		commentReplies,
	),
	mcptool.Define[*reddit.Client, MessageListInput](
		"reddit_post_replies",
		"List replies to the authenticated user's posts",
		"PostReplies",
		postReplies,
	),
	mcptool.Define[*reddit.Client, ComposeInput](
		"reddit_compose",
		"Send a private message (PM) to a Reddit user",
		"Compose",
		compose,
	),
	mcptool.Define[*reddit.Client, MarkReadInput](
		"reddit_mark_read",
		"Mark one or more inbox items as read by Reddit fullname",
		"MarkRead",
		markRead,
	),
	mcptool.Define[*reddit.Client, MarkReadInput](
		"reddit_mark_unread",
		"Mark one or more inbox items as unread by Reddit fullname",
		"MarkUnread",
		markUnread,
	),
}
