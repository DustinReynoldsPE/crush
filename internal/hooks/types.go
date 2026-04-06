package hooks

// HookType defines the point in the agent lifecycle where the hook is executed.
type HookType = string

const (
	// UserPromptSubmit is executed after the user sends a prompt, before agent processing.
	UserPromptSubmit HookType = "UserPromptSubmit"
	// PreToolUse is executed before a tool is called.
	PreToolUse HookType = "PreToolUse"
	// PostToolUse is executed after a tool has returned a result.
	PostToolUse HookType = "PostToolUse"
	// Stop is executed when the agent turn ends cleanly.
	Stop HookType = "Stop"
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
