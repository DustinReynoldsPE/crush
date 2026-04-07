---
id: add-configchange-hook-c7df
stage: done
deps: []
links: []
created: 2026-04-07T03:18:44Z
type: feature
priority: 4
assignee: Dustin Reynolds
skipped: [spec, design, implement, test, verify]
version: 2
---
# Add ConfigChange hook event

Fire ConfigChange when a configuration file changes during an active session. Enables reacting to live config updates, reloading hooks, or logging configuration drift.

## Scope
- Add ConfigChange to HookEventName constants in internal/hooks/types.go
- Wire into config file watching in the app layer
- Matcher filters by config source (user_settings, project_settings, local_settings, policy_settings, skills)
- Payload: session_id, RawEventData with source and changed file path

## Acceptance Criteria
- Fires when a config file changes during a session
- Non-blocking (async)
- Payload includes session_id, data.source, data.path
