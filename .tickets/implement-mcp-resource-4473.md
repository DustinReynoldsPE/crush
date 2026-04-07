---
id: implement-mcp-resource-4473
stage: triage
deps: []
links: []
created: 2026-04-07T04:41:00Z
type: feature
priority: 3
assignee: Dustin Reynolds
version: 1
---
# Implement MCP resource template discovery


MCP servers can advertise URI templates for dynamic resource creation via resources/templates/list. go-sdk exposes ListResourceTemplates on ClientSession but crush never calls it. Servers that expose parameterized resources (e.g. file://{path}, db://{table}) can't be fully utilized.

## Acceptance Criteria

ListResourceTemplates called during MCP client initialization alongside ListResources; templates cached and exposed via Resources() or a new Templates() accessor; resource template list refreshed on ResourceListChangedNotification
