---
id: relevance-filtered-instruction-4dbe
stage: triage
deps: []
links: []
created: 2026-04-07T05:17:52Z
type: feature
priority: 3
assignee: Dustin Reynolds
tags: [token-reduction, context, instructions]
version: 1
---
# Relevance-filtered instruction file injection


Rather than injecting entire CLAUDE.md/AGENTS.md files, use keyword matching against the current prompt to inject only the sections relevant to the current task.

## Acceptance Criteria

Section-level parsing of instruction files; keyword relevance scoring; full file injected when no match found (safe fallback)
