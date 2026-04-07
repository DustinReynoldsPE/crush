---
id: support-stop-hook-0bb8
stage: done
deps: []
links: []
created: 2026-04-06T02:59:38Z
type: feature
priority: 2
assignee: Dustin Reynolds
skipped: [spec, design, implement, test, verify]
version: 2
---
# Support Stop hook blocking to continue conversation

Per spec, a Stop hook returning decision=deny/block prevents the agent from stopping and injects a continuation prompt. Currently the Stop hook result is ignored. Need to check the result and re-queue a continuation if denied. Parent: add-lifecycle-hooks-13d8
