---
id: wire-permissionrequest-permissiondenied-4e7e
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
# Wire PermissionRequest and PermissionDenied hook events

Hook into internal/permission to fire PermissionRequest (blocking — can auto-allow/deny) and PermissionDenied (non-blocking) events. Requires integrating hooks.Manager into the permission service. Parent: add-lifecycle-hooks-13d8
