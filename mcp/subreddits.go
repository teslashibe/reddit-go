package mcp

import (
	"context"

	reddit "github.com/teslashibe/reddit-go"
	"github.com/teslashibe/mcptool"
)

// SubredditNameInput is the typed input for reddit_subscribe and reddit_unsubscribe.
type SubredditNameInput struct {
	Subreddit string `json:"subreddit" jsonschema:"description=subreddit name without the r/ prefix (e.g. 'golang'),required"`
}

// MySubscriptionsInput is the typed input for reddit_my_subscriptions.
type MySubscriptionsInput struct {
	Limit int `json:"limit,omitempty" jsonschema:"description=max subreddits to return,minimum=1,maximum=100,default=25"`
}

// SubredditAboutInput is the typed input for reddit_subreddit_about.
type SubredditAboutInput struct {
	Name string `json:"name" jsonschema:"description=subreddit name without the r/ prefix (e.g. 'golang'),required"`
}

func subscribe(_ context.Context, c *reddit.Client, in SubredditNameInput) (any, error) {
	if err := c.Subscribe(in.Subreddit); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "subreddit": in.Subreddit, "subscribed": true}, nil
}

func unsubscribe(_ context.Context, c *reddit.Client, in SubredditNameInput) (any, error) {
	if err := c.Unsubscribe(in.Subreddit); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "subreddit": in.Subreddit, "subscribed": false}, nil
}

func mySubscriptions(_ context.Context, c *reddit.Client, in MySubscriptionsInput) (any, error) {
	limit := in.Limit
	if limit <= 0 {
		limit = 25
	}
	res, err := c.MySubscriptions(limit)
	if err != nil {
		return nil, err
	}
	return mcptool.PageOf(res.Subreddits, res.After, limit), nil
}

func subredditAbout(_ context.Context, c *reddit.Client, in SubredditAboutInput) (any, error) {
	return c.SubredditAbout(in.Name)
}

var subredditTools = []mcptool.Tool{
	mcptool.Define[*reddit.Client, SubredditNameInput](
		"reddit_subscribe",
		"Subscribe the authenticated user to a subreddit",
		"Subscribe",
		subscribe,
	),
	mcptool.Define[*reddit.Client, SubredditNameInput](
		"reddit_unsubscribe",
		"Unsubscribe the authenticated user from a subreddit",
		"Unsubscribe",
		unsubscribe,
	),
	mcptool.Define[*reddit.Client, MySubscriptionsInput](
		"reddit_my_subscriptions",
		"List the subreddits the authenticated user is subscribed to",
		"MySubscriptions",
		mySubscriptions,
	),
	mcptool.Define[*reddit.Client, SubredditAboutInput](
		"reddit_subreddit_about",
		"Fetch metadata about a subreddit (subscribers, description, NSFW flag, ...)",
		"SubredditAbout",
		subredditAbout,
	),
}
