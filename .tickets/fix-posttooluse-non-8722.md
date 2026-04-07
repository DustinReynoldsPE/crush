---
id: fix-posttooluse-non-8722
stage: done
deps: []
links: []
created: 2026-04-06T02:59:38Z
type: feature
priority: 1
assignee: Dustin Reynolds
skipped: [design, implement, test, verify]
version: 3
---
# Fix PostToolUse to be non-blocking per spec

PostToolUse currently returns its hook error from OnToolResult, aborting the stream. Per the Claude Code spec, PostToolUse exit 2 means 'show stderr (tool already ran)' — it must not block execution. The hook result should be surfaced as context to the model, not an error. Parent: add-lifecycle-hooks-13d8
