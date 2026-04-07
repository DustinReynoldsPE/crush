package hooks

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func newEvent(hookType HookType, sessionID, toolName string) HookEvent {
	return HookEvent{
		HookEventName: hookType,
		SessionID:     sessionID,
		ToolName:      toolName,
	}
}

// ── Exit code semantics ──────────────────────────────────────────────────────

func TestExecutor_ExitZero_DefaultProceed(t *testing.T) {
	t.Parallel()
	e := NewExecutor()
	result, err := e.Execute(context.Background(), HookConfig{Command: "true"}, newEvent(PreToolUse, "s1", "bash"))
	require.NoError(t, err)
	require.Equal(t, "proceed", result.Decision)
}

func TestExecutor_ExitTwo_Deny(t *testing.T) {
	t.Parallel()
	e := NewExecutor()
	cfg := HookConfig{Command: `echo 'blocked by policy' >&2; exit 2`}
	result, err := e.Execute(context.Background(), cfg, newEvent(PreToolUse, "s1", "bash"))
	require.NoError(t, err)
	require.Equal(t, "deny", result.Decision)
	require.Contains(t, result.Reason, "blocked by policy")
}

func TestExecutor_ExitOne_NonBlockingError(t *testing.T) {
	t.Parallel()
	e := NewExecutor()
	result, err := e.Execute(context.Background(), HookConfig{Command: "exit 1"}, newEvent(PreToolUse, "s1", "bash"))
	require.NoError(t, err) // non-blocking: no Go error
	require.Equal(t, "error", result.Decision)
	require.Contains(t, result.Reason, "Exit 1")
}

func TestExecutor_ExitThree_NonBlockingError(t *testing.T) {
	t.Parallel()
	e := NewExecutor()
	result, err := e.Execute(context.Background(), HookConfig{Command: "exit 3"}, newEvent(PreToolUse, "s1", "bash"))
	require.NoError(t, err)
	require.Equal(t, "error", result.Decision)
}

// ── JSON stdout parsing ──────────────────────────────────────────────────────

func TestExecutor_JSONStdout_ProceedDecision(t *testing.T) {
	t.Parallel()
	e := NewExecutor()
	cfg := HookConfig{Command: `echo '{"decision":"proceed","reason":"all clear"}'`}
	result, err := e.Execute(context.Background(), cfg, newEvent(PreToolUse, "s1", "bash"))
	require.NoError(t, err)
	require.Equal(t, "proceed", result.Decision)
	require.Equal(t, "all clear", result.Reason)
}

func TestExecutor_JSONStdout_DenyDecision(t *testing.T) {
	t.Parallel()
	e := NewExecutor()
	cfg := HookConfig{Command: `echo '{"decision":"deny","reason":"not today"}'`}
	result, err := e.Execute(context.Background(), cfg, newEvent(PreToolUse, "s1", "bash"))
	require.NoError(t, err)
	require.Equal(t, "deny", result.Decision)
	require.Equal(t, "not today", result.Reason)
}

func TestExecutor_JSONStdout_ModifyDecision(t *testing.T) {
	t.Parallel()
	e := NewExecutor()
	cfg := HookConfig{Command: `echo '{"decision":"modify","reason":"adjusted","modified_event":{"patched":true}}'`}
	result, err := e.Execute(context.Background(), cfg, newEvent(PreToolUse, "s1", "write"))
	require.NoError(t, err)
	require.Equal(t, "modify", result.Decision)
	require.NotNil(t, result.ModifiedEvent)
}

func TestExecutor_NonJSONStdout_DefaultProceed(t *testing.T) {
	t.Parallel()
	e := NewExecutor()
	cfg := HookConfig{Command: `echo "plain text — not JSON"`}
	result, err := e.Execute(context.Background(), cfg, newEvent(PreToolUse, "s1", "bash"))
	require.NoError(t, err)
	require.Equal(t, "proceed", result.Decision)
}

func TestExecutor_EmptyStdout_DefaultProceed(t *testing.T) {
	t.Parallel()
	e := NewExecutor()
	result, err := e.Execute(context.Background(), HookConfig{Command: "true"}, newEvent(Stop, "s1", ""))
	require.NoError(t, err)
	require.Equal(t, "proceed", result.Decision)
}

