---
id: add-lifecycle-hooks-13d8
stage: done
deps: []
links: []
created: 2026-04-05T16:01:47Z
type: feature
priority: 2
assignee: Dustin Reynolds
skipped: [spec, design, implement, test, verify]
version: 3
---
# Add lifecycle hooks system to Crush

Add user-configurable lifecycle hooks to Crush, similar to Claude Code's hook system. Crush's internal pubsub.Broker already broadcasts typed events (PermissionRequest, AgentEvent, Message, Session). Wire these into a user-facing hook executor that runs shell commands at key lifecycle points. Reference: upstream issue #1336, closed PR #1337 (POC that didn't handle blocking results).
