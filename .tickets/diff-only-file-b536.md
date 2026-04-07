---
id: diff-only-file-b536
stage: triage
deps: []
links: []
created: 2026-04-07T05:17:52Z
type: feature
priority: 3
assignee: Dustin Reynolds
tags: [token-reduction, context]
version: 1
---
# Diff-only file content in context


When the agent reads a file it already read earlier in the session, send only the diff since last read rather than the full file again.

## Acceptance Criteria

File reads tracked per session; subsequent reads of same file send unified diff; full content sent on first read or after compaction
