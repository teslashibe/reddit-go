package mcp

// Excluded enumerates exported methods on *reddit.Client that are
// intentionally not exposed via MCP. Each entry must have a non-empty reason.
//
// The coverage test in mcp_test.go fails if any exported method on *Client is
// neither wrapped by a Tool nor present in this map (or vice-versa: if an
// entry here doesn't correspond to a real method).
//
// When the underlying client gains a new method:
//   - prefer to add an MCP tool for it (see auth.go / posts.go / etc.)
//   - if the method is unsuitable for an agent (internal observability,
//     auth-only helper, etc.), add it here with a reason
var Excluded = map[string]string{
	"RateLimit":    "internal observability; surfaced via the host application's MCP middleware, not as a callable tool",
	"AuthSnapshot": "auth-only helper for the host to persist a minted session; not an agent-callable action",
	"HealthCheck":  "host-side liveness probe (chat-aware) for connection status; not an agent-callable action",
	// Image upload exposes a single MCP entry point — reddit_submit_image
	// (SubmitImageFromURL). The other helpers are deliberately Go-only:
	// SubmitImage takes an already-uploaded S3 URL (an internal artifact
	// the agent never sees), SubmitImageFromFile / UploadMediaFromFile
	// take a local filesystem path that the agent has no concept of, and
	// UploadMedia takes raw bytes that don't fit through the JSON-RPC
	// tool envelope.
	"SubmitImage":         "internal helper; agent uses SubmitImageFromURL via reddit_submit_image",
	"SubmitImageFromFile": "filesystem path not meaningful to a remote agent; use reddit_submit_image with a URL",
	"UploadMedia":         "raw-bytes API not suitable for JSON-RPC tool envelopes; use reddit_submit_image",
	"UploadMediaFromFile": "filesystem path not meaningful to a remote agent; use reddit_submit_image",
}
