---
id: add-filechanged-hook-fe19
stage: done
deps: []
links: []
created: 2026-04-07T03:18:44Z
type: feature
priority: 3
assignee: Dustin Reynolds
skipped: [design, implement, test, verify]
version: 3
---
# Add FileChanged hook event

Fire FileChanged when a watched file changes on disk. The matcher field specifies which filenames (by basename) to watch. Enables reactive config reloading, environment refresh, or triggering logic when specific files are modified.

## Scope
- Add FileChanged to HookEventName constants in internal/hooks/types.go
- Set up a file watcher in the app layer for files matching hook matchers
- Fire async when a watched file changes
- Matcher filters by basename (e.g. .envrc, .env)

## Acceptance Criteria
- Fires when a file matching the matcher basename changes on disk
- Non-blocking (async)
- Payload includes session_id, data.path (full path), data.filename (basename)

<!-- checkpoint: testing -->
<!-- exit-state: Implementation complete and all tests passing. FileChanged hook fires async for file writes/creates in workDir, with optional Filename matcher for basename filtering. -->
<!-- key-files: internal/hooks/types.go, internal/hooks/config.go, internal/hooks/manager.go, internal/hooks/filewatcher.go, internal/agent/coordinator.go -->
<!-- open-questions: none -->
