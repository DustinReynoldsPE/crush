---
id: add-elicitation-elicitationresult-9a18
stage: triage
deps: [implement-mcp-elicitation-bdd1]
links: []
created: 2026-04-07T03:18:44Z
type: feature
priority: 3
assignee: Dustin Reynolds
version: 2
---
# Add Elicitation and ElicitationResult hook events

Fire Elicitation when an MCP server requests user input during a tool call, and ElicitationResult after the user responds before the response is sent back. Enables intercepting, logging, or pre-filling MCP elicitation dialogs.

## Context

go-sdk v1.4.1 already supports elicitation. `mcp.ClientHandler.ElicitationHandler` is the callback crush needs to set when initializing MCP clients in `internal/agent/tools/mcp/init.go`. The handler receives `*mcp.ElicitParams` and must return `*mcp.ElicitResult`. There is no work needed in fantasy or catwalk.

## Scope
- Add `Elicitation` and `ElicitationResult` to `HookEventName` constants in `internal/hooks/types.go`
- In `internal/agent/tools/mcp/init.go`, set `ClientHandler.ElicitationHandler` when building each MCP client
- The handler should: fire `Elicitation` hook async, surface a TUI prompt for user input (or auto-respond if a hook modifies the response), then fire `ElicitationResult` hook async before returning
- Matcher can filter by MCP server name via existing matcher pattern
- Payload: `session_id`, `data.server` (MCP server name), `data.prompt` (message from server), `data.schema` (JSON schema of expected input)
- `ElicitationResult` payload additionally includes `data.response` (the user's answer)

## Acceptance Criteria
- `Elicitation` hook fires when an MCP server sends `elicitation/create`
- `ElicitationResult` hook fires after user responds, before result is returned to server
- Both are async (non-blocking to hook result)
- Payload includes `session_id`, `data.server`, `data.prompt`
- `ElicitationResult` includes `data.response`
- TUI renders the elicitation prompt and accepts user input
- No handler registered → server receives an empty/cancelled result gracefully
