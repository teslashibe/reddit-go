package mcp

import (
	"context"

	reddit "github.com/teslashibe/reddit-go"
	"github.com/teslashibe/mcptool"
)

// SearchInput is the typed input for reddit_search.
type SearchInput struct {
	Query string `json:"query" jsonschema:"description=search query string,required"`
	Limit int    `json:"limit,omitempty" jsonschema:"description=max posts to return,minimum=1,maximum=100,default=25"`
}

// SearchInSubredditInput is the typed input for reddit_search_in_subreddit.
type SearchInSubredditInput struct {
	Subreddit string `json:"subreddit" jsonschema:"description=subreddit name without the r/ prefix (e.g. 'golang'),required"`
	Query     string `json:"query" jsonschema:"description=search query string,required"`
	Limit     int    `json:"limit,omitempty" jsonschema:"description=max posts to return,minimum=1,maximum=100,default=25"`
}

func searchAll(_ context.Context, c *reddit.Client, in SearchInput) (any, error) {
	limit := in.Limit
	if limit <= 0 {
		limit = 25
	}
	res, err := c.Search(in.Query, limit)
	if err != nil {
		return nil, err
	}
	return mcptool.PageOf(res.Posts, res.After, limit), nil
}

func searchInSubreddit(_ context.Context, c *reddit.Client, in SearchInSubredditInput) (any, error) {
	limit := in.Limit
	if limit <= 0 {
		limit = 25
	}
	res, err := c.SearchInSubreddit(in.Subreddit, in.Query, limit)
	if err != nil {
		return nil, err
	}
	return mcptool.PageOf(res.Posts, res.After, limit), nil
}

var searchTools = []mcptool.Tool{
	mcptool.Define[*reddit.Client, SearchInput](
		"reddit_search",
		"Search all of Reddit for posts matching the query",
		"Search",
		searchAll,
	),
	mcptool.Define[*reddit.Client, SearchInSubredditInput](
		"reddit_search_in_subreddit",
		"Search within a specific subreddit for posts matching the query",
		"SearchInSubreddit",
		searchInSubreddit,
	),
}
