package hooks

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"sync"
)

// Manager coordinates the execution of multiple configured hooks for a given event.
type Manager struct {
	executor *Executor
	hooks    map[HookType][]HookConfig
	mu       sync.Mutex
}

// NewManager creates a new Hook Manager.
func NewManager(hooks map[HookType][]HookConfig) *Manager {
	m := &Manager{
		executor: NewExecutor(),
		hooks:    hooks,
	}
	slog.Info("Hook Manager initialized", "hook_types", len(m.hooks))
	return m
}

// Execute runs all matching hooks for hookType sequentially. A "deny" result
// from any hook stops the chain immediately. Returns "proceed" if no hooks are
// configured or all hooks approve.
func (m *Manager) Execute(ctx context.Context, hookType HookType, event HookEvent) (HookResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	hookConfigs, ok := m.hooks[hookType]
	if !ok || len(hookConfigs) == 0 {
		return HookResult{Decision: "proceed", Reason: "No hooks configured for this event type."}, nil
	}

	// Stamp the event name so scripts can self-identify.
	event.HookEventName = hookType

	finalResult := HookResult{Decision: "proceed", Reason: "No hooks executed or all approved."}

	for _, hookCfg := range hookConfigs {
		if !m.matchesEvent(hookCfg, event) {
			continue
		}

		if ctx.Err() != nil {
			slog.Warn("Context cancelled during hook execution", "type", hookType)
			return HookResult{Decision: "error", Reason: "Context cancelled during hook execution."}, ctx.Err()
		}

		result, execErr := m.executor.Execute(ctx, hookCfg, event)
		if execErr != nil {
			slog.Error("Hook execution failed", "type", hookType, "error", execErr)
			return HookResult{Decision: "error", Reason: fmt.Sprintf("Execution failure: %v", execErr)}, execErr
		}

		finalResult = m.applyHookResult(finalResult, result)

		if finalResult.Decision == "deny" {
			slog.Warn("Hook denied action", "type", hookType, "reason", finalResult.Reason)
			return finalResult, nil
		}
	}

	return finalResult, nil
}

// matchesEvent returns true if hookCfg should fire for this event.
// A hook with no matcher fires for all events of its type.
// ToolName is an exact match; Pattern is a regexp match against event.ToolName.
func (m *Manager) matchesEvent(hookCfg HookConfig, event HookEvent) bool {
	matcher := hookCfg.Matcher
	if matcher.ToolName == "" && matcher.Pattern == "" {
		return true
	}
	if matcher.ToolName != "" {
		return matcher.ToolName == event.ToolName
	}
	if matcher.Pattern != "" {
		matched, err := regexp.MatchString(matcher.Pattern, event.ToolName)
		return err == nil && matched
	}
	return true
}

// applyHookResult merges the new hook result into the running final result.
// Priority: deny > modify > proceed.
func (m *Manager) applyHookResult(current, new HookResult) HookResult {
	if new.Decision == "deny" {
		return HookResult{Decision: "deny", Reason: new.Reason}
	}
	if current.Decision == "deny" {
		return current
	}
	if new.Decision == "modify" {
		if current.Decision == "modify" {
			return HookResult{
				Decision:      "modify",
				Reason:        fmt.Sprintf("%s; %s", current.Reason, new.Reason),
				ModifiedEvent: new.ModifiedEvent,
			}
		}
		return HookResult{Decision: "modify", Reason: new.Reason, ModifiedEvent: new.ModifiedEvent}
	}
	// Both proceed — keep latest reason.
	return HookResult{Decision: "proceed", Reason: new.Reason}
}
