package agent

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	charm_fantasy "charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/hooks"
	"github.com/stretchr/testify/require"
)

// TestSubagentStartHook_FiresOnSubAgentSpawn verifies that SubagentStart fires
// asynchronously when runSubAgent creates a sub-session.
func TestSubagentStartHook_FiresOnSubAgentSpawn(t *testing.T) {
	t.Parallel()
	env := testEnv(t)

	sentinel := filepath.Join(t.TempDir(), "subagent_start_fired")
	script := writeAgentHookScript(t, "#!/bin/sh\ntouch "+sentinel+"\n")

	hm := hooks.NewManager(map[hooks.HookType][]hooks.HookConfig{
		hooks.SubagentStart: {{Command: script}},
	})

	coord := newTestCoordinator(t, env, "stub", stubProviderCfg())
	coord.hooksManager = hm

	parentSession, err := env.sessions.Create(t.Context(), "Parent")
	require.NoError(t, err)

	subAgent := newMockAgent("stub", 4096, func(_ context.Context, _ SessionAgentCall) (*charm_fantasy.AgentResult, error) {
		return agentResultWithText("done"), nil
	})

	_, err = coord.runSubAgent(t.Context(), subAgentParams{
		Agent:          subAgent,
		SessionID:      parentSession.ID,
		AgentMessageID: "msg-1",
		ToolCallID:     "tool-1",
		Prompt:         "do something",
		SessionTitle:   "Sub Task",
	})
	require.NoError(t, err)

	require.Eventually(t, func() bool { return agentFileExists(sentinel) },
		3*time.Second, 10*time.Millisecond, "SubagentStart hook must fire")
}

// TestSubagentStopHook_FiresAfterSubAgentCompletes verifies SubagentStop fires
// after the sub-agent run finishes, regardless of result.
func TestSubagentStopHook_FiresAfterSubAgentCompletes(t *testing.T) {
	t.Parallel()
	env := testEnv(t)

	sentinel := filepath.Join(t.TempDir(), "subagent_stop_fired")
	script := writeAgentHookScript(t, "#!/bin/sh\ntouch "+sentinel+"\n")

	hm := hooks.NewManager(map[hooks.HookType][]hooks.HookConfig{
		hooks.SubagentStop: {{Command: script}},
	})

	coord := newTestCoordinator(t, env, "stub", stubProviderCfg())
	coord.hooksManager = hm

	parentSession, err := env.sessions.Create(t.Context(), "Parent")
	require.NoError(t, err)

	subAgent := newMockAgent("stub", 4096, func(_ context.Context, _ SessionAgentCall) (*charm_fantasy.AgentResult, error) {
		return agentResultWithText("done"), nil
	})

	_, err = coord.runSubAgent(t.Context(), subAgentParams{
		Agent:          subAgent,
		SessionID:      parentSession.ID,
		AgentMessageID: "msg-2",
		ToolCallID:     "tool-2",
		Prompt:         "do something",
		SessionTitle:   "Sub Task",
	})
	require.NoError(t, err)

	require.Eventually(t, func() bool { return agentFileExists(sentinel) },
		3*time.Second, 10*time.Millisecond, "SubagentStop hook must fire after completion")
}

// TestSubagentHook_PayloadHasAgentSessionID verifies the SubagentStart payload
// includes the agent_session_id field.
func TestSubagentHook_PayloadHasAgentSessionID(t *testing.T) {
	t.Parallel()
	env := testEnv(t)

	sentinel := filepath.Join(t.TempDir(), "agent_session_id_ok")
	script := writeAgentHookScript(t, `#!/bin/sh
payload=$(cat)
agent_sid=$(echo "$payload" | jq -r '.data.agent_session_id')
[ -n "$agent_sid" ] && [ "$agent_sid" != "null" ] && touch `+sentinel+`
`)

	hm := hooks.NewManager(map[hooks.HookType][]hooks.HookConfig{
		hooks.SubagentStart: {{Command: script}},
	})

	coord := newTestCoordinator(t, env, "stub", stubProviderCfg())
	coord.hooksManager = hm

	parentSession, err := env.sessions.Create(t.Context(), "Parent")
	require.NoError(t, err)

	subAgent := newMockAgent("stub", 4096, func(_ context.Context, _ SessionAgentCall) (*charm_fantasy.AgentResult, error) {
		return agentResultWithText("done"), nil
	})

	_, err = coord.runSubAgent(t.Context(), subAgentParams{
		Agent:          subAgent,
		SessionID:      parentSession.ID,
		AgentMessageID: "msg-3",
		ToolCallID:     "tool-3",
		Prompt:         "do something",
		SessionTitle:   "Sub Task",
	})
	require.NoError(t, err)

	require.Eventually(t, func() bool { return agentFileExists(sentinel) },
		3*time.Second, 10*time.Millisecond, "SubagentStart hook must include agent_session_id")
}
