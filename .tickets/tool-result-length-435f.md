---
id: tool-result-length-435f
stage: triage
deps: []
links: []
created: 2026-04-07T05:17:52Z
type: feature
priority: 2
assignee: Dustin Reynolds
tags: [token-reduction, tools]
version: 1
---
# Tool result length capping


Cap tool results at a configurable character limit (e.g. 4000 chars) and append a truncation notice. Prevents large bash/grep outputs from bloating the transcript.

## Acceptance Criteria

Configurable per-tool and global cap; truncation suffix shows bytes omitted; cap applies before result enters context
