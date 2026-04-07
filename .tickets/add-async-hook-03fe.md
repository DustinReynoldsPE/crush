---
id: add-async-hook-03fe
stage: done
deps: []
links: []
created: 2026-04-06T02:59:38Z
type: feature
priority: 3
assignee: Dustin Reynolds
skipped: [spec, design, implement, test, verify]
version: 2
---
# Add async hook support

Add async bool field to HookConfig. When true, fire the hook in a background goroutine and do not wait for the result or apply its decision. Useful for logging/notification hooks on non-blocking events. Parent: add-lifecycle-hooks-13d8
