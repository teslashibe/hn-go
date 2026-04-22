// Package mcp exposes the hn-go [hn.Client] surface as a set of MCP
// (Model Context Protocol) tools that any host application can mount on
// its own MCP server.
//
// All tools wrap exported methods on *hn.Client. Each tool is defined
// via [mcptool.Define] so the JSON input schema is reflected from the
// typed input struct — no hand-maintained schemas, no drift.
//
// Usage from a host application:
//
//	import (
//	    "github.com/teslashibe/mcptool"
//	    hn "github.com/teslashibe/hn-go"
//	    hnmcp "github.com/teslashibe/hn-go/mcp"
//	)
//
//	client, _ := hn.New("user&token")
//	for _, tool := range hnmcp.Provider{}.Tools() {
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

// Provider implements [mcptool.Provider] for hn-go. The zero value is
// ready to use.
type Provider struct{}

// Platform returns "hn".
func (Provider) Platform() string { return "hn" }

// Tools returns every hn-go MCP tool, in registration order.
func (Provider) Tools() []mcptool.Tool {
	out := make([]mcptool.Tool, 0,
		len(firebaseTools)+len(feedTools)+len(searchTools)+len(actionTools))
	out = append(out, firebaseTools...)
	out = append(out, feedTools...)
	out = append(out, searchTools...)
	out = append(out, actionTools...)
	return out
}
