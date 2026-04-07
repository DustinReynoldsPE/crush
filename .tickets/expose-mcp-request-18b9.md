---
id: expose-mcp-request-18b9
stage: triage
deps: []
links: []
created: 2026-04-07T04:41:07Z
type: feature
priority: 3
assignee: Dustin Reynolds
version: 1
---
# Expose MCP request cancellation for in-flight tool calls


go-sdk handles the notifications/cancelled protocol internally but crush never exposes a cancel path for in-flight MCP tool calls. When the user interrupts crush mid-execution, MCP tool calls run to completion on the server side rather than being aborted. The context passed to CallTool is cancelled but the cancellation notification is not sent to the server.

## Acceptance Criteria

Context cancellation during CallTool propagates a notifications/cancelled message to the MCP server; server-side tool execution is aborted where supported; crush handles CancelledNotification from servers gracefully; no orphaned server-side processes on user interrupt
