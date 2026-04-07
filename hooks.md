# Agent Hooks in Crush

Crush supports lifecycle hooks that let you inject custom logic at specific points during an agent session. Hooks are shell commands (or HTTP endpoints) configured in `crush.json` and executed automatically by the hook manager.

## Hook types

| Event | Blocking | When it fires |
|---|---|---|
| `SessionStart` | Yes | Once at the start of `Run()`, before any prompt processing. A deny aborts the session. |
| `UserPromptSubmit` | Yes | After the user submits a prompt, before agent processing. A deny rejects the prompt. |
| `PreToolUse` | Yes | Before each tool call. A deny surfaces as a tool error so the model can recover. |
| `PostToolUse` | No | After a tool returns a successful result. Deny/error logged, not propagated. |
| `PostToolUseFailure` | No | After a tool returns an error result. Deny/error logged, not propagated. |
| `Stop` | Yes | When the agent turn ends cleanly. A deny injects the reason as a continuation prompt. |
| `PermissionRequest` | Yes | Before a permission dialog is shown. Can auto-approve or auto-deny. |
| `PermissionDenied` | No | After a permission is denied. Informational only. |
| `Notification` | No (async) | When the agent finishes a turn and would surface a user notification. Enables routing to Slack, ntfy, desktop notifiers, etc. |
| `AgentError` | No (async) | When `agent.Stream()` returns a genuine error (API failure, network error, provider error). Not fired for context cancellations or permission denials. |
| `ContextWindowFull` | No (async) | When the context window threshold is crossed and auto-summarization is about to begin. |
| `PreStep` | No (async) | Before each LLM inference call in the agent loop. Fires once per step. |
| `PostStep` | No (async) | After each agent step completes. Fires once per step with token usage. |
| `PreCompact` | No (async) | Before context compaction (summarization) begins, for both automatic (context window full) and manual triggers. |
| `PostCompact` | No (async) | After context compaction completes successfully. Not fired if compaction fails. |
| `SubagentStart` | No (async) | When a sub-agent session is spawned (multi-agent tool use). |
| `SubagentStop` | No (async) | When a sub-agent session finishes, on both success and error. |
| `StopFailure` | No (async) | When the agent turn ends due to a genuine error (API failure, rate limit, provider error). Not fired for context cancellations or permission denials. |
| `SessionEnd` | No (async) | When a session turn completes and no further queued prompts remain. Fires on both success and error paths (except cancellations). |
| `CwdChanged` | No (async) | When the working directory changes after a `bash` command (e.g. via `cd`). |
| `TaskCreated` | No (async) | When a new task is added to the todo list. |
| `TaskCompleted` | No (async) | When a task transitions to completed status. |
| `InstructionsLoaded` | No (async) | When a context/instructions file (CLAUDE.md, AGENTS.md, .cursor/rules/*.md, etc.) is loaded into the system prompt. |

**Blocking vs. async:** Blocking hooks run synchronously in the agent's call chain — their decision (`proceed`, `deny`, `modify`) affects control flow. Async hooks are fire-and-forget; their result is logged but never affects the agent.

## Payload

Every hook receives a JSON object on stdin:

```json
{
  "hook_event_name": "PreToolUse",
  "session_id": "abc-123",
  "tool_name": "bash",
  "tool_input": { "command": "ls" },
  "data": {}
}
```

Fields present depend on the event type:

| Field | Events |
|---|---|
| `hook_event_name` | All |
| `session_id` | All |
| `tool_name` | `PreToolUse`, `PostToolUse`, `PostToolUseFailure`, `PermissionRequest`, `PermissionDenied` |
| `tool_input` | `PreToolUse` |
| `data.message` | `Notification` (`"agent_finished"`) |
| `data.tokens_used` | `ContextWindowFull` (integer) |
| `data.threshold` | `ContextWindowFull` (integer) |
| `data.step_index` | `PreStep`, `PostStep` (0-based integer) |
| `data.finish_reason` | `PostStep` (string: `"stop"`, `"tool-calls"`, `"length"`, etc.) |
| `data.input_tokens` | `PostStep` (integer) |
| `data.output_tokens` | `PostStep` (integer) |
| `data.trigger` | `PreCompact`, `PostCompact` (`"auto"` or `"manual"`) |
| `data.agent_session_id` | `SubagentStart`, `SubagentStop` (sub-session ID string) |
| `data.error` | `StopFailure` (error string), `AgentError` (error string) |
| `data.finish_reason` | `StopFailure` (string) |
| `data.previous_cwd` | `CwdChanged` (old working directory path) |
| `data.cwd` | `CwdChanged` (new working directory path) |
| `data.title` | `TaskCreated`, `TaskCompleted` (task content string) |
| `data.path` | `InstructionsLoaded` (absolute file path) |
| `data.reason` | `InstructionsLoaded` (`"session_start"`) |

## Hook decisions

A hook script communicates its decision via exit code or stdout JSON:

| Exit code | Meaning |
|---|---|
| `0` | Proceed |
| `2` | Deny — stderr content becomes the reason |

Or return structured JSON:

```json
{ "decision": "deny", "reason": "blocked by policy" }
```

For `PermissionRequest`, `"approve"` is also a valid decision.

## Configuration

Hooks are configured in `crush.json` under `options.hooks`, keyed by event name:

```json
{
  "options": {
    "hooks": {
      "SessionStart": [
        { "command": "path/to/session_check.sh" }
      ],
      "PreToolUse": [
        {
          "command": "path/to/security_check.sh",
          "matcher": { "pattern": "bash|edit|write" },
          "timeout_seconds": 10
        }
      ],
      "PostToolUse": [
        {
          "command": "path/to/audit_log.sh",
          "async": true
        }
      ],
      "Notification": [
        { "command": "path/to/notify_slack.sh", "async": true }
      ],
      "AgentError": [
        { "command": "path/to/alert_oncall.sh", "async": true }
      ],
      "ContextWindowFull": [
        { "command": "path/to/log_context_pressure.sh", "async": true }
      ],
      "PreStep": [
        { "command": "path/to/log_step_start.sh", "async": true }
      ],
      "PostStep": [
        { "command": "path/to/track_token_usage.sh", "async": true }
      ]
    }
  }
}
```

### HookConfig fields

| Field | Type | Description |
|---|---|---|
| `command` | string | Shell command to execute |
| `timeout_seconds` | int | Max execution time (0 = no limit) |
| `async` | bool | Fire-and-forget; result ignored |
| `matcher.tool_name` | string | Only fire for this exact tool name |
| `matcher.pattern` | string | Only fire when tool name matches this regex |

## Examples

### Block dangerous shell commands

```sh
#!/bin/sh
# pre-tool-use: deny rm -rf
input=$(cat)
cmd=$(echo "$input" | jq -r '.tool_input.command // ""')
case "$cmd" in
  *"rm -rf"*) echo "destructive rm not allowed" >&2; exit 2 ;;
esac
```

### Notify Slack when agent finishes

```sh
#!/bin/sh
# notification hook (async: true)
payload=$(cat)
session=$(echo "$payload" | jq -r '.session_id')
curl -s -X POST "$SLACK_WEBHOOK" \
  -H 'Content-type: application/json' \
  -d "{\"text\": \"Crush agent finished (session: $session)\"}" > /dev/null
```

### Alert on API errors

```sh
#!/bin/sh
# agent-error hook (async: true)
err=$(cat | jq -r '.data.error')
curl -s -X POST "$PAGERDUTY_URL" \
  -d "{\"error\": \"$err\"}" > /dev/null
```

### Log context window pressure

```sh
#!/bin/sh
# context-window-full hook (async: true)
payload=$(cat)
tokens=$(echo "$payload" | jq -r '.data.tokens_used')
threshold=$(echo "$payload" | jq -r '.data.threshold')
echo "$(date -u +%FT%TZ) context_window_full tokens=$tokens threshold=$threshold" \
  >> /var/log/crush/context.log
```

### Track per-step token usage

```sh
#!/bin/sh
# post-step hook (async: true)
payload=$(cat)
step=$(echo "$payload" | jq -r '.data.step_index')
input=$(echo "$payload" | jq -r '.data.input_tokens')
output=$(echo "$payload" | jq -r '.data.output_tokens')
reason=$(echo "$payload" | jq -r '.data.finish_reason')
echo "$(date -u +%FT%TZ) step=$step input=$input output=$output reason=$reason" \
  >> /var/log/crush/steps.log
```

### React to directory changes

```sh
#!/bin/sh
# cwd-changed hook (async: true)
payload=$(cat)
prev=$(echo "$payload" | jq -r '.data.previous_cwd')
cwd=$(echo "$payload" | jq -r '.data.cwd')
# e.g. reload direnv when cwd changes
[ -f "$cwd/.envrc" ] && direnv allow "$cwd" > /dev/null 2>&1
echo "$(date -u +%FT%TZ) cwd_changed from=$prev to=$cwd" >> ~/.crush/cwd.log
```

### Track task progress externally

```sh
#!/bin/sh
# task-completed hook (async: true)
payload=$(cat)
title=$(echo "$payload" | jq -r '.data.title')
session=$(echo "$payload" | jq -r '.session_id')
curl -s -X POST "$WEBHOOK_URL" \
  -H 'Content-Type: application/json' \
  -d "{\"event\":\"task_completed\",\"title\":\"$title\",\"session\":\"$session\"}" \
  > /dev/null
```

### Audit which instruction files are active

```sh
#!/bin/sh
# instructions-loaded hook (async: true)
payload=$(cat)
path=$(echo "$payload" | jq -r '.data.path')
reason=$(echo "$payload" | jq -r '.data.reason')
echo "$(date -u +%FT%TZ) instructions_loaded path=$path reason=$reason" \
  >> ~/.crush/instructions.log
```
