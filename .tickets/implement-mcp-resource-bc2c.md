---
id: implement-mcp-resource-bc2c
stage: triage
deps: []
links: []
created: 2026-04-07T04:40:49Z
type: feature
priority: 2
assignee: Dustin Reynolds
version: 1
---
# Implement MCP resource subscriptions


Crush caches MCP resources at list time but never subscribes to change notifications. go-sdk exposes Subscribe/Unsubscribe on ClientSession and ResourceUpdatedHandler on ClientOptions. Without subscriptions, cached resource data goes stale when the MCP server updates a resource.

## Acceptance Criteria

ResourceUpdatedHandler registered in mcp/init.go; Subscribe called for resources that clients have read; Unsubscribe called on session teardown; updated resource triggers cache invalidation and re-read; ResourceListChangedNotification handling already exists (unchanged)
