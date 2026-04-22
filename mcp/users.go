package mcp

import (
	"context"

	reddit "github.com/teslashibe/reddit-go"
	"github.com/teslashibe/mcptool"
)

// MyListingInput is the shared typed input for the read-only "my *" listings
// (posts, comments, overview, saved, upvoted).
type MyListingInput struct {
	Limit int `json:"limit,omitempty" jsonschema:"description=max items to return,minimum=1,maximum=100,default=25"`
}

func myListingLimit(in MyListingInput) int {
	if in.Limit <= 0 {
		return 25
	}
	return in.Limit
}

func myPosts(_ context.Context, c *reddit.Client, in MyListingInput) (any, error) {
	limit := myListingLimit(in)
	res, err := c.MyPosts(limit)
	if err != nil {
		return nil, err
	}
	return mcptool.PageOf(res.Posts, res.After, limit), nil
}

func myComments(_ context.Context, c *reddit.Client, in MyListingInput) (any, error) {
	limit := myListingLimit(in)
	res, err := c.MyComments(limit)
	if err != nil {
		return nil, err
	}
	return mcptool.PageOf(res.Comments, res.After, limit), nil
}

func myOverview(_ context.Context, c *reddit.Client, in MyListingInput) (any, error) {
	limit := myListingLimit(in)
	res, err := c.MyOverview(limit)
	if err != nil {
		return nil, err
	}
	return mcptool.PageOf(res.Items, res.After, limit), nil
}

func saved(_ context.Context, c *reddit.Client, in MyListingInput) (any, error) {
	limit := myListingLimit(in)
	res, err := c.Saved(limit)
	if err != nil {
		return nil, err
	}
	return mcptool.PageOf(res.Items, res.After, limit), nil
}

func upvoted(_ context.Context, c *reddit.Client, in MyListingInput) (any, error) {
	limit := myListingLimit(in)
	res, err := c.Upvoted(limit)
	if err != nil {
		return nil, err
	}
	return mcptool.PageOf(res.Posts, res.After, limit), nil
}

// UserAboutInput is the typed input for reddit_user_about.
type UserAboutInput struct {
	Username string `json:"username" jsonschema:"description=Reddit username without the u/ prefix,required"`
}

func userAbout(_ context.Context, c *reddit.Client, in UserAboutInput) (any, error) {
	return c.UserAbout(in.Username)
}

var userTools = []mcptool.Tool{
	mcptool.Define[*reddit.Client, MyListingInput](
		"reddit_my_posts",
		"List the authenticated user's submitted posts",
		"MyPosts",
		myPosts,
	),
	mcptool.Define[*reddit.Client, MyListingInput](
		"reddit_my_comments",
		"List the authenticated user's comments",
		"MyComments",
		myComments,
	),
	mcptool.Define[*reddit.Client, MyListingInput](
		"reddit_my_overview",
		"List the authenticated user's mixed posts and comments feed",
		"MyOverview",
		myOverview,
	),
	mcptool.Define[*reddit.Client, MyListingInput](
		"reddit_saved",
		"List the authenticated user's saved (bookmarked) posts and comments",
		"Saved",
		saved,
	),
	mcptool.Define[*reddit.Client, MyListingInput](
		"reddit_upvoted",
		"List posts the authenticated user has upvoted",
		"Upvoted",
		upvoted,
	),
	mcptool.Define[*reddit.Client, UserAboutInput](
		"reddit_user_about",
		"Fetch public profile information for any Reddit user by username",
		"UserAbout",
		userAbout,
	),
}
