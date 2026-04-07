package hooks

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"sync"
	"time"

	"github.com/charmbracelet/crush/internal/pubsub"
)

// Manager coordinates the execution of multiple configured hooks for a given event.
type Manager struct {
	executor  *Executor
	hooks     map[HookType][]HookConfig
	publisher pubsub.Publisher[HookNotification]
	mu        sync.Mutex
}

// ManagerOption configures a Manager.
type ManagerOption func(*Manager)

// WithPublisher attaches a publisher so the Manager emits lifecycle notifications.
// Callers that omit this option get silent (no-op) behaviour.
func WithPublisher(p pubsub.Publisher[HookNotification]) ManagerOption {
	return func(m *Manager) { m.publisher = p }
}

// NewManager creates a new Hook Manager.
func NewManager(hooks map[HookType][]HookConfig, opts ...ManagerOption) *Manager {
	m := &Manager{
		executor: NewExecutor(),
		hooks:    hooks,
	}
	for _, o := range opts {
		o(m)
	}
	slog.Info("Hook Manager initialized", "hook_types", len(m.hooks))
	return m
}

// publish emits a Notification if a publisher is configured.
func (m *Manager) publish(n HookNotification) {
	if m.publisher != nil {
		m.publisher.Publish(pubsub.UpdatedEvent, n)
	}
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

		// Async hooks fire and forget — never block or affect the chain.
		if hookCfg.Async {
			go func(cfg HookConfig, ev HookEvent) {
				if _, execErr := m.executor.Execute(context.Background(), cfg, ev); execErr != nil {
					slog.Warn("Async hook execution failed", "type", hookType, "error", execErr)
				}
			}(hookCfg, event)
			continue
		}

		start := time.Now()
		m.publish(HookNotification{
			Type:      NotificationRunning,
			HookType:  hookType,
			Command:   hookCfg.Command,
			SessionID: event.SessionID,
		})

		result, execErr := m.executor.Execute(ctx, hookCfg, event)

		elapsed := time.Since(start)
		if execErr != nil {
			slog.Error("Hook execution failed", "type", hookType, "error", execErr)
			m.publish(HookNotification{
				Type:      NotificationComplete,
				HookType:  hookType,
				Command:   hookCfg.Command,
				SessionID: event.SessionID,
				Decision:  "error",
				Reason:    fmt.Sprintf("Execution failure: %v", execErr),
				Elapsed:   elapsed,
			})
			return HookResult{Decision: "error", Reason: fmt.Sprintf("Execution failure: %v", execErr)}, execErr
		}

		finalResult = m.applyHookResult(finalResult, result)

		if finalResult.Decision == "deny" {
			slog.Warn("Hook denied action", "type", hookType, "reason", finalResult.Reason)
			m.publish(HookNotification{
				Type:      NotificationComplete,
				HookType:  hookType,
				Command:   hookCfg.Command,
				SessionID: event.SessionID,
				Decision:  "deny",
				Reason:    finalResult.Reason,
				Elapsed:   elapsed,
			})
			return finalResult, nil
		}

		m.publish(HookNotification{
			Type:      NotificationComplete,
			HookType:  hookType,
			Command:   hookCfg.Command,
			SessionID: event.SessionID,
			Decision:  finalResult.Decision,
			Reason:    finalResult.Reason,
			Elapsed:   elapsed,
		})
	}

	return finalResult, nil
}

// matchesEvent returns true if hookCfg should fire for this event.
// A hook with no matcher fires for all events of its type.
// ToolName is an exact match; Pattern is a regexp match against event.ToolName.
// Filename is an exact match against the "filename" key in RawEventData (FileChanged events).
func (m *Manager) matchesEvent(hookCfg HookConfig, event HookEvent) bool {
	matcher := hookCfg.Matcher
	if matcher.ToolName == "" && matcher.Pattern == "" && matcher.Filename == "" {
		return true
	}
	if matcher.Filename != "" {
		data, ok := event.RawEventData.(map[string]string)
		if !ok {
			return false
		}
		return data["filename"] == matcher.Filename
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
	// Pass through the latest result (covers "approve" and other extension decisions).
	return HookResult{Decision: new.Decision, Reason: new.Reason, ModifiedEvent: new.ModifiedEvent}
}
