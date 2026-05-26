package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	reddit "github.com/teslashibe/reddit-go"
	redditmcp "github.com/teslashibe/reddit-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
)

func main() {
	token := os.Getenv("REDDIT_TOKEN")
	if token == "" {
		log.Fatal("REDDIT_TOKEN environment variable required (token_v2 cookie value)")
	}

	client := reddit.New(&reddit.Options{Token: token})

	s := server.NewMCPServer(
		"reddit-mcp",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	provider := redditmcp.Provider{}
	for _, tool := range provider.Tools() {
		t := tool
		mcpTool := mcpgo.Tool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: toInputSchema(t.InputSchema),
		}
		s.AddTool(mcpTool, func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
			raw, err := json.Marshal(req.Params.Arguments)
			if err != nil {
				return nil, fmt.Errorf("marshal args: %w", err)
			}
			result, toolErr := t.Invoke(ctx, client, raw)
			if toolErr != nil {
				return nil, fmt.Errorf("tool error: %w", toolErr)
			}
			out, err := json.Marshal(result)
			if err != nil {
				return nil, fmt.Errorf("marshal result: %w", err)
			}
			return mcpgo.NewToolResultText(string(out)), nil
		})
	}

	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

func toInputSchema(raw map[string]any) mcpgo.ToolInputSchema {
	schema := mcpgo.ToolInputSchema{
		Type: "object",
	}
	if props, ok := raw["properties"]; ok {
		if m, ok := props.(map[string]any); ok {
			schema.Properties = m
		}
	}
	if req, ok := raw["required"]; ok {
		if arr, ok := req.([]any); ok {
			strs := make([]string, len(arr))
			for i, v := range arr {
				strs[i] = fmt.Sprint(v)
			}
			schema.Required = strs
		}
	}
	return schema
}
