package mcp

// Excluded enumerates exported methods on *hn.Client that are
// intentionally not exposed via MCP. Each entry must have a non-empty reason.
//
// The coverage test in mcp_test.go fails if any exported method on *Client is
// neither wrapped by a Tool nor present in this map (or vice-versa: if an
// entry here doesn't correspond to a real method).
//
// When the underlying client gains a new method:
//   - prefer to add an MCP tool for it (see firebase.go / feed.go / etc.)
//   - if the method is unsuitable for an agent (internal observability,
//     auth-only helper, etc.), add it here with a reason
var Excluded = map[string]string{
	"RateLimit":    "internal observability; surfaced via the host application's MCP middleware, not as a callable tool",
	"AuthSnapshot": "auth-only helper; returns the session cookie for the host to persist, which an agent must never see",
	"GetMe":        "auth-only helper; the host liveness prober uses it to check the session, not an agent action",
}
