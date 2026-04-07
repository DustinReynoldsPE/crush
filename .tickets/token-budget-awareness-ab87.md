---
id: token-budget-awareness-ab87
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
# Token budget awareness in system prompt


Inject current token utilization percentage into the system prompt each turn (e.g. 'Context: 34% full') to nudge the model toward more concise responses when budget is tight.

## Acceptance Criteria

Utilization % computed from current transcript; injected as a lightweight system prompt suffix; configurable threshold to enable injection
