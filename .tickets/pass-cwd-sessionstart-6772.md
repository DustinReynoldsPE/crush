---
id: pass-cwd-sessionstart-6772
stage: triage
deps: []
links: []
created: 2026-04-07T17:23:39Z
type: task
priority: 1
assignee: Dustin Reynolds
version: 1
---
# Pass cwd in SessionStart hook payload


The SessionStart hook event only includes session_id. The memory-session-start.sh hook bails out silently when cwd is missing because it cannot derive a project name.

## What to do

In internal/agent/agent.go, add cwd to the RawEventData for the SessionStart hook:

  hookResult, hookErr := a.hooksManager.Execute(ctx, hooks.SessionStart, hooks.HookEvent{
      SessionID:    call.SessionID,
      RawEventData: map[string]string{"cwd": a.workDir},  // or however workDir is accessed
  })

## Why

memory-session-start.sh uses cwd to derive the project name (basename of cwd), which it uses to:
1. Filter memory searches by project
2. Bail silently if no project context (the current behavior when cwd is missing)

Without cwd, the SessionStart hook is a no-op for all crush sessions — no memories are ever injected.

## Verify

After the fix, run:
  payload='{"hook_event_name":"SessionStart","session_id":"test","cwd":"/home/dustin/code/research"}'
  echo $payload | bash ~/.config/crush/hooks/memory-session-start.sh

Should return a hookSpecificOutput JSON with memories from the research project.

## Note on hookSpecificOutput

crush does not yet implement the hookSpecificOutput context-injection mechanism (Claude Code extension). Even with cwd passing, the memories will be fetched but not injected into the system prompt. That is a separate ticket.
