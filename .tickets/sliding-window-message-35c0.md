---
id: sliding-window-message-35c0
stage: triage
deps: []
links: []
created: 2026-04-07T05:17:52Z
type: feature
priority: 2
assignee: Dustin Reynolds
tags: [token-reduction, context]
version: 1
---
# Sliding window message truncation


Drop oldest N messages from transcript once a soft token threshold is hit, rather than sending full history every turn. Old tool results are rarely relevant to the current task.

## Acceptance Criteria

Configurable soft threshold; oldest messages pruned first; system prompt preserved; truncation logged
