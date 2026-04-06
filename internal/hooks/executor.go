package hooks

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"time"
)

// waitDelay is how long to wait for I/O goroutines to finish after the process
// is killed, before forcibly closing pipes. Prevents hangs when sh spawns child
// processes that survive the parent.
const waitDelay = 2 * time.Second

// Executor handles the execution of a hook script as an external subprocess.
type Executor struct{}

// NewExecutor creates a new Executor instance.
func NewExecutor() *Executor {
	return &Executor{}
}

// Execute runs the hook command with the given event as JSON on stdin.
// It applies the per-hook timeout from hookCfg.TimeoutSeconds if set, then
// interprets exit codes: 0 = proceed (parse stdout for JSON override), 2 =
// deny (stderr becomes the reason), anything else = non-blocking error.
func (e *Executor) Execute(ctx context.Context, hookCfg HookConfig, event HookEvent) (HookResult, error) {
	if hookCfg.TimeoutSeconds > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(hookCfg.TimeoutSeconds)*time.Second)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", hookCfg.Command)
	cmd.WaitDelay = waitDelay

	payload, err := json.Marshal(event)
	if err != nil {
		return HookResult{}, fmt.Errorf("failed to marshal hook event: %w", err)
	}
	cmd.Stdin = bytes.NewReader(payload)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	runErr := cmd.Run()

	if runErr != nil {
		// Context deadline exceeded — treat as non-blocking timeout.
		if ctx.Err() == context.DeadlineExceeded {
			return HookResult{Decision: "error", Reason: "Hook execution timed out."}, nil
		}

		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			switch exitErr.ExitCode() {
			case 2:
				// Exit 2 = explicit deny; use stderr as the reason.
				return HookResult{Decision: "deny", Reason: stderrBuf.String()}, nil
			default:
				// Any other non-zero exit = non-blocking error; execution continues.
				slog.Warn("Hook exited with non-zero code",
					"command", hookCfg.Command,
					"code", exitErr.ExitCode(),
					"stderr", stderrBuf.String(),
				)
				return HookResult{Decision: "error", Reason: fmt.Sprintf("Exit %d: %s", exitErr.ExitCode(), stderrBuf.String())}, nil
			}
		}

		// Command could not be started (e.g. not found).
		return HookResult{}, fmt.Errorf("failed to run hook %q: %w", hookCfg.Command, runErr)
	}

	// Exit 0 — default to proceed, override from JSON stdout if present.
	result := HookResult{Decision: "proceed", Reason: "Execution successful."}

	if stdout := stdoutBuf.Bytes(); len(stdout) > 0 {
		var out JSONHookResult
		if parseErr := json.Unmarshal(stdout, &out); parseErr == nil {
			if out.Decision != "" {
				result.Decision = out.Decision
				result.Reason = out.Reason
				result.ModifiedEvent = out.ModifiedEvent
			}
		} else {
			slog.Debug("Hook stdout is not valid JSON; treating as proceed", "output", string(stdout))
		}
	}

	return result, nil
}

// JSONHookResult is the expected JSON structure for hook script output on stdout.
type JSONHookResult struct {
	Decision      string      `json:"decision"`               // "proceed", "deny", "modify"
	Reason        string      `json:"reason"`                 // Human-readable explanation.
	ModifiedEvent interface{} `json:"modified_event,omitempty"` // Replacement event data.
}
