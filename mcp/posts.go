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

// SubmitImageInput is the typed input for reddit_submit_image. The
// agent passes an http(s) URL — the engagement-studio backend serves
// chat-attached images via signed URLs at /api/uploads/<id>?sig=...
// and the agent quotes that URL into the message it sends; this tool
// fetches the bytes and walks Reddit's lease + S3 + submit dance.
//
// We deliberately don't accept raw bytes — passing megabytes through
// the JSON-RPC tool envelope makes every transcript huge, and the
// engagement-studio upload endpoint already gives us a stable
// short-lived URL.
type SubmitImageInput struct {
	Subreddit string `json:"subreddit" jsonschema:"description=subreddit name without the r/ prefix (e.g. 'golang'),required"`
	Title     string `json:"title" jsonschema:"description=post title,required"`
	ImageURL  string `json:"image_url" jsonschema:"description=publicly fetchable URL of the image to upload (PNG/JPEG/GIF/WebP). The tool downloads the bytes and re-uploads to Reddit's S3 — DO NOT pass an i.redd.it URL here.,required,format=uri"`
}

func submitImage(_ context.Context, c *reddit.Client, in SubmitImageInput) (any, error) {
	post, err := c.SubmitImageFromURL(in.Subreddit, in.Title, in.ImageURL)
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

// PostInfoInput is the typed input for reddit_post_info.
type PostInfoInput struct {
	ID string `json:"id" jsonschema:"description=Reddit post ID — bare ('abc123') or fullname ('t3_abc123'). Find this in the post URL: reddit.com/r/<sub>/comments/<id>/<slug>,required"`
}

func postInfo(_ context.Context, c *reddit.Client, in PostInfoInput) (any, error) {
	return c.PostInfo(in.ID)
}

// PostsInfoInput is the typed input for reddit_posts_info.
type PostsInfoInput struct {
	IDs []string `json:"ids" jsonschema:"description=Reddit post IDs (bare or t3_-prefixed). Up to 100 per call; the client chunks larger requests automatically.,required"`
}

func postsInfo(_ context.Context, c *reddit.Client, in PostsInfoInput) (any, error) {
	return c.PostsInfo(in.IDs)
}

// PostCommentsInput is the typed input for reddit_post_comments.
type PostCommentsInput struct {
	ID    string `json:"id" jsonschema:"description=Reddit post ID — bare ('abc123') or fullname ('t3_abc123'),required"`
	Sort  string `json:"sort,omitempty" jsonschema:"description=comment sort,enum=confidence,enum=top,enum=new,enum=controversial,enum=old,enum=qa"`
	Limit int    `json:"limit,omitempty" jsonschema:"description=max comments to return (Reddit's max is 500),minimum=1,maximum=500"`
	Depth int    `json:"depth,omitempty" jsonschema:"description=max tree depth; 1 returns only top-level replies,minimum=1,maximum=10"`
}

func postComments(_ context.Context, c *reddit.Client, in PostCommentsInput) (any, error) {
	post, comments, err := c.PostComments(in.ID, &reddit.PostCommentsOptions{
		Sort:  in.Sort,
		Limit: in.Limit,
		Depth: in.Depth,
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"post":     post,
		"comments": comments,
	}, nil
}

// PostInsightsInput is the typed input for reddit_post_insights.
type PostInsightsInput struct {
	ID string `json:"id" jsonschema:"description=Reddit post ID — bare ('abc123') or fullname ('t3_abc123'). Must be a post authored by the authenticated user; insights are owner-only.,required"`
}

func postInsights(_ context.Context, c *reddit.Client, in PostInsightsInput) (any, error) {
	return c.PostInsights(in.ID)
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
	mcptool.Define[*reddit.Client, SubmitImageInput](
		"reddit_submit_image",
		"Create an image post in a subreddit by fetching an image URL and uploading it to Reddit's S3.",
		"SubmitImageFromURL",
		submitImage,
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
	mcptool.Define[*reddit.Client, PostInfoInput](
		"reddit_post_info",
		"Fetch live metrics for one post: score, upvote_ratio, num_comments, view_count, awards.",
		"PostInfo",
		postInfo,
	),
	mcptool.Define[*reddit.Client, PostsInfoInput](
		"reddit_posts_info",
		"Batch-fetch live metrics for up to 100 posts in one request (cheaper than N reddit_post_info calls).",
		"PostsInfo",
		postsInfo,
	),
	mcptool.Define[*reddit.Client, PostCommentsInput](
		"reddit_post_comments",
		"Fetch a post + its comment tree (depth-first). Use to analyze sentiment or read replies.",
		"PostComments",
		postComments,
	),
	mcptool.Define[*reddit.Client, PostInsightsInput](
		"reddit_post_insights",
		"Author-only analytics for OUR post: total views, hourly view chart, shares, top comments, ribbon.",
		"PostInsights",
		postInsights,
	),
}
