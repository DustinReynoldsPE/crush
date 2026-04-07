---
id: small-model-routing-0d71
stage: triage
deps: []
links: []
created: 2026-04-07T05:17:52Z
type: feature
priority: 2
assignee: Dustin Reynolds
tags: [token-reduction, models]
version: 1
---
# Small model routing for cheap subtasks


Route simple subtasks (tool-result interpretation, format checks, classification) to the small model. Reserve the large model for planning and code generation.

## Acceptance Criteria

Configurable routing rules; subtask types documented; cost difference visible in session metrics
