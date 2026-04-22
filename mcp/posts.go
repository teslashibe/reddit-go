package mcp

import (
	"context"

	reddit "github.com/teslashibe/reddit-go"
	"github.com/teslashibe/mcptool"
)

// SubmitInput is the typed input for reddit_submit.
type SubmitInput struct {
	Subreddit string `json:"subreddit" jsonschema:"description=subreddit name without the r/ prefix (e.g. 'golang'),required"`
	Title     string `json:"title" jsonschema:"description=post title,required"`
	Body      string `json:"body" jsonschema:"description=post body (Markdown is supported),required"`
}

func submit(_ context.Context, c *reddit.Client, in SubmitInput) (any, error) {
	post, err := c.Submit(in.Subreddit, in.Title, in.Body)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"ok":        true,
		"subreddit": in.Subreddit,
		"id":        post.ID,
		"fullname":  post.Fullname,
		"url":       post.URL,
	}, nil
}

// SubmitLinkInput is the typed input for reddit_submit_link.
type SubmitLinkInput struct {
	Subreddit string `json:"subreddit" jsonschema:"description=subreddit name without the r/ prefix (e.g. 'golang'),required"`
	Title     string `json:"title" jsonschema:"description=post title,required"`
	URL       string `json:"url" jsonschema:"description=link URL to submit,required,format=uri"`
}

func submitLink(_ context.Context, c *reddit.Client, in SubmitLinkInput) (any, error) {
	post, err := c.SubmitLink(in.Subreddit, in.Title, in.URL)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"ok":        true,
		"subreddit": in.Subreddit,
		"id":        post.ID,
		"fullname":  post.Fullname,
		"url":       post.URL,
	}, nil
}

// ReplyInput is the typed input for reddit_reply.
type ReplyInput struct {
	ParentID string `json:"parent_id" jsonschema:"description=Reddit fullname of the parent post (t3_xxx) or comment (t1_xxx) being replied to,required"`
	Body     string `json:"body" jsonschema:"description=plain-text comment body (Markdown is supported),required"`
}

func reply(_ context.Context, c *reddit.Client, in ReplyInput) (any, error) {
	id, err := c.Reply(in.ParentID, in.Body)
	if err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "parent_id": in.ParentID, "comment_id": id}, nil
}

// EditInput is the typed input for reddit_edit.
type EditInput struct {
	Fullname string `json:"fullname" jsonschema:"description=Reddit fullname of the post or comment to edit (must be authored by the authenticated user),required"`
	Body     string `json:"body" jsonschema:"description=new body text (Markdown is supported),required"`
}

func edit(_ context.Context, c *reddit.Client, in EditInput) (any, error) {
	if err := c.Edit(in.Fullname, in.Body); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "fullname": in.Fullname}, nil
}

// DeleteInput is the typed input for reddit_delete.
type DeleteInput struct {
	Fullname string `json:"fullname" jsonschema:"description=Reddit fullname of the post or comment to delete (must be authored by the authenticated user),required"`
}

func deletePost(_ context.Context, c *reddit.Client, in DeleteInput) (any, error) {
	if err := c.Delete(in.Fullname); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "fullname": in.Fullname, "deleted": true}, nil
}

var postTools = []mcptool.Tool{
	mcptool.Define[*reddit.Client, SubmitInput](
		"reddit_submit",
		"Create a self-text post in a subreddit",
		"Submit",
		submit,
	),
	mcptool.Define[*reddit.Client, SubmitLinkInput](
		"reddit_submit_link",
		"Create a link post in a subreddit",
		"SubmitLink",
		submitLink,
	),
	mcptool.Define[*reddit.Client, ReplyInput](
		"reddit_reply",
		"Reply to a post or comment by Reddit fullname (returns the new comment ID)",
		"Reply",
		reply,
	),
	mcptool.Define[*reddit.Client, EditInput](
		"reddit_edit",
		"Edit the body of one of the authenticated user's posts or comments by Reddit fullname",
		"Edit",
		edit,
	),
	mcptool.Define[*reddit.Client, DeleteInput](
		"reddit_delete",
		"Delete one of the authenticated user's posts or comments by Reddit fullname",
		"Delete",
		deletePost,
	),
}
