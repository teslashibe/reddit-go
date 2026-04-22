// Package mcp exposes the reddit-go [reddit.Client] surface as a set of
// MCP (Model Context Protocol) tools that any host application can mount on
// its own MCP server.
//
// All tools wrap exported methods on *reddit.Client. Each tool is defined
// via [mcptool.Define] so the JSON input schema is reflected from the typed
// input struct — no hand-maintained schemas, no drift.
//
// Usage from a host application:
//
//	import (
//	    "github.com/teslashibe/mcptool"
//	    reddit "github.com/teslashibe/reddit-go"
//	    redditmcp "github.com/teslashibe/reddit-go/mcp"
//	)
//
//	client := reddit.New(&reddit.Options{Token: "..."})
//	for _, tool := range redditmcp.Provider{}.Tools() {
//	    // register tool with your MCP server, passing client as the client arg
//	    // when invoking
//	}
//
// The [Excluded] map documents methods on *Client that are intentionally not
// exposed via MCP, with a one-line reason. The coverage test in mcp_test.go
// fails if a new exported method is added without either being wrapped by a
// tool or appearing in [Excluded].
package mcp

import "github.com/teslashibe/mcptool"

// Provider implements [mcptool.Provider] for reddit-go. The zero value is
// ready to use.
type Provider struct{}

// Platform returns "reddit".
func (Provider) Platform() string { return "reddit" }

// Tools returns every reddit-go MCP tool, in registration order.
func (Provider) Tools() []mcptool.Tool {
	out := make([]mcptool.Tool, 0,
		len(identityTools)+
			len(accountTools)+
			len(actionTools)+
			len(chatTools)+
			len(messageTools)+
			len(postTools)+
			len(subredditTools)+
			len(searchTools)+
			len(userTools),
	)
	out = append(out, identityTools...)
	out = append(out, accountTools...)
	out = append(out, actionTools...)
	out = append(out, chatTools...)
	out = append(out, messageTools...)
	out = append(out, postTools...)
	out = append(out, subredditTools...)
	out = append(out, searchTools...)
	out = append(out, userTools...)
	return out
}
