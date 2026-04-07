---
id: mid-session-demand-ccfe
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
# Mid-session on-demand compaction


Allow users to trigger context compaction before the window fills. Summarizing at 60% utilization is cheaper than waiting for 95%.

## Acceptance Criteria

User can trigger compaction via a command/keybind at any point; compaction fires PreCompact/PostCompact hooks; works alongside auto-compaction
