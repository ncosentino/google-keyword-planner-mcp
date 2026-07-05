package main

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// toolArrayFields declares, for each tool name, which top-level argument fields
// are array-typed. coerceStringifiedArrayArgs uses this to know which fields to
// repair; it is intentionally a plain data map (not per-tool duplicated logic),
// so every tool with a keyword-list parameter is covered by one code path.
var toolArrayFields = map[string][]string{
	"generate_keyword_ideas": {"seed_keywords"},
	"get_historical_metrics": {"keywords"},
	"get_keyword_forecast":   {"keywords"},
}

// coerceStringifiedArrayArgs returns a receiving middleware that repairs a
// widespread MCP client bug: some clients JSON-encode an array-typed tool
// argument as a string (e.g. `"[\"a\",\"b\"]"` instead of `["a","b"]`) before
// sending it. That failure mode has been reproduced against multiple, unrelated
// MCP servers and confirmed independent of this server's schema shape (see
// google-keyword-planner-mcp#4), so it is repaired defensively here rather than
// left to reject valid calls.
//
// This must run as middleware, not inside a tool handler: the SDK validates
// arguments against the tool's JSON Schema before a registered handler is ever
// invoked, so a malformed argument never reaches the handler to be fixed there.
// Middleware runs earlier, while the arguments are still raw JSON.
func coerceStringifiedArrayArgs(arrayFieldsByTool map[string][]string) mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			call, ok := req.(*mcp.CallToolRequest)
			if !ok || method != "tools/call" {
				return next(ctx, method, req)
			}
			fields := arrayFieldsByTool[call.Params.Name]
			if len(fields) == 0 || len(call.Params.Arguments) == 0 {
				return next(ctx, method, req)
			}

			var args map[string]json.RawMessage
			if err := json.Unmarshal(call.Params.Arguments, &args); err != nil {
				// Malformed JSON entirely; let normal validation surface the error.
				return next(ctx, method, req)
			}

			changed := false
			for _, field := range fields {
				if coerced, ok := coerceStringifiedArray(args[field]); ok {
					args[field] = coerced
					changed = true
				}
			}
			if changed {
				if rewritten, err := json.Marshal(args); err == nil {
					call.Params.Arguments = rewritten
				}
			}
			return next(ctx, method, req)
		}
	}
}

// coerceStringifiedArray reports whether raw is a JSON string that itself
// decodes to a JSON array, returning that array's raw JSON if so. It returns
// ok=false for a raw value that is missing, already an array, or a string that
// doesn't decode to an array -- in all of those cases the caller should leave
// the value untouched and let normal schema validation handle it.
func coerceStringifiedArray(raw json.RawMessage) (coerced json.RawMessage, ok bool) {
	if len(raw) == 0 {
		return nil, false
	}
	var asString string
	if err := json.Unmarshal(raw, &asString); err != nil {
		return nil, false // not a JSON string (e.g. already an array); leave as-is.
	}
	var probe []json.RawMessage
	if err := json.Unmarshal([]byte(asString), &probe); err != nil {
		return nil, false // string doesn't decode to a JSON array; leave for validation to reject.
	}
	return json.RawMessage(asString), true
}
