package mcp

import (
	"context"

	reddit "github.com/teslashibe/reddit-go"
	"github.com/teslashibe/mcptool"
)

// PreferencesInput is the typed input for reddit_get_preferences.
type PreferencesInput struct{}

func getPreferences(_ context.Context, c *reddit.Client, _ PreferencesInput) (any, error) {
	return c.Preferences()
}

// TrophiesInput is the typed input for reddit_get_trophies.
type TrophiesInput struct{}

func getTrophies(_ context.Context, c *reddit.Client, _ TrophiesInput) (any, error) {
	return c.Trophies()
}

// FriendsInput is the typed input for reddit_get_friends.
type FriendsInput struct{}

func getFriends(_ context.Context, c *reddit.Client, _ FriendsInput) (any, error) {
	return c.Friends()
}

var accountTools = []mcptool.Tool{
	mcptool.Define[*reddit.Client, PreferencesInput](
		"reddit_get_preferences",
		"Fetch the authenticated user's account preferences as a raw key/value map",
		"Preferences",
		getPreferences,
	),
	mcptool.Define[*reddit.Client, TrophiesInput](
		"reddit_get_trophies",
		"List the authenticated user's Reddit trophies (e.g. Five-Year Club, Verified Email)",
		"Trophies",
		getTrophies,
	),
	mcptool.Define[*reddit.Client, FriendsInput](
		"reddit_get_friends",
		"List the authenticated user's Reddit friends",
		"Friends",
		getFriends,
	),
}
