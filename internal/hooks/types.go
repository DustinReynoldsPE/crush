package hooks

// HookType defines the point in the agent lifecycle where the hook is executed.
type HookType = string

const (
	// SessionStart is executed once at the start of Run(), before any user input is processed.
	// It is blocking: a deny result aborts the session immediately.
	SessionStart HookType = "SessionStart"
	// UserPromptSubmit is executed after the user sends a prompt, before agent processing.
	UserPromptSubmit HookType = "UserPromptSubmit"
	// PreToolUse is executed before a tool is called.
	PreToolUse HookType = "PreToolUse"
	// PostToolUse is executed after a tool has returned a successful result.
	PostToolUse HookType = "PostToolUse"
	// PostToolUseFailure is executed after a tool has returned an error result.
	PostToolUseFailure HookType = "PostToolUseFailure"
	// Stop is executed when the agent turn ends cleanly.
	Stop HookType = "Stop"
	// PermissionRequest is executed before a permission dialog is shown; can auto-approve or auto-deny.
	PermissionRequest HookType = "PermissionRequest"
	// PermissionDenied is executed after a permission is denied (non-blocking).
	PermissionDenied HookType = "PermissionDenied"
	// Notification is executed when the agent finishes a turn cleanly and
	// would surface a notification to the user. It is always fired
	// asynchronously (fire-and-forget) and never blocks the response.
	// The RawEventData payload includes a "message" field (e.g. "agent_finished").
	Notification HookType = "Notification"
	// AgentError is executed when agent.Stream() returns a genuine error
	// (API failure, network error, provider error). It does NOT fire for
	// intentional stops: context cancellations or permission denials.
	// Always fired asynchronously; never affects error propagation.
	// The RawEventData payload includes an "error" field with the error string.
	AgentError HookType = "AgentError"
)

// HookEvent represents the context of an event triggering a hook.
type HookEvent struct {
	// HookEventName mirrors the event type for scripts that handle multiple events.
	HookEventName string `json:"hook_event_name"`
	// SessionID is the ID of the current session.
	SessionID string `json:"session_id"`
	// ToolName is the name of the tool being used, if applicable.
	ToolName string `json:"tool_name,omitempty"`
	// ToolInput contains the parameters passed to the tool.
	ToolInput interface{} `json:"tool_input,omitempty"`
	// RawEventData holds any other relevant event data.
	RawEventData interface{} `json:"data,omitempty"`
}

// HookResult represents the outcome of a hook execution.
type HookResult struct {
	// Decision determines the action to take: "proceed", "deny", "modify", or "error".
	Decision string
	// Reason provides a descriptive explanation for the decision or modification.
	Reason string
	// ModifiedEvent contains the event data if the hook returned a modification.
	ModifiedEvent interface{}
}
