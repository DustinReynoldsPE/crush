---
id: support-pretooluse-updatedinput-80a1
stage: done
deps: []
links: []
created: 2026-04-06T02:59:38Z
type: feature
priority: 1
assignee: Dustin Reynolds
skipped: [design, implement, test, verify]
version: 3
---
# Support PreToolUse updatedInput to modify tool parameters

PreToolUse hooks can return updatedInput in their JSON output to modify tool parameters before execution. Currently only deny/proceed is checked. Need to extract updatedInput from the JSON response and pass it back through PrepareStep so fantasy uses the modified input. Parent: add-lifecycle-hooks-13d8
