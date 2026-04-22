package mcp

import (
	"context"

	reddit "github.com/teslashibe/reddit-go"
	"github.com/teslashibe/mcptool"
)

// FullnameInput is the shared typed input for thing-targeted writes.
// Reddit "fullnames" look like t1_xxx (comment) or t3_xxx (post).
type FullnameInput struct {
	Fullname string `json:"fullname" jsonschema:"description=Reddit fullname (e.g. t3_abc123 for a post or t1_def456 for a comment),required"`
}

// ReportInput is the typed input for reddit_report.
type ReportInput struct {
	Fullname string `json:"fullname" jsonschema:"description=Reddit fullname of the post or comment to report (e.g. t3_abc123),required"`
	Reason   string `json:"reason" jsonschema:"description=human-readable report reason shown to subreddit moderators,required"`
}

func upvote(_ context.Context, c *reddit.Client, in FullnameInput) (any, error) {
	if err := c.Upvote(in.Fullname); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "fullname": in.Fullname, "direction": 1}, nil
}

func downvote(_ context.Context, c *reddit.Client, in FullnameInput) (any, error) {
	if err := c.Downvote(in.Fullname); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "fullname": in.Fullname, "direction": -1}, nil
}

func unvote(_ context.Context, c *reddit.Client, in FullnameInput) (any, error) {
	if err := c.Unvote(in.Fullname); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "fullname": in.Fullname, "direction": 0}, nil
}

func save(_ context.Context, c *reddit.Client, in FullnameInput) (any, error) {
	if err := c.Save(in.Fullname); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "fullname": in.Fullname, "saved": true}, nil
}

func unsave(_ context.Context, c *reddit.Client, in FullnameInput) (any, error) {
	if err := c.Unsave(in.Fullname); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "fullname": in.Fullname, "saved": false}, nil
}

func hide(_ context.Context, c *reddit.Client, in FullnameInput) (any, error) {
	if err := c.Hide(in.Fullname); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "fullname": in.Fullname, "hidden": true}, nil
}

func unhide(_ context.Context, c *reddit.Client, in FullnameInput) (any, error) {
	if err := c.Unhide(in.Fullname); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "fullname": in.Fullname, "hidden": false}, nil
}

func report(_ context.Context, c *reddit.Client, in ReportInput) (any, error) {
	if err := c.Report(in.Fullname, in.Reason); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "fullname": in.Fullname, "reason": in.Reason}, nil
}

var actionTools = []mcptool.Tool{
	mcptool.Define[*reddit.Client, FullnameInput](
		"reddit_upvote",
		"Cast an upvote on a post or comment by Reddit fullname (t3_xxx or t1_xxx)",
		"Upvote",
		upvote,
	),
	mcptool.Define[*reddit.Client, FullnameInput](
		"reddit_downvote",
		"Cast a downvote on a post or comment by Reddit fullname (t3_xxx or t1_xxx)",
		"Downvote",
		downvote,
	),
	mcptool.Define[*reddit.Client, FullnameInput](
		"reddit_unvote",
		"Remove any existing vote on a post or comment by Reddit fullname",
		"Unvote",
		unvote,
	),
	mcptool.Define[*reddit.Client, FullnameInput](
		"reddit_save",
		"Bookmark (save) a post or comment by Reddit fullname",
		"Save",
		save,
	),
	mcptool.Define[*reddit.Client, FullnameInput](
		"reddit_unsave",
		"Remove a post or comment from saved items by Reddit fullname",
		"Unsave",
		unsave,
	),
	mcptool.Define[*reddit.Client, FullnameInput](
		"reddit_hide",
		"Hide a post from the authenticated user's feed by Reddit fullname",
		"Hide",
		hide,
	),
	mcptool.Define[*reddit.Client, FullnameInput](
		"reddit_unhide",
		"Reverse a previous hide on a post by Reddit fullname",
		"Unhide",
		unhide,
	),
	mcptool.Define[*reddit.Client, ReportInput](
		"reddit_report",
		"Report a post or comment to subreddit moderators with a human-readable reason",
		"Report",
		report,
	),
}
