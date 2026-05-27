package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

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

	opts := &reddit.Options{Token: token}
	if cookies := loadCookies(); cookies != nil {
		opts.Cookies = cookies
	}
	client := reddit.New(opts)

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

// loadCookies tries, in order:
//  1. REDDIT_COOKIES env var (JSON object)
//  2. ~/.config/cookie-sync/reddit.json (written by the Chrome extension)
//
// Returns nil if neither source has cookies.
func loadCookies() map[string]string {
	if raw := os.Getenv("REDDIT_COOKIES"); raw != "" {
		var cookies map[string]string
		if err := json.Unmarshal([]byte(raw), &cookies); err != nil {
			log.Fatalf("REDDIT_COOKIES is not valid JSON: %v", err)
		}
		log.Println("[cookie-sync] loaded cookies from REDDIT_COOKIES env var")
		return cookies
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	path := filepath.Join(home, ".config", "cookie-sync", "reddit.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var synced struct {
		Cookies map[string]string `json:"cookies"`
	}
	if err := json.Unmarshal(data, &synced); err != nil {
		log.Printf("[cookie-sync] warning: %s is not valid JSON: %v", path, err)
		return nil
	}
	if len(synced.Cookies) == 0 {
		return nil
	}
	log.Printf("[cookie-sync] loaded %d cookies from %s", len(synced.Cookies), path)
	return synced.Cookies
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
