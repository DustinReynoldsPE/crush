---
id: subscribe-hook-executor-0588
stage: done
deps: []
links: []
created: 2026-04-05T22:59:46Z
type: feature
priority: 2
assignee: Dustin Reynolds
parent: add-lifecycle-hooks-13d8
skipped: [design, implement, test, verify]
version: 4
---
# Subscribe Hook Executor to events

Wire hooks.NewManager() construction at app startup by reading Options.Hooks config and building the map[HookType][]HookConfig. Pass HooksManager through coordinator.buildAgent into SessionAgentOptions. Add remaining hook call sites in agent.go: PostToolUse (in OnToolResult), Stop (after agent loop ends), UserPromptSubmit (before Run dispatches to LLM).
