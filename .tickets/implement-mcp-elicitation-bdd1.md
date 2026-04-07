---
id: implement-mcp-elicitation-bdd1
stage: triage
deps: []
links: []
created: 2026-04-07T04:40:31Z
type: feature
priority: 1
assignee: Dustin Reynolds
version: 1
---
# Implement MCP elicitation handler for user input


MCP servers can send elicitation/create requests mid-tool-execution to collect user input (form or URL mode). go-sdk v1.4.1 supports this via ClientHandler.ElicitationHandler but crush never sets it. Without it, MCP servers requiring user auth or input cannot function.

## Acceptance Criteria

ClientHandler.ElicitationHandler set in internal/agent/tools/mcp/init.go; form-mode renders JSON schema fields as TUI prompt; URL-mode opens URL for OAuth-style flow; response returned to server; Elicitation and ElicitationResult hooks fire async; no handler = graceful cancel returned to server
