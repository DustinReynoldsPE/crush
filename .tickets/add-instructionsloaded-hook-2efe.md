---
id: add-instructionsloaded-hook-2efe
stage: done
deps: []
links: []
created: 2026-04-07T03:18:44Z
type: feature
priority: 3
assignee: Dustin Reynolds
skipped: [spec, design, implement, test, verify]
version: 2
---
# Add InstructionsLoaded hook event

Fire InstructionsLoaded when a CLAUDE.md or .claude/rules/*.md file is loaded into context — at session start and when lazily loaded during a session. Enables auditing which instruction files are active, logging, or modifying behavior when specific rule sets are loaded.

## Scope
- Add InstructionsLoaded to HookEventName constants in internal/hooks/types.go
- Identify where CLAUDE.md and rules files are loaded in the config/prompt layer
- Fire async with file path and load reason in payload
- Matcher filters by load reason (session_start, nested_traversal, path_glob_match, include, compact)

## Acceptance Criteria
- Fires each time an instructions file is loaded
- Non-blocking (async)
- Payload includes session_id, data.path, data.reason
