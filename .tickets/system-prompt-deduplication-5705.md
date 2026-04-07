---
id: system-prompt-deduplication-5705
stage: triage
deps: []
links: []
created: 2026-04-07T05:17:52Z
type: feature
priority: 2
assignee: Dustin Reynolds
tags: [token-reduction, performance]
version: 1
---
# System prompt deduplication across turns


Cache the system prompt prefix and only send it on the first turn or after compaction, rather than re-injecting the full system prompt every turn.

## Acceptance Criteria

System prompt cached per session; only delta/changes resent on subsequent turns; reset after compaction
