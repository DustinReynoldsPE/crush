---
id: wire-posttoolusefailure-hook-483e
stage: done
deps: []
links: []
created: 2026-04-06T02:59:38Z
type: feature
priority: 2
assignee: Dustin Reynolds
skipped: [spec, design, implement, test, verify]
version: 2
---
# Wire PostToolUseFailure hook event

Add a PostToolUseFailure hook call site in the error handling path of agent.go Run(), fired when a tool returns an error result. Sends tool name, input, and error message to hooks. Non-blocking per spec. Parent: add-lifecycle-hooks-13d8