// ── Stdin / event payload ────────────────────────────────────────────────────

func TestExecutor_StdinContainsSessionID(t *testing.T) {
	t.Parallel()
	script := writeScript(t, `#!/bin/sh
session=$(cat | jq -r '.session_id')
[ "$session" = "sess-xyz" ] || { echo "wrong session: $session" >&2; exit 2; }
`)
	e := NewExecutor()
	result, err := e.Execute(context.Background(), HookConfig{Command: script}, newEvent(PreToolUse, "sess-xyz", "bash"))
	require.NoError(t, err)
	require.Equal(t, "proceed", result.Decision)
}

func TestExecutor_StdinContainsToolName(t *testing.T) {
	t.Parallel()
	script := writeScript(t, `#!/bin/sh
tool=$(cat | jq -r '.tool_name')
[ "$tool" = "write" ] || { echo "wrong tool: $tool" >&2; exit 2; }
`)
	e := NewExecutor()
	result, err := e.Execute(context.Background(), HookConfig{Command: script}, newEvent(PreToolUse, "s1", "write"))
	require.NoError(t, err)
	require.Equal(t, "proceed", result.Decision)
}

func TestExecutor_StdinContainsHookEventName(t *testing.T) {
	t.Parallel()
	script := writeScript(t, `#!/bin/sh
name=$(cat | jq -r '.hook_event_name')
[ "$name" = "PostToolUse" ] || { echo "wrong name: $name" >&2; exit 2; }
`)
	e := NewExecutor()
	result, err := e.Execute(context.Background(), HookConfig{Command: script}, newEvent(PostToolUse, "s1", "bash"))
	require.NoError(t, err)
	require.Equal(t, "proceed", result.Decision)
}

func TestExecutor_StdinContainsToolInput(t *testing.T) {
	t.Parallel()
	script := writeScript(t, `#!/bin/sh
cmd=$(cat | jq -r '.tool_input.command')
[ "$cmd" = "ls -la" ] || { echo "wrong command: $cmd" >&2; exit 2; }
`)
	e := NewExecutor()
	event := HookEvent{
		HookEventName: PreToolUse,
		SessionID:     "s1",
		ToolName:      "bash",
		ToolInput:     map[string]string{"command": "ls -la"},
	}
	result, err := e.Execute(context.Background(), HookConfig{Command: script}, event)
	require.NoError(t, err)
	require.Equal(t, "proceed", result.Decision)
}

// ── Timeout ──────────────────────────────────────────────────────────────────

func TestExecutor_PerHookTimeout_Enforced(t *testing.T) {
	t.Parallel()
	e := NewExecutor()
	cfg := HookConfig{Command: "sleep 60", TimeoutSeconds: 1}
	start := time.Now()
	result, err := e.Execute(context.Background(), cfg, newEvent(PreToolUse, "s1", "bash"))
	require.NoError(t, err)
	require.Equal(t, "error", result.Decision)
	require.Contains(t, result.Reason, "timed out")
	require.Less(t, time.Since(start), 8*time.Second)
}

func TestExecutor_ContextCancellation_DoesNotHang(t *testing.T) {
	t.Parallel()
	e := NewExecutor()
	cfg := HookConfig{Command: "sleep 60"}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	start := time.Now()
	e.Execute(ctx, cfg, newEvent(PreToolUse, "s1", "bash")) //nolint:errcheck
	require.Less(t, time.Since(start), 8*time.Second)
}

// ── Event marshaling ─────────────────────────────────────────────────────────

func TestHookEvent_JSONMarshal_SnakeCaseKeys(t *testing.T) {
	t.Parallel()
	event := HookEvent{
		HookEventName: PostToolUse,
		SessionID:     "session-1",
		ToolName:      "bash",
		ToolInput:     map[string]string{"command": "ls"},
	}
	data, err := json.Marshal(event)
	require.NoError(t, err)

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &parsed))
	require.Equal(t, "PostToolUse", parsed["hook_event_name"])
	require.Equal(t, "session-1", parsed["session_id"])
	require.Equal(t, "bash", parsed["tool_name"])
	require.NotNil(t, parsed["tool_input"])
}

// ── Env injection ────────────────────────────────────────────────────────────

