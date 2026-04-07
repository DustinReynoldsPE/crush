package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"os"
	"sync"
	"testing"
	"time"

	"charm.land/catwalk/pkg/catwalk"
	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/hooks"
	"github.com/stretchr/testify/require"

	_ "github.com/joho/godotenv/autoload"
)

// singleStepModel returns one stop response immediately with no tool calls.
type singleStepModel struct {
	mu      sync.Mutex
	calls   int
	stopErr error
}

func (m *singleStepModel) Generate(_ context.Context, _ fantasy.Call) (*fantasy.Response, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *singleStepModel) Stream(_ context.Context, _ fantasy.Call) (fantasy.StreamResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.stopErr != nil {
		return nil, m.stopErr
	}
	m.calls++
	return func(yield func(fantasy.StreamPart) bool) {
		yield(fantasy.StreamPart{
			Type:         fantasy.StreamPartTypeFinish,
			FinishReason: fantasy.FinishReasonStop,
			Usage: fantasy.Usage{
				InputTokens:  100,
				OutputTokens: 50,
			},
		})
	}, nil
}

func (m *singleStepModel) GenerateObject(_ context.Context, _ fantasy.ObjectCall) (*fantasy.ObjectResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *singleStepModel) StreamObject(_ context.Context, _ fantasy.ObjectCall) (fantasy.ObjectStreamResponse, error) {
	return iter.Seq[fantasy.ObjectStreamPart](nil), fmt.Errorf("not implemented")
}

func (m *singleStepModel) Provider() string { return "stub" }
func (m *singleStepModel) Model() string    { return "stub-single-step" }

func testAgentWithStepHooks(env fakeEnv, hm *hooks.Manager, lm fantasy.LanguageModel) SessionAgent {
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

// TestStepHooks_PreAndPostFireOncePerStep verifies that PreStep and PostStep
// each fire once for a single-step agent run, with step_index=0.
func TestStepHooks_PreAndPostFireOncePerStep(t *testing.T) {
	t.Parallel()
	env := testEnv(t)

	preSentinel := t.TempDir() + "/pre-step"
	postSentinel := t.TempDir() + "/post-step"

	preScript := writeAgentHookScript(t, fmt.Sprintf(`#!/bin/sh
idx=$(cat | jq -r '.data.step_index')
echo "$idx" > %s
`, preSentinel))
	postScript := writeAgentHookScript(t, fmt.Sprintf(`#!/bin/sh
payload=$(cat)
idx=$(echo "$payload" | jq -r '.data.step_index')
reason=$(echo "$payload" | jq -r '.data.finish_reason')
echo "${idx}:${reason}" > %s
`, postSentinel))

	hm := hooks.NewManager(map[hooks.HookType][]hooks.HookConfig{
		hooks.PreStep:  {{Command: preScript, Async: true}},
		hooks.PostStep: {{Command: postScript, Async: true}},
	})

	lm := &singleStepModel{}
	a := testAgentWithStepHooks(env, hm, lm)

	sess, err := env.sessions.Create(t.Context(), "Test")
	require.NoError(t, err)
	seedSessionMessage(t, env, sess.ID)

	_, err = a.Run(t.Context(), SessionAgentCall{
		Prompt:    "Hello",
		SessionID: sess.ID,
	})
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		_, e := os.Stat(preSentinel)
		return e == nil
	}, 3*time.Second, 50*time.Millisecond, "PreStep hook must fire")

	require.Eventually(t, func() bool {
		_, e := os.Stat(postSentinel)
		return e == nil
	}, 3*time.Second, 50*time.Millisecond, "PostStep hook must fire")

	preData, _ := os.ReadFile(preSentinel)
	require.Equal(t, "0", string([]byte(preData)[:1]), "PreStep step_index must be 0")

	postData, _ := os.ReadFile(postSentinel)
	require.Contains(t, string(postData), "0:stop", "PostStep must have step_index=0 and finish_reason=stop")
}

// TestStepHooks_PayloadFields verifies PostStep payload includes token counts.
func TestStepHooks_PostStep_IncludesTokenUsage(t *testing.T) {
	t.Parallel()
	env := testEnv(t)

	resultFile := t.TempDir() + "/post-payload"
	script := writeAgentHookScript(t, fmt.Sprintf(`#!/bin/sh
cat > %s
`, resultFile))

	hm := hooks.NewManager(map[hooks.HookType][]hooks.HookConfig{
		hooks.PostStep: {{Command: script, Async: true}},
	})

	lm := &singleStepModel{}
	a := testAgentWithStepHooks(env, hm, lm)

	sess, err := env.sessions.Create(t.Context(), "Test")
	require.NoError(t, err)
	seedSessionMessage(t, env, sess.ID)

	_, err = a.Run(t.Context(), SessionAgentCall{
		Prompt:    "Hello",
		SessionID: sess.ID,
	})
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		_, e := os.Stat(resultFile)
		return e == nil
	}, 3*time.Second, 50*time.Millisecond, "PostStep hook must fire")

	raw, err := os.ReadFile(resultFile)
	require.NoError(t, err)

	var payload struct {
		Data struct {
			InputTokens  int64  `json:"input_tokens"`
			OutputTokens int64  `json:"output_tokens"`
			FinishReason string `json:"finish_reason"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(raw, &payload))
	require.Equal(t, int64(100), payload.Data.InputTokens)
	require.Equal(t, int64(50), payload.Data.OutputTokens)
	require.Equal(t, "stop", payload.Data.FinishReason)
}
