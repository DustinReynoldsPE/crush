---
id: add-precompact-postcompact-08a3
stage: done
deps: []
links: []
created: 2026-04-07T03:18:44Z
type: feature
priority: 3
assignee: Dustin Reynolds
skipped: [spec, design, implement, test, verify]
version: 2
---
# Add PreCompact and PostCompact hook events

Fire PreCompact before context compaction begins and PostCompact after it completes. Enables dumping context to disk, alerting, or triggering follow-up logic around summarization.

## Scope
- Add PreCompact and PostCompact to HookEventName constants in internal/hooks/types.go
- Wire in the Summarize() path in internal/agent/agent.go
- PreCompact: fire async before Summarize() is called (alongside ContextWindowFull for auto-compaction, and separately for manual compaction)
- PostCompact: fire async after Summarize() returns successfully
- Payload: session_id, RawEventData with trigger reason (auto or manual)

## Acceptance Criteria
- PreCompact fires before summarization begins
- PostCompact fires after summarization completes successfully
- Both are non-blocking (async)
- Payload includes session_id and data.trigger ("auto" or "manual")
