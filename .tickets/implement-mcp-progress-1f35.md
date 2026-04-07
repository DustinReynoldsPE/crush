---
id: implement-mcp-progress-1f35
stage: triage
deps: []
links: []
created: 2026-04-07T04:40:43Z
type: feature
priority: 2
assignee: Dustin Reynolds
version: 1
---
# Implement MCP progress notification handler


MCP servers can emit progress/notifications during long-running tool calls via NotifyProgress. go-sdk exposes ProgressNotificationHandler on ClientOptions but crush never registers it. Users get no feedback during slow tool executions (file indexing, long computations, etc.).

## Acceptance Criteria

ProgressNotificationHandler registered in mcp/init.go; progress events routed to TUI via pubsub; TUI renders progress bar or status line for in-flight MCP tool calls; handler receives progressToken, progress, total, and message fields
