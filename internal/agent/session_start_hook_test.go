package agent

import (
	"os"
	"path/filepath"
	"testing"

	"charm.land/catwalk/pkg/catwalk"
	"github.com/charmbracelet/crush/internal/hooks"
	"github.com/charmbracelet/crush/internal/message"
	"github.com/stretchr/testify/require"

	_ "github.com/joho/godotenv/autoload"
)

func agentFileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func writeAgentHookScript(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "hook.sh")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o755))
	return path
}

// seedSessionMessage inserts a minimal user message into the session so that
// len(msgs) > 0 and the title-generation goroutine is not spawned.
func seedSessionMessage(t *testing.T, env fakeEnv, sessionID string) {
	t.Helper()
	_, err := env.messages.Create(t.Context(), sessionID, message.CreateMessageParams{
		Role:  message.User,
		Parts: []message.ContentPart{message.TextContent{Text: "seed"}},
	})
	require.NoError(t, err)
}

// testSessionAgentWithHooks builds a minimal agent wired with hm.
// The nil LanguageModel is intentional: these tests exercise hook paths that
// return before any model call is made.
func testSessionAgentWithHooks(env fakeEnv, hm *hooks.Manager) SessionAgent {
	nilModel := Model{
		Model: nil,
		CatwalkCfg: catwalk.Model{
			ContextWindow:    200_000,
			DefaultMaxTokens: 10_000,
		},
	}
	return NewSessionAgent(SessionAgentOptions{
		LargeModel:   nilModel,
		SmallModel:   nilModel,
		IsYolo:       true,
		Sessions:     env.sessions,
		Messages:     env.messages,
		HooksManager: hm,
	})
}

// TestSessionStartHook_DenyAbortsRun verifies that a denying SessionStart hook
// causes Run() to return an error before reaching the LLM.
func TestSessionStartHook_DenyAbortsRun(t *testing.T) {
	t.Parallel()
	env := testEnv(t)

	denyScript := writeAgentHookScript(t, `#!/bin/sh
echo "session blocked by policy" >&2
exit 2
`)
	hm := hooks.NewManager(map[hooks.HookType][]hooks.HookConfig{
		hooks.SessionStart: {{Command: denyScript}},
	})

	a := testSessionAgentWithHooks(env, hm)
	sess, err := env.sessions.Create(t.Context(), "Test")
	require.NoError(t, err)

	_, err = a.Run(t.Context(), SessionAgentCall{
		Prompt:    "Hello",
		SessionID: sess.ID,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "session blocked by policy")
}

// TestSessionStartHook_DenyPayloadContainsSessionID verifies the hook receives
// the correct session_id in its JSON payload.
func TestSessionStartHook_DenyPayloadContainsSessionID(t *testing.T) {
	t.Parallel()
	env := testEnv(t)

	sess, err := env.sessions.Create(t.Context(), "Test")
	require.NoError(t, err)

	// The script denies only if the session_id field is missing or wrong.
	script := writeAgentHookScript(t, `#!/bin/sh
sid=$(cat | jq -r '.session_id')
[ -n "$sid" ] || { echo "missing session_id" >&2; exit 2; }
exit 2  # always deny so Run() returns early
`)
	hm := hooks.NewManager(map[hooks.HookType][]hooks.HookConfig{
		hooks.SessionStart: {{Command: script}},
	})

	a := testSessionAgentWithHooks(env, hm)
	_, err = a.Run(t.Context(), SessionAgentCall{
		Prompt:    "Hello",
		SessionID: sess.ID,
	})
	// The script exits 2 regardless — if session_id was absent it would also
	// write "missing session_id" to stderr, which would appear in err.Error().
	require.Error(t, err)
	require.NotContains(t, err.Error(), "missing session_id")
}

// TestSessionStartHook_DenySkipsUserPromptSubmit verifies that a denying
// SessionStart hook prevents UserPromptSubmit from firing.
func TestSessionStartHook_DenySkipsUserPromptSubmit(t *testing.T) {
	t.Parallel()
	env := testEnv(t)

	upsSentinel := t.TempDir() + "/ups"
	upsScript := writeAgentHookScript(t, `#!/bin/sh
touch `+upsSentinel+`
`)
	hm := hooks.NewManager(map[hooks.HookType][]hooks.HookConfig{
		hooks.SessionStart:     {{Command: `exit 2`}},
		hooks.UserPromptSubmit: {{Command: upsScript}},
	})

	a := testSessionAgentWithHooks(env, hm)
	sess, err := env.sessions.Create(t.Context(), "Test")
	require.NoError(t, err)

	_, err = a.Run(t.Context(), SessionAgentCall{
		Prompt:    "Hello",
		SessionID: sess.ID,
	})
	require.Error(t, err)
	require.False(t, agentFileExists(upsSentinel), "UserPromptSubmit must not fire after SessionStart deny")
}

// TestSessionStartHook_FiresBeforeUserPromptSubmit verifies ordering: SessionStart
// runs first, then UserPromptSubmit. The UserPromptSubmit hook denies to stop
// execution before the LLM is reached.
func TestSessionStartHook_FiresBeforeUserPromptSubmit(t *testing.T) {
	t.Parallel()
	env := testEnv(t)

	ssSentinel := t.TempDir() + "/session-start-ran"
	ssScript := writeAgentHookScript(t, `#!/bin/sh
touch `+ssSentinel+`
exit 0
`)
	// UserPromptSubmit denies so Run() returns before reaching the LLM.
	hm := hooks.NewManager(map[hooks.HookType][]hooks.HookConfig{
		hooks.SessionStart:     {{Command: ssScript}},
		hooks.UserPromptSubmit: {{Command: `exit 2`}},
	})

	a := testSessionAgentWithHooks(env, hm)
	sess, err := env.sessions.Create(t.Context(), "Test")
	require.NoError(t, err)
	// Pre-seed a message so the title-generation goroutine does not spawn
	// (it only runs on first message). This avoids a nil-model panic in tests.
	seedSessionMessage(t, env, sess.ID)

	_, err = a.Run(t.Context(), SessionAgentCall{
		Prompt:    "Hello",
		SessionID: sess.ID,
	})
	// UserPromptSubmit deny causes a non-nil error.
	require.Error(t, err)
	// SessionStart must have fired (sentinel exists) before UserPromptSubmit.
	require.True(t, agentFileExists(ssSentinel), "SessionStart hook must fire before UserPromptSubmit")
}
