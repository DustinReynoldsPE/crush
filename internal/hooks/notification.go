package hooks

import "time"

// NotificationType identifies the lifecycle phase of a hook event.
type NotificationType string

const (
	// NotificationRunning is published when a synchronous hook begins execution.
	// Never fired for async hooks.
	NotificationRunning NotificationType = "hook_running"
	// NotificationComplete is published when a hook chain finishes execution.
	NotificationComplete NotificationType = "hook_complete"
)

// HookNotification is a lifecycle event published by the hook Manager.
type HookNotification struct {
	Type      NotificationType
	HookType  HookType
	Command   string
	SessionID string
	// Complete-only fields:
	Decision string
	Reason   string
	Elapsed  time.Duration
}
