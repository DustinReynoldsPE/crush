---
id: idle-session-auto-e748
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
# Idle session auto-compact


When a session sits idle for a configurable duration, automatically trigger compaction so users return to a summarized context rather than a stale full transcript.

## Acceptance Criteria

Configurable idle timeout; auto-compact fires PreCompact/PostCompact hooks with trigger=auto; user notified on resume
