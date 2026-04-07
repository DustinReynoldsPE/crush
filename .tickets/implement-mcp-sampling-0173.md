---
id: implement-mcp-sampling-0173
stage: triage
deps: []
links: []
created: 2026-04-07T04:40:37Z
type: feature
priority: 1
assignee: Dustin Reynolds
version: 1
---
# Implement MCP sampling (CreateMessage) handler


MCP servers can request crush to generate LLM responses on their behalf via sampling/createMessage. go-sdk exposes CreateMessageHandler and CreateMessageWithToolsHandler on ClientHandler but crush never sets them. This is a bidirectional capability where the server drives LLM calls through the client. Requires routing the request through crush's agent/provider layer.

## Acceptance Criteria

CreateMessageHandler set in mcp/init.go; routes request to current session's LLM provider; respects maxTokens and model preferences from params; returns CreateMessageResult with generated content; capability advertised in ClientInfo
