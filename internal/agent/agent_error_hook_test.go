package agent

import (
	"context"
	"errors"
	"fmt"
	"iter"
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

// errLanguageModel is a stub that always returns a configurable error from Stream.
type errLanguageModel struct {
	err error
}

func (m *errLanguageModel) Generate(_ context.Context, _ fantasy.Call) (*fantasy.Response, error) {
	return nil, m.err
}
func (m *errLanguageModel) Stream(_ context.Context, _ fantasy.Call) (fantasy.StreamResponse, error) {
	return nil, m.err
}
func (m *errLanguageModel) GenerateObject(_ context.Context, _ fantasy.ObjectCall) (*fantasy.ObjectResponse, error) {
	return nil, m.err
}
func (m *errLanguageModel) StreamObject(_ context.Context, _ fantasy.ObjectCall) (fantasy.ObjectStreamResponse, error) {
	return iter.Seq[fantasy.ObjectStreamPart](nil), m.err
}
func (m *errLanguageModel) Provider() string { return "stub" }
func (m *errLanguageModel) Model() string    { return "stub-error-model" }

// testSessionAgentWithHooksAndStubModel builds an agent wired with hm and an errLanguageModel.
func testSessionAgentWithHooksAndStubModel(env fakeEnv, hm *hooks.Manager, lm fantasy.LanguageModel) SessionAgent {
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

// TestAgentErrorHook_FiresOnGenuineStreamError verifies that AgentError fires
// when agent.Stream returns an error that is not a cancel or permission denial.
func TestAgentErrorHook_FiresOnGenuineStreamError(t *testing.T) {
	t.Parallel()
	env := testEnv(t)

	sentinel := t.TempDir() + "/agent-error-ran"
	script := writeAgentHookScript(t, fmt.Sprintf(`#!/bin/sh
errMsg=$(cat | jq -r '.data.error')
[ -n "$errMsg" ] && [ "$errMsg" != "null" ] && touch %s
`, sentinel))

	hm := hooks.NewManager(map[hooks.HookType][]hooks.HookConfig{
		hooks.AgentError: {{Command: script, Async: true}},
	})

	lm := &errLanguageModel{err: errors.New("provider returned 503")}
	a := testSessionAgentWithHooksAndStubModel(env, hm, lm)

	sess, err := env.sessions.Create(t.Context(), "Test")
	require.NoError(t, err)
	seedSessionMessage(t, env, sess.ID)

	_, err = a.Run(t.Context(), SessionAgentCall{
		Prompt:    "Hello",
		SessionID: sess.ID,
	})
	require.Error(t, err)

	// Give the async hook goroutine a moment to complete.
	require.Eventually(t, func() bool {
		_, statErr := os.Stat(sentinel)
		return statErr == nil
	}, 3*time.Second, 50*time.Millisecond, "AgentError hook must fire with non-empty error field")
}

// TestAgentErrorHook_DoesNotFireOnContextCanceled verifies that AgentError is
// suppressed for intentional context cancellations.
func TestAgentErrorHook_DoesNotFireOnContextCanceled(t *testing.T) {
	t.Parallel()
	env := testEnv(t)

	sentinel := t.TempDir() + "/agent-error-ran"
	script := writeAgentHookScript(t, fmt.Sprintf(`#!/bin/sh
touch %s
`, sentinel))

	hm := hooks.NewManager(map[hooks.HookType][]hooks.HookConfig{
		hooks.AgentError: {{Command: script}},
	})

	lm := &errLanguageModel{err: context.Canceled}
	a := testSessionAgentWithHooksAndStubModel(env, hm, lm)

	sess, err := env.sessions.Create(t.Context(), "Test")
	require.NoError(t, err)
	seedSessionMessage(t, env, sess.ID)

	_, err = a.Run(t.Context(), SessionAgentCall{
		Prompt:    "Hello",
		SessionID: sess.ID,
	})
	require.Error(t, err)

	// Allow a short window for any errant async hook to fire.
	time.Sleep(200 * time.Millisecond)
	require.False(t, agentFileExists(sentinel), "AgentError must not fire for context.Canceled")
}

// TestAgentErrorHook_DoesNotFireOnPermissionDenied verifies that AgentError is
// suppressed for permission.ErrorPermissionDenied errors.
func TestAgentErrorHook_DoesNotFireOnPermissionDenied(t *testing.T) {
	t.Parallel()
	env := testEnv(t)

	sentinel := t.TempDir() + "/agent-error-ran"
	script := writeAgentHookScript(t, fmt.Sprintf(`#!/bin/sh
touch %s
`, sentinel))

	hm := hooks.NewManager(map[hooks.HookType][]hooks.HookConfig{
		hooks.AgentError: {{Command: script}},
	})

	lm := &errLanguageModel{err: permission.ErrorPermissionDenied}
	a := testSessionAgentWithHooksAndStubModel(env, hm, lm)

	sess, err := env.sessions.Create(t.Context(), "Test")
	require.NoError(t, err)
	seedSessionMessage(t, env, sess.ID)

	_, err = a.Run(t.Context(), SessionAgentCall{
		Prompt:    "Hello",
		SessionID: sess.ID,
	})
	require.Error(t, err)

	time.Sleep(200 * time.Millisecond)
	require.False(t, agentFileExists(sentinel), "AgentError must not fire for permission.ErrorPermissionDenied")
}
