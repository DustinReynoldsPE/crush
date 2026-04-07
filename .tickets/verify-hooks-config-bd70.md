---
id: verify-hooks-config-bd70
stage: done
deps: []
links: []
created: 2026-04-06T02:59:38Z
type: task
priority: 2
assignee: Dustin Reynolds
skipped: [implement, test, verify]
version: 2
---
# Verify hooks config round-trips through crush.json

Write a config load test that exercises the full JSON deserialization path for hooks: parse map[HookType][]HookConfig from crush.json, verify HookMatcher fields survive the round-trip, and confirm NewManager receives the correct map. Parent: add-lifecycle-hooks-13d8
