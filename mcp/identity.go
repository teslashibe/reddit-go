package mcp

import (
	"context"

	reddit "github.com/teslashibe/reddit-go"
	"github.com/teslashibe/mcptool"
)

// MeInput is the typed input for reddit_me. It takes no arguments.
type MeInput struct{}

func me(_ context.Context, c *reddit.Client, _ MeInput) (any, error) {
	return c.Me()
}

var identityTools = []mcptool.Tool{
	mcptool.Define[*reddit.Client, MeInput](
		"reddit_me",
		"Return the authenticated user's identity (username and ID); use to verify the token is valid",
		"Me",
		me,
	),
}
