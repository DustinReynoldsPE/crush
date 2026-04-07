---
id: implement-mcp-logging-dc6e
stage: triage
deps: []
links: []
created: 2026-04-07T04:41:12Z
type: feature
priority: 4
assignee: Dustin Reynolds
version: 1
---
# Implement MCP logging level control


Crush receives log messages from MCP servers via LoggingMessageHandler but never calls SetLoggingLevel to control server verbosity. Servers default to their own log level, which may be too noisy or too quiet. go-sdk exposes SetLoggingLevel on ClientSession.

## Acceptance Criteria

SetLoggingLevel called on each MCP client session after initialization, using debug level when crush --debug is active and warning level otherwise; level adjustable per-server via crush.json mcp config
