package agent

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/hooks"
	"github.com/stretchr/testify/require"
)

// stubProviderCfg returns a minimal ProviderConfig suitable for test coordinators.
func stubProviderCfg() config.ProviderConfig {
	return config.ProviderConfig{}
}

var errSummarizeFailed = errors.New("summarize failed")

// TestPreCompactHook_ManualSummarize verifies that coordinator.Summarize fires
// PreCompact before and PostCompact after a successful compaction.
func TestPreCompactHook_ManualSummarize(t *testing.T) {
	t.Parallel()
	env := testEnv(t)

	presentinel := filepath.Join(t.TempDir(), "pre_compact_fired")
	postsentinel := filepath.Join(t.TempDir(), "post_compact_fired")

	preScript := writeAgentHookScript(t, "#!/bin/sh\ntouch "+presentinel+"\n")
	postScript := writeAgentHookScript(t, "#!/bin/sh\ntouch "+postsentinel+"\n")

	hm := hooks.NewManager(map[hooks.HookType][]hooks.HookConfig{
		hooks.PreCompact:  {{Command: preScript}},
		hooks.PostCompact: {{Command: postScript}},
	})

	coord := newTestCoordinator(t, env, "stub", stubProviderCfg())
	coord.hooksManager = hm
	coord.currentAgent = newMockAgent("stub", 4096, nil)

	parentSession, err := env.sessions.Create(t.Context(), "Parent")
	require.NoError(t, err)

	err = coord.Summarize(context.Background(), parentSession.ID)
	require.NoError(t, err)

	require.Eventually(t, func() bool { return agentFileExists(presentinel) },
		3*time.Second, 10*time.Millisecond, "PreCompact hook must fire")
	require.Eventually(t, func() bool { return agentFileExists(postsentinel) },
		3*time.Second, 10*time.Millisecond, "PostCompact hook must fire after successful compaction")
}

// TestPreCompactHook_TriggerField verifies that the trigger field is "manual"
// in the PreCompact payload when fired via coordinator.Summarize.
func TestPreCompactHook_TriggerField(t *testing.T) {
	t.Parallel()
	env := testEnv(t)

	sentinel := filepath.Join(t.TempDir(), "trigger_ok")
	script := writeAgentHookScript(t, `#!/bin/sh
payload=$(cat)
trigger=$(echo "$payload" | jq -r '.data.trigger')
[ "$trigger" = "manual" ] && touch `+sentinel+`
`)

	hm := hooks.NewManager(map[hooks.HookType][]hooks.HookConfig{
		hooks.PreCompact: {{Command: script}},
	})

	coord := newTestCoordinator(t, env, "stub", stubProviderCfg())
	coord.hooksManager = hm
	coord.currentAgent = newMockAgent("stub", 4096, nil)

	parentSession, err := env.sessions.Create(t.Context(), "Parent")
	require.NoError(t, err)

	_ = coord.Summarize(context.Background(), parentSession.ID)

	require.Eventually(t, func() bool { return agentFileExists(sentinel) },
		3*time.Second, 10*time.Millisecond, "PreCompact hook must receive trigger=manual")
}

// TestPostCompactHook_NoFireOnError verifies PostCompact does not fire when
// Summarize returns an error.
func TestPostCompactHook_NoFireOnError(t *testing.T) {
	t.Parallel()
	env := testEnv(t)

	sentinel := filepath.Join(t.TempDir(), "post_compact_fired")
	postScript := writeAgentHookScript(t, "#!/bin/sh\ntouch "+sentinel+"\n")

	hm := hooks.NewManager(map[hooks.HookType][]hooks.HookConfig{
		hooks.PostCompact: {{Command: postScript}},
	})

	failAgent := newMockAgent("stub", 4096, nil)
	failAgent.summarizeErr = errSummarizeFailed

	coord := newTestCoordinator(t, env, "stub", stubProviderCfg())
	coord.hooksManager = hm
	coord.currentAgent = failAgent

	parentSession, err := env.sessions.Create(t.Context(), "Parent")
	require.NoError(t, err)

	err = coord.Summarize(context.Background(), parentSession.ID)
	require.Error(t, err)

	time.Sleep(100 * time.Millisecond)
	require.False(t, agentFileExists(sentinel), "PostCompact must not fire when Summarize errors")
}
