package agent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"charm.land/catwalk/pkg/catwalk"
	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/hooks"
	"github.com/charmbracelet/crush/internal/permission"
	"github.com/stretchr/testify/require"

	_ "github.com/joho/godotenv/autoload"
)

func testAgentWithModel(env fakeEnv, hm *hooks.Manager, lm fantasy.LanguageModel) SessionAgent {
	m := Model{
		Model: lm,
		CatwalkCfg: catwalk.Model{
			ContextWindow:    200_000,
			DefaultMaxTokens: 10_000,
		},
	}
	return NewSessionAgent(SessionAgentOptions{
		LargeModel:   m,
		SmallModel:   m,
		IsYolo:       true,
		Sessions:     env.sessions,
		Messages:     env.messages,
		HooksManager: hm,
	})
}

// ── StopFailure ──────────────────────────────────────────────────────────────

func TestStopFailureHook_FiresOnGenuineError(t *testing.T) {
	t.Parallel()
	env := testEnv(t)

	sentinel := t.TempDir() + "/stop-failure-ran"
	script := writeAgentHookScript(t, fmt.Sprintf(`#!/bin/sh
errMsg=$(cat | jq -r '.data.error')
[ -n "$errMsg" ] && [ "$errMsg" != "null" ] && touch %s
`, sentinel))

	hm := hooks.NewManager(map[hooks.HookType][]hooks.HookConfig{
		hooks.StopFailure: {{Command: script, Async: true}},
	})

	a := testAgentWithModel(env, hm, &errLanguageModel{err: errors.New("provider returned 503")})
	sess, err := env.sessions.Create(t.Context(), "Test")
	require.NoError(t, err)
	seedSessionMessage(t, env, sess.ID)

	_, err = a.Run(t.Context(), SessionAgentCall{Prompt: "Hello", SessionID: sess.ID})
	require.Error(t, err)

	require.Eventually(t, func() bool {
		_, e := os.Stat(sentinel)
		return e == nil
	}, 3*time.Second, 50*time.Millisecond, "StopFailure hook must fire")
}

func TestStopFailureHook_DoesNotFireOnCancel(t *testing.T) {
	t.Parallel()
	env := testEnv(t)

	sentinel := t.TempDir() + "/stop-failure-ran"
	script := writeAgentHookScript(t, fmt.Sprintf("#!/bin/sh\ntouch %s\n", sentinel))

	hm := hooks.NewManager(map[hooks.HookType][]hooks.HookConfig{
		hooks.StopFailure: {{Command: script}},
	})

	a := testAgentWithModel(env, hm, &errLanguageModel{err: context.Canceled})
	sess, err := env.sessions.Create(t.Context(), "Test")
	require.NoError(t, err)
	seedSessionMessage(t, env, sess.ID)

	_, err = a.Run(t.Context(), SessionAgentCall{Prompt: "Hello", SessionID: sess.ID})
	require.Error(t, err)

	time.Sleep(200 * time.Millisecond)
	require.False(t, agentFileExists(sentinel), "StopFailure must not fire for context.Canceled")
}

func TestStopFailureHook_DoesNotFireOnPermissionDenied(t *testing.T) {
	t.Parallel()
	env := testEnv(t)

	sentinel := t.TempDir() + "/stop-failure-ran"
	script := writeAgentHookScript(t, fmt.Sprintf("#!/bin/sh\ntouch %s\n", sentinel))

	hm := hooks.NewManager(map[hooks.HookType][]hooks.HookConfig{
		hooks.StopFailure: {{Command: script}},
	})

	a := testAgentWithModel(env, hm, &errLanguageModel{err: permission.ErrorPermissionDenied})
	sess, err := env.sessions.Create(t.Context(), "Test")
	require.NoError(t, err)
	seedSessionMessage(t, env, sess.ID)

	_, err = a.Run(t.Context(), SessionAgentCall{Prompt: "Hello", SessionID: sess.ID})
	require.Error(t, err)

	time.Sleep(200 * time.Millisecond)
	require.False(t, agentFileExists(sentinel), "StopFailure must not fire for permission.ErrorPermissionDenied")
}

// ── SessionEnd ───────────────────────────────────────────────────────────────

func TestSessionEndHook_FiresOnCleanExit(t *testing.T) {
	t.Parallel()
	env := testEnv(t)

	sentinel := t.TempDir() + "/session-end-ran"
	script := writeAgentHookScript(t, fmt.Sprintf(`#!/bin/sh
sid=$(cat | jq -r '.session_id')
[ -n "$sid" ] && [ "$sid" != "null" ] && touch %s
`, sentinel))

	hm := hooks.NewManager(map[hooks.HookType][]hooks.HookConfig{
		hooks.SessionEnd: {{Command: script, Async: true}},
	})

	a := testAgentWithModel(env, hm, &singleStepModel{})
	sess, err := env.sessions.Create(t.Context(), "Test")
	require.NoError(t, err)
	seedSessionMessage(t, env, sess.ID)

	_, err = a.Run(t.Context(), SessionAgentCall{Prompt: "Hello", SessionID: sess.ID})
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		_, e := os.Stat(sentinel)
		return e == nil
	}, 3*time.Second, 50*time.Millisecond, "SessionEnd hook must fire on clean exit")
}

func TestSessionEndHook_FiresOnGenuineError(t *testing.T) {
	t.Parallel()
	env := testEnv(t)

	sentinel := t.TempDir() + "/session-end-ran"
	script := writeAgentHookScript(t, fmt.Sprintf("#!/bin/sh\ntouch %s\n", sentinel))

	hm := hooks.NewManager(map[hooks.HookType][]hooks.HookConfig{
		hooks.SessionEnd: {{Command: script, Async: true}},
	})

	a := testAgentWithModel(env, hm, &errLanguageModel{err: errors.New("provider error")})
	sess, err := env.sessions.Create(t.Context(), "Test")
	require.NoError(t, err)
	seedSessionMessage(t, env, sess.ID)

	_, err = a.Run(t.Context(), SessionAgentCall{Prompt: "Hello", SessionID: sess.ID})
	require.Error(t, err)

	require.Eventually(t, func() bool {
		_, e := os.Stat(sentinel)
		return e == nil
	}, 3*time.Second, 50*time.Millisecond, "SessionEnd hook must fire on error exit")
}

func TestSessionEndHook_DoesNotFireOnCancel(t *testing.T) {
	t.Parallel()
	env := testEnv(t)

	sentinel := t.TempDir() + "/session-end-ran"
	script := writeAgentHookScript(t, fmt.Sprintf("#!/bin/sh\ntouch %s\n", sentinel))

	hm := hooks.NewManager(map[hooks.HookType][]hooks.HookConfig{
		hooks.SessionEnd: {{Command: script}},
	})

	a := testAgentWithModel(env, hm, &errLanguageModel{err: context.Canceled})
	sess, err := env.sessions.Create(t.Context(), "Test")
	require.NoError(t, err)
	seedSessionMessage(t, env, sess.ID)

	_, err = a.Run(t.Context(), SessionAgentCall{Prompt: "Hello", SessionID: sess.ID})
	require.Error(t, err)

	time.Sleep(200 * time.Millisecond)
	require.False(t, agentFileExists(sentinel), "SessionEnd must not fire for context.Canceled")
}
