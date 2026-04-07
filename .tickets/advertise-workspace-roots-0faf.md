---
id: advertise-workspace-roots-0faf
stage: triage
deps: []
links: []
created: 2026-04-07T04:40:55Z
type: feature
priority: 2
assignee: Dustin Reynolds
version: 1
---
# Advertise workspace roots to MCP servers


MCP servers can request the client's filesystem roots to understand file access scope. go-sdk exposes AddRoots on the Client but crush never calls it. MCP servers that use roots to scope file operations or security policies cannot function correctly without this.

## Acceptance Criteria

crush calls AddRoots on each MCP client with the configured working directory and any additional roots from config; roots updated when working directory changes; servers receive roots/list response with correct paths
