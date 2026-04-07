---
id: add-taskcreated-taskcompleted-3131
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
# Add TaskCreated and TaskCompleted hook events

Fire TaskCreated when a task is created via the TaskCreate tool and TaskCompleted when a task is marked done via TaskUpdate. Enables external task tracking, notifications, and workflow automation.

## Scope
- Add TaskCreated and TaskCompleted to HookEventName constants in internal/hooks/types.go
- Wire into the task tool implementations
- Payload: session_id, RawEventData with task id and title

## Acceptance Criteria
- TaskCreated fires when TaskCreate tool is called
- TaskCompleted fires when a task status is set to completed
- Both are non-blocking (async)
- Payload includes session_id, data.task_id, data.title
