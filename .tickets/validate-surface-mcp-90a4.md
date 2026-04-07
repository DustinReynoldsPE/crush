---
id: validate-surface-mcp-90a4
stage: triage
deps: []
links: []
created: 2026-04-07T04:41:18Z
type: feature
priority: 4
assignee: Dustin Reynolds
version: 1
---
# Validate and surface MCP structured tool output (OutputSchema)


MCP tools can declare an OutputSchema (JSON schema) and return StructuredContent alongside text content. go-sdk passes StructuredContent through in CallToolResult but crush ignores it entirely — it only surfaces the text/image/audio content. Structured output enables richer tool integrations (JSON data, typed responses).

## Acceptance Criteria

StructuredContent from CallToolResult extracted and made available alongside text content; OutputSchema from Tool definition used to validate StructuredContent when present; structured data surfaced to LLM as additional context or rendered in TUI for user visibility
