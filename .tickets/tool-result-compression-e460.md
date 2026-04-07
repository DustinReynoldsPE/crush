---
id: tool-result-compression-e460
stage: triage
deps: []
links: []
created: 2026-04-07T05:17:52Z
type: feature
priority: 3
assignee: Dustin Reynolds
tags: [token-reduction, hooks]
version: 1
---
# Tool result compression via PostToolUse hook


Allow PostToolUse hooks to rewrite/compress tool output before it enters the transcript — strip ANSI codes, collapse repetitive log lines, summarize git log to hashes and subjects.

## Acceptance Criteria

Hook can return modified tool result; compression applied before transcript insertion; hook result format documented