func TestExecutor_Env_InjectedIntoSubprocess(t *testing.T) {
	t.Parallel()
	script := writeScript(t, `#!/bin/sh
[ "$CRUSH_HOOK_SECRET" = "hunter2" ] || { echo "wrong: $CRUSH_HOOK_SECRET" >&2; exit 2; }
`)
	cfg := HookConfig{Command: script, Env: map[string]string{"CRUSH_HOOK_SECRET": "hunter2"}}
	result, err := NewExecutor().Execute(context.Background(), cfg, newEvent(SessionStart, "s1", ""))
	require.NoError(t, err)
	require.Equal(t, "proceed", result.Decision)
}

func TestExecutor_Env_MultipleVarsAllVisible(t *testing.T) {
	t.Parallel()
	script := writeScript(t, `#!/bin/sh
[ "$CRUSH_KEY_A" = "alpha" ] && [ "$CRUSH_KEY_B" = "beta" ] || { echo "missing vars" >&2; exit 2; }
`)
	cfg := HookConfig{Command: script, Env: map[string]string{"CRUSH_KEY_A": "alpha", "CRUSH_KEY_B": "beta"}}
	result, err := NewExecutor().Execute(context.Background(), cfg, newEvent(SessionStart, "s1", ""))
	require.NoError(t, err)
	require.Equal(t, "proceed", result.Decision)
}

func TestExecutor_Env_ParentEnvStillInherited(t *testing.T) {
	t.Parallel()
	// PATH must be present or sh can't find any commands — proves parent env is kept.
	script := writeScript(t, `#!/bin/sh
[ -n "$PATH" ] || { echo "PATH missing" >&2; exit 2; }
`)
	cfg := HookConfig{Command: script, Env: map[string]string{"CRUSH_EXTRA": "injected"}}
	result, err := NewExecutor().Execute(context.Background(), cfg, newEvent(SessionStart, "s1", ""))
	require.NoError(t, err)
	require.Equal(t, "proceed", result.Decision)
}

func TestExecutor_Env_OverridesParentVar(t *testing.T) {
	t.Setenv("CRUSH_OVERRIDE_TEST", "original")
	script := writeScript(t, `#!/bin/sh
[ "$CRUSH_OVERRIDE_TEST" = "overridden" ] || { echo "got: $CRUSH_OVERRIDE_TEST" >&2; exit 2; }
`)
	cfg := HookConfig{Command: script, Env: map[string]string{"CRUSH_OVERRIDE_TEST": "overridden"}}
	result, err := NewExecutor().Execute(context.Background(), cfg, newEvent(SessionStart, "s1", ""))
	require.NoError(t, err)
	require.Equal(t, "proceed", result.Decision)
}

func TestExecutor_Env_NilMap_NoRegression(t *testing.T) {
	t.Parallel()
	result, err := NewExecutor().Execute(context.Background(), HookConfig{Command: "true"}, newEvent(PreToolUse, "s1", "bash"))
	require.NoError(t, err)
	require.Equal(t, "proceed", result.Decision)
}

func TestHookConfig_Env_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	cfg := HookConfig{
		Command: "my-hook.sh",
		Env:     map[string]string{"API_KEY": "secret", "HOST": "localhost"},
	}
	data, err := json.Marshal(cfg)
	require.NoError(t, err)
	var decoded HookConfig
	require.NoError(t, json.Unmarshal(data, &decoded))
	require.Equal(t, cfg.Env, decoded.Env)
}

func TestHookConfig_Env_OmittedFromJSONWhenNil(t *testing.T) {
	t.Parallel()
	data, err := json.Marshal(HookConfig{Command: "script.sh"})
	require.NoError(t, err)
	require.NotContains(t, string(data), `"env"`)
}

func TestHookEvent_EmptyOptionals_OmittedFromJSON(t *testing.T) {
	t.Parallel()
	event := HookEvent{SessionID: "s1"}
	data, err := json.Marshal(event)
	require.NoError(t, err)

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &parsed))
	_, hasToolName := parsed["tool_name"]
	_, hasToolInput := parsed["tool_input"]
	require.False(t, hasToolName, "tool_name should be omitted when empty")
	require.False(t, hasToolInput, "tool_input should be omitted when nil")
}
