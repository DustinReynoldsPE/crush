package hooks

// HookMatcher defines criteria for matching an event to a specific hook script.
type HookMatcher struct {
	// ToolName is used to match against the name of a tool being called.
	// If empty, it matches all tools.
	ToolName string `json:"tool_name,omitempty"`
	// MatcherType indicates the type of match (e.g., "exact", "regex").
	MatcherType string `json:"matcher_type,omitempty"`
	// Pattern is the pattern to match against (e.g., a regex pattern for tool names).
	Pattern string `json:"pattern,omitempty"`
}

// HookConfig defines the configuration for a single hook script.
type HookConfig struct {
	// Command is the path to the script that will be executed.
	Command string `json:"command"`
	// TimeoutSeconds specifies the maximum time the hook script is allowed to run.
	TimeoutSeconds int `json:"timeout_seconds,omitempty"`
	// Matcher is used to determine if this hook should run for a given event.
	Matcher HookMatcher `json:"matcher,omitempty"`
}

// HookSet groups configurations by HookType.
type HookSet struct {
	// Hooks is a list of configurations for a specific HookType.
	Hooks []HookConfig `json:"hooks"`
}