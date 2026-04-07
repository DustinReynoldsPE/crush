# Hook Ideas

Practical use cases for each Crush lifecycle hook. 5 ideas per hook, organized by category.

---

## Session Lifecycle

### SessionStart

1. **Branch Isolation per Session** — On startup, check if a per-session git branch exists and create one if not, then write `export GIT_BRANCH=...` to `$CLAUDE_ENV_FILE` so every subsequent bash tool runs on an isolated branch, enabling parallel sessions without conflicts.

2. **Token Budget Injection** — Read a per-project `.crush/budget.json` file and inject a system message into `additionalContext` telling the agent its token/cost ceiling for this session; block the session (non-zero exit) if the project's monthly spend already exceeds a configured threshold.

3. **Ticket Context Preload** — Parse the current git branch name for a Linear/Jira ticket ID (e.g. `feat/LIN-4321-auth`), fetch the ticket's title, description, and acceptance criteria via the Linear API, and inject them into `additionalContext` so the agent starts with full task context without the user having to paste it.

4. **Environment Sanity Check** — Verify required tools (correct Node version, Docker daemon running, env vars set) and abort the session with a clear error message if prerequisites are missing, preventing the agent from proceeding into a broken environment that will waste tokens diagnosing tool failures.

5. **Audit Trail Initialization** — Write a session start record (timestamp, `session_id`, `cwd`, git commit SHA, current user) to a centralized append-only audit log (e.g. a shared PostgreSQL table or S3 object), satisfying compliance requirements for who ran AI-assisted sessions on which codebases and when.

### UserPromptSubmit

1. **Secret Leak Prevention** — Scan the submitted prompt for patterns matching API keys, passwords, and tokens (regex against common secret formats like `sk-...`, `ghp_...`, AWS key patterns) and block the prompt with exit code 2 before it ever reaches the model, preventing accidental credential exfiltration through the LLM API.

2. **Prompt Injection Guard** — Detect prompt injection attempts in the user's text — instructions like "ignore previous instructions", "you are now DAN", or "disregard your system prompt" — and reject the prompt with a policy violation message, protecting team-shared sessions from social engineering attacks.

3. **Scope Enforcement** — Read a `.crush/policy.json` file that defines allowed file paths and blocked operations for the current project, then block prompts that explicitly request actions outside those bounds (e.g. "deploy to production", "modify the billing service") before the agent can act on them.

4. **Auto Context Enrichment** — Detect when a prompt references a function name or file path and automatically run `grep`/`go doc` to locate the symbol, injecting its definition and call sites into `additionalContext` so the agent starts each turn with precise code context rather than spending tool calls searching.

5. **Prompt Cost Estimator** — Estimate token cost of the prompt combined with current transcript length, append a cost-awareness note to `additionalContext` if the projected turn cost exceeds a threshold (e.g. "Note: this session has consumed ~$1.20 so far"), and optionally block prompts when a hard session cap is reached.

### Stop

1. **Test Gate** — Run the project's test suite (`go test ./...`, `npm test`, `pytest`) after every agent turn and, if tests fail, exit with code 2 and feed the failure output back to the agent as stderr so it self-corrects before the turn is considered complete — enforcing a "never leave tests broken" policy automatically.

2. **Work Completeness Verification** — Use a fast, cheap model (e.g. `claude-haiku`) to evaluate whether the agent's response actually addressed all parts of the user's original prompt; if the evaluation returns "incomplete", exit 2 with a continuation prompt listing the unaddressed items, preventing premature stops on multi-part tasks.

3. **Auto-Commit Checkpoint** — On clean stop, run `git add -p` (or a targeted `git add` of changed files) and commit with the user's original prompt as the commit message, creating atomic, labelled checkpoints after each agent turn that are easy to bisect or roll back without manual intervention.

4. **Slack Turn Summary** — Post a one-line summary of the completed turn to a team Slack channel (via incoming webhook), including session ID, cwd, and a truncated first line of the agent's response — giving distributed teams real-time visibility into what AI-assisted work is happening across the codebase.

5. **Linear Issue Auto-Update** — Parse the current branch for a Linear ticket ID, then call the Linear API to append a comment summarizing what the agent just did (files changed, commands run, current status), keeping the ticket's activity feed current without requiring the developer to manually update it.

### StopFailure

1. **PagerDuty Rate Limit Alert** — On `rate_limit` error type, fire a PagerDuty event via their Events API with severity `warning`, including session ID and timestamp, so on-call engineers are notified when shared-team API quotas are being exhausted and can redistribute load or request quota increases proactively.

2. **Slack Error Channel Notification** — Post the error type, session ID, cwd, and git branch to a `#ai-agent-errors` Slack channel so the team has a shared, searchable record of API failures across all developer sessions, making it easy to spot patterns (e.g. billing errors recurring on Fridays, or a specific project burning through tokens).

3. **Billing Error Circuit Breaker** — On `billing_error`, write a lockfile to `~/.crush/billing.lock` that future `SessionStart` hooks check before allowing new sessions to start, preventing developers from unknowingly continuing to attempt expensive sessions when the account is in a failed billing state.

4. **Error Frequency Tracking** — Append each failure event to a local SQLite database (`~/.crush/failures.db`) with error type and timestamp, then query it to detect if the same error type has occurred more than N times in the past hour and send a digest alert — distinguishing a transient blip from a sustained outage.

5. **Transcript Preservation on Failure** — Immediately copy the in-progress transcript to a safe backup path (`~/crush-recovery/SESSION_ID.jsonl`) so that if a server error causes context loss, the partial work can be reviewed, resumed manually, or replayed into a new session.

### SessionEnd

1. **Automated PR Creation** — On session end, check if there are uncommitted changes or unpushed commits on a feature branch and, if so, run `gh pr create` with a body summarising the session's transcript, turning every completed AI coding session into a reviewable PR without any manual steps.

2. **Session Cost Report** — Parse the session transcript JSONL to sum token usage across all turns, compute estimated API cost, and append a line to a per-project `~/.crush/cost-log.csv` — enabling weekly/monthly spend reports per project or per developer without requiring any external billing dashboard.

3. **Transcript Archival to S3** — Compress and upload the session transcript to an S3 bucket (`s3://company-ai-audit/YYYY/MM/DD/SESSION_ID.jsonl.gz`) for long-term compliance storage, giving security teams a permanent, searchable record of all AI-assisted code changes tied to the code they produced.

4. **Knowledge Base Update** — Run a script that scans the session transcript for tool-call results that produced new facts (architecture decisions, API shape discoveries, recurring bug patterns) and appends structured entries to a project-level `KNOWLEDGE.md` or vector store, building a living knowledge base from every session.

5. **Metrics Dashboard Push** — Emit a structured JSON event to a Datadog/Grafana endpoint with session metadata (duration, turn count, total tokens, files modified, tests passed/failed), enabling an engineering-wide dashboard that tracks AI coding agent usage, productivity impact, and reliability trends over time.

---

## Tool & Permission

### PreToolUse

1. **Dangerous Command Blocker** — Parse `tool_input.command` for patterns like `rm -rf /`, `dd if=`, `mkfs`, or `:(){ :|:& };:` (fork bombs) and deny with a clear reason before they execute.

2. **Secret Leak Prevention** — Scan `tool_input` for patterns matching AWS keys (`AKIA[0-9A-Z]{16}`), private key headers, or `.env` variable assignments before any write or bash tool runs, blocking accidental credential commits.

3. **Rate Limiter / Throttle Gate** — Track tool invocation counts per session in a shared file; deny the call with a `"rate limit exceeded"` message if a threshold (e.g. 60 bash calls/minute) is breached to prevent runaway agents.

4. **Scope Enforcement (Path Guard)** — For file write/edit tools, compare the target path against an allowlist of directories (e.g. only within the project root); deny writes outside the workspace to prevent the agent from touching system files.

5. **Compliance Pre-flight Check** — Before any `git commit` or `git push` command, validate that required checks have run (tests, linter exit codes stored in a session state file) and deny the commit if they haven't, enforcing a mandatory CI gate.

### PostToolUse

1. **Structured Audit Log** — Append a JSONL record of every tool call (tool name, sanitized input, timestamp, session ID) to a tamper-evident append-only log file for SOC 2 / compliance audit trails.

2. **Cost Metering** — After each tool use, read cumulative token counts from a session ledger and emit a metric to a StatsD or Prometheus pushgateway for per-project cost dashboards.

3. **SIEM Event Forwarding** — Forward `tool_name` and key input fields to a Splunk HEC or Elasticsearch ingest endpoint so security teams can correlate agent activity with other system events in real time.

4. **Auto-Formatter / Linter Trigger** — After a successful file write or edit, run `gofmt`, `prettier`, or `ruff` on the changed file path extracted from `tool_input`, keeping the codebase clean without requiring the agent to do it explicitly.

5. **Git Auto-Snapshot** — After each successful file edit, create a timestamped WIP commit so that any subsequent mistake can be rolled back without losing intermediate work.

### PostToolUseFailure

1. **Failure Rate Alerting** — Count consecutive tool failures per session; if a threshold (e.g. 3 bash failures in a row) is exceeded, fire a PagerDuty or Slack alert so a human can intervene before the agent spirals into a retry loop.

2. **Error Taxonomy Logger** — Parse the tool error output and classify it (permission denied, network timeout, syntax error, not found) then write structured metrics to a monitoring system to identify which tool categories fail most often.

3. **Distributed Tracing** — Record each failure event with its tool name, input hash, and error type to an OpenTelemetry span so post-mortem analysis can identify flaky tools or environment issues.

4. **Fallback Environment Reset** — On bash tool failure, check whether the error indicates a missing binary or broken PATH and automatically run an environment repair script (e.g. re-source `.envrc`, reload `nvm`) to restore a known-good state.

5. **Developer Notification on Edit Failure** — When a file write or edit tool fails (e.g. permission denied on a production config), send an immediate desktop notification or SMS so the developer knows the agent hit a guardrail and may need guidance.

### PermissionRequest

1. **Auto-Approve Safe Patterns** — Automatically approve permission requests for read-only tools (bash commands matching `^(ls|cat|grep|git (log|diff|status))`) without prompting, reducing friction for clearly benign operations.

2. **Auto-Deny Sensitive Paths** — Inspect the requested tool and input; if the command touches `/etc/`, `~/.ssh/`, or production environment files, return `deny` immediately with a policy reason instead of showing the dialog to the user.

3. **Time-of-Day Policy Gate** — During off-hours (nights/weekends), auto-deny permission requests for destructive operations (deletes, force-pushes) and require an explicit out-of-band approval token to proceed, enforcing a change-freeze policy.

4. **Team Role-Based Approval** — Look up the current user in a roles config file; if the user is a read-only reviewer, auto-deny all write/execute permission requests and log the attempt, implementing RBAC for shared agent environments.

5. **Scope-Scoped Auto-Approve** — When the workspace config declares an `allowed_tools` list, auto-approve any permission request whose tool name and target path fall within that declared scope, removing repetitive dialogs for pre-vetted operations.

### PermissionDenied

1. **Denial Audit Trail** — Write every denial event (tool name, session ID, timestamp, reason) to an immutable append-only log, giving security teams a record of what the agent attempted but was not permitted to do.

2. **Slack / Webhook Escalation** — Post a message to a `#agent-alerts` Slack channel when a permission is denied for a write or execute operation, so the team can decide in real time whether to re-run with elevated permissions.

3. **Session Denial Budget** — Track denial counts per session; if denials exceed a configurable threshold, emit a warning metric and optionally kill a runaway or misconfigured agent that keeps requesting forbidden operations.

4. **Developer Hint Injection** — Parse the denied tool name and write a human-readable hint to a status file (e.g. "Tip: add `bash` to `allowed_tools` in crush.json to allow shell commands") that the TUI can surface, reducing confusion about why an operation was blocked.

5. **Compliance Evidence Collection** — For regulated environments, serialize the full denial payload to a content-addressed store (e.g. S3 with object lock) to produce an immutable evidence record that a specific action was evaluated and rejected per policy.

---

## Step & Compaction

### PreStep

1. **Rate-Limit Guard** — Read a shared counter file and exit 2 if the step count for the current minute exceeds a configured threshold, preventing runaway agentic loops from exhausting API quota.

2. **Per-Session Step Log** — Append a timestamped line to a structured JSONL audit log before each inference call, giving a precise timeline of when each LLM request was initiated across long debugging sessions.

3. **Live Progress Indicator** — Send a push notification via ntfy/Pushover on every Nth step (e.g. `step_index % 10 == 0`) so you can monitor background agent runs on mobile without polling the terminal.

4. **Context Snapshot Checkpoint** — At each step boundary, copy the current working directory's git diff to a timestamped snapshot file so you can replay exactly what state the agent saw before each inference call.

5. **Dynamic System-Load Backoff** — Query `uptime` or a Prometheus endpoint for current CPU load and sleep 2-5 seconds before expensive steps when load average exceeds a threshold, preventing the agent from saturating the machine during builds or tests.

### PostStep

1. **Per-Step Token Cost Accumulator** — Parse `input_tokens` and `output_tokens`, multiply by your model's per-token pricing, and append to a running total; emit a warning to stderr when the session crosses a configurable budget (e.g. $0.50).

2. **Prometheus Pushgateway Metrics** — Push `crush_step_tokens{type="input"}` and `crush_step_tokens{type="output"}` gauges to a Prometheus Pushgateway after each step so Grafana dashboards show real-time token burn rate across all active sessions.

3. **Finish-Reason Anomaly Alert** — If `finish_reason` is `"length"` (truncated output), fire a Slack or ntfy alert because it indicates the model hit its output token cap mid-response and the next step may be degraded or looping.

4. **Datadog Custom Metric Emission** — Ship `crush.step.input_tokens`, `crush.step.output_tokens`, and `crush.step.index` as Datadog StatsD metrics for fleet-level agent cost observability.

5. **Step-Duration Percentile Tracking** — Record wall-clock time between PreStep and PostStep (using a temp file keyed by `session_id:step_index`) and append to a TSV log; a companion script can then compute p50/p95 inference latency to identify slow model responses or network degradation.

### ContextWindowFull

1. **Cost-Spike Alert Before Summarization** — When `tokens_used` is reported, calculate the estimated dollar cost of the context so far and send a Slack/PagerDuty alert if it exceeds a per-session budget, since compaction itself will incur another LLM call.

2. **Context Pressure Prometheus Gauge** — Emit a `crush_context_utilization` metric as `tokens_used / threshold` (a 0.0–1.0 ratio) to Pushgateway so Grafana can alert when sessions are regularly hitting near-100% context pressure and may need task decomposition.

3. **Append-Only Session Breadcrumb** — Write a structured line to a per-project log recording timestamp, session ID, `tokens_used`, and `threshold`; helps post-mortem analysis of which task types cause context blowout.

4. **Automatic Git Stash + Checkpoint Commit** — Before compaction erases conversation history, run `git stash` and create a WIP commit with a timestamped message so the code state at peak context is recoverable even if the model's summarization loses nuance.

5. **Notify Developer to Split the Task** — Send a desktop notification (via `notify-send` or `osascript`) advising the user that the context window is full and suggesting they break the current task into smaller sub-tasks rather than relying on lossy compaction.

### PreCompact

1. **Pre-Compaction State Snapshot** — Dump `git diff HEAD` and the current todo list to a timestamped file in `~/.crush/compaction-snapshots/` so you have a human-readable record of agent progress at the moment summarization begins, useful for auditing what context was lost.

2. **Distinguish Auto vs. Manual in Metrics** — Read `data.trigger` and emit separate `crush_compaction_total{trigger="auto"}` vs. `crush_compaction_total{trigger="manual"}` counters to Prometheus, letting you track how often the agent exhausts context autonomously vs. when users intervene.

3. **Pause Background Tasks During Compaction** — Signal a companion watcher process (e.g. via a lock file or `kill -STOP`) to suspend resource-intensive background jobs (test runners, watchers) so compaction's LLM call gets full network bandwidth.

4. **Log Compaction Frequency per Project** — Append an entry to a per-workspace counter file; if a project triggers more than N auto-compactions in a session, send an alert recommending the user add a `.crush/context.md` with standing instructions to keep context lean.

5. **Pre-Compaction Webhook to External Orchestrator** — POST the session ID and trigger type to an internal orchestration service so it can mark the session as "compacting" in a dashboard, preventing duplicate task dispatch while the agent's memory is being rebuilt.

### PostCompact

1. **Compaction Cost Accounting** — Record the wall-clock duration between PreCompact and PostCompact (using a temp file keyed by session ID) and log it alongside the trigger type; compaction itself is an LLM call, so tracking its latency and frequency reveals hidden cost centers.

2. **Reset Per-Session Step Counter** — After a successful compaction, zero out the step counter used by the PreStep rate-limit guard so the budget resets and the agent can continue without being throttled for pre-compaction steps already paid for.

3. **Post-Compaction Slack Summary** — Send a brief Slack message noting that a long-running session was compacted (including session ID and trigger), so teammates monitoring shared agent sessions know the context was summarized and to treat subsequent output as potentially lower-fidelity.

4. **Increment Compaction Counter in Datadog** — Emit a `crush.compaction.completed` increment with a `trigger` tag; a Datadog monitor can alert when auto-compaction rate exceeds a threshold (e.g. 3 per hour) signaling sessions are burning extra tokens due to poor task structure.

5. **Write a Human-Readable Compaction Event to Project Log** — Append a line like `[2026-04-07T14:23Z] context compacted (trigger=auto)` to a `.crush/session.log` file in the workspace, giving developers reviewing a git blame a clear signal of when agent memory was reset mid-session.

---

## Subagent & Notification

### SubagentStart

1. **Slack/Discord Spawn Alert** — Post a message to a team channel with the `agent_session_id` and timestamp whenever a subagent is spawned, so teammates can see parallel work in flight without watching the TUI.

2. **Cost-Attribution Ledger** — Append a start-time record keyed by `agent_session_id` to a local SQLite or JSONL file so that when `SubagentStop` fires you can compute per-subagent wall-clock time and estimated token cost, then roll it up to a project or user.

3. **Concurrency Cap / Rate Limiter** — Read a shared counter file; if the number of active subagents exceeds a configured threshold, write a warning to a status file or send a PagerDuty event so an operator can intervene before API rate limits are hit.

4. **Live Status Dashboard Update** — `curl` a self-hosted webhook (e.g. a Grafana annotation endpoint or a simple SSE server) with the `agent_session_id` so a browser dashboard shows a real-time list of running subagents with start times.

5. **Isolated Scratch Workspace** — Before the subagent does any file work, create a scratch directory named after `agent_session_id` and set an env-var file that the subagent can source, giving it a clean workspace that is easy to clean up on `SubagentStop`.

### SubagentStop

1. **Completion Desktop Notification** — Run `notify-send` (Linux) or `osascript` (macOS) with the `agent_session_id` so you get a system notification the moment a long-running background subagent finishes, even if you have switched to another app.

2. **Per-Subagent Cost Report** — Look up the matching start-time record written by `SubagentStart`, compute elapsed seconds, estimate token cost using a configurable rate, and append a one-line CSV row to `~/.crush/cost_log.csv` for later analysis.

3. **CI Artifact Upload** — After a subagent that writes test results or build artifacts finishes, `rclone` or `aws s3 cp` the scratch directory to an S3 bucket tagged with `agent_session_id`, keeping outputs organized per subagent run without manual cleanup.

4. **Failure Triage Webhook** — If the subagent stopped with an error, POST to a PagerDuty Events v2 endpoint with severity `warning` and the `agent_session_id` as the dedup key, so on-call engineers get a page without being woken for clean exits.

5. **Worktree / Sandbox Cleanup** — Remove the scratch directory created by the matching `SubagentStart` hook, ensuring no orphaned temp files accumulate across long multi-agent sessions.

### Notification

1. **Desktop Pop-Up** — Run `notify-send "Crush finished" "$message"` (or `osascript -e 'display notification ...'` on macOS) so you get an OS-level alert the moment the agent finishes a turn, useful when working in a different window.

2. **ntfy / Pushover Mobile Push** — `curl -d "$message" https://ntfy.sh/my-crush-topic` to forward the notification to a phone, handy for long-running headless sessions kicked off over SSH where you cannot watch the TUI.

3. **Slack DM on Turn Complete** — POST to a Slack incoming webhook with the session title and message so the turn-complete notification reaches you in Slack even when the terminal is minimized or on a remote machine.

4. **Speech Announcement** — Pipe the `message` field to `espeak` or macOS `say` for an audio cue that the agent has finished, useful in deep-focus coding sessions where glancing at the screen breaks flow.

5. **Structured Activity Log** — Append a JSON line `{"ts": ..., "session": ..., "message": ...}` to `~/.crush/activity.jsonl` on every notification so you have a full audit trail of when each session finished its turns, queryable later with `jq`.

### AgentError

1. **PagerDuty Critical Alert** — POST to the PagerDuty Events API with `severity: "critical"` and the `error` string as the summary whenever a genuine API or network failure occurs, ensuring on-call engineers are paged for infrastructure-level failures rather than ordinary cancellations.

2. **Slack Error Channel Ping** — Send the `error` field and `session_id` to a dedicated `#crush-errors` Slack channel so the team is aware of repeated API failures during a heavy multi-agent run without anyone having to watch logs.

3. **Exponential-Backoff Retry Trigger** — Write the failed `session_id` and error to a retry queue file; a separate daemon reads the queue and re-invokes `crush` with `--resume` after a delay, implementing automatic retry logic outside the agent itself.

4. **Error Rate Circuit Breaker** — Count how many `AgentError` events have fired in the last 60 seconds by checking timestamps in a small state file; if the rate exceeds a threshold, write a sentinel file that a `PreToolUse` hook checks to pause new work and alert operators before runaway API costs accumulate.

5. **Structured Error Telemetry** — POST the `error` string, `session_id`, and UTC timestamp to an internal observability endpoint (Datadog, Honeycomb, or a self-hosted OpenTelemetry collector) so engineers can track error frequency, correlate spikes with model deployments, and set alerts on SLOs.

---

## Filesystem & Config

### CwdChanged

1. **direnv Integration** — On every directory change, run `direnv allow && direnv exec . env` to load or unload `.envrc` variables automatically, so the agent always operates with the correct environment for the current project without requiring a shell restart.

2. **Automatic Runtime Version Switch** — Parse `.nvmrc`, `.python-version`, or `.ruby-version` in the new `cwd` and invoke the appropriate version manager (`nvm use`, `pyenv local`, `rbenv local`) to switch runtimes, preventing mismatches when the agent moves between polyglot monorepo packages.

3. **Git Repository Context Update** — On `cwd` change, extract the current git remote URL and branch, then write a short context file (e.g. `~/.cache/crush/repo-context.json`) that other tools can read to know which project is active, enabling integrations like Slack status updates or Linear project switching.

4. **Session Audit Log** — Append a timestamped entry (project name, previous cwd, new cwd) to a CSV audit trail whenever the working directory changes, giving teams an automatic record of which repositories the agent touched during a session.

5. **tmux / Terminal Title Update** — Run a shell command to set the terminal or tmux pane title to the new directory's project name, keeping the developer oriented when Crush moves across multiple directories within a single session.

### TaskCreated

1. **Linear Issue Creation** — POST to the Linear API with the task title mapped to a new issue in the current project, so every task the agent creates is automatically reflected in the team's issue tracker without manual copy-paste.

2. **Desktop Notification** — Send a macOS/Linux notification (`notify-send` or `osascript`) with the task title so developers watching the session from another window know what the agent has queued up next.

3. **Append to Markdown Task Log** — Write the task title and a timestamp to a `TASKS.md` file in the repo root, building a human-readable in-repo audit trail of what the agent decided to do during the session.

4. **GitHub Issue Creation** — Use `gh issue create` to open a draft GitHub issue for each task, linking agent-generated work items directly to the repository so they appear in PR-related discussions and project boards.

5. **Time Tracker Integration** — Start a new time-tracking entry via the Toggl or Clockify REST API using the task title as the entry description, enabling automatic time attribution without requiring the developer to manually start a timer.

### TaskCompleted

1. **Linear Issue Transition** — Call the Linear API to move the corresponding issue to "Done" when the agent marks a task complete, keeping the issue tracker synchronized with agent progress in real time.

2. **Slack / Teams Notification** — POST a completion message to a team webhook channel with the task title and elapsed time, giving the team visibility into what the agent accomplished without them needing to watch the TUI.

3. **Trigger CI/CD Pipeline** — Invoke a `gh workflow run` or `curl` against a pipeline API endpoint when a task completes, useful for tasks that should immediately kick off a build, test run, or deployment upon finishing.

4. **Commit on Task Boundary** — Run a non-interactive `git add` and `git commit -m "chore: <task_title>"` to create a granular commit at each task boundary, turning the task list into a clean, reviewable commit history.

5. **Stop Time Tracker** — Stop the active Toggl/Clockify entry that was started on `TaskCreated`, automatically closing the time record and capturing actual duration without any developer interaction.

### InstructionsLoaded

1. **Instruction Version Audit** — Log the file path, its git hash (`git log -1 --format=%H -- <path>`), and a timestamp to a local audit file every time instructions are loaded, creating a tamper-evident record of which version of rules governed a session.

2. **Conflict Checker** — Parse the loaded instructions file and compare its key directives against a known-good baseline using a diff tool, alerting the developer if a workspace CLAUDE.md overrides or contradicts global user instructions in a potentially dangerous way.

3. **Metrics Reporting** — Send the instruction file path and word count to an internal telemetry endpoint, helping platform teams track which projects have up-to-date AGENTS.md files and which have gone stale.

4. **Context Preload Cache Warm** — On load of an AGENTS.md or CLAUDE.md, immediately prime the token cache for the file before the agent starts generating, reducing first-response latency on cache-cold sessions.

5. **Instructions Changelog Diff** — Run `git diff HEAD~1 -- <path>` and write the diff to a transient file, giving the agent (or the developer) a quick summary of what changed since the last session started — useful for long-running projects with frequently updated instructions.

### FileChanged

1. **Auto-Format on Write** — When a `.go`, `.ts`, or `.py` file is saved, immediately invoke `gofmt`, `prettier`, or `ruff format` on the changed path so files are always formatted before the agent reads them back or stages them. Pair with `matcher.filename` to target specific files.

2. **Incremental Test Runner** — On any source file change, derive the relevant test file path (e.g. `foo.go` → `foo_test.go`) and run only those tests, giving the agent immediate feedback about correctness without a full test suite run.

3. **Schema Validation** — When a file matching `*.json`, `*.yaml`, or `*.toml` is written, run a schema validator (`ajv`, `yq`, `taplo`) against the file and write validation errors to a `.crush/lint-results` file the agent can check before proceeding.

4. **Documentation Sync** — When a public API source file changes (e.g. `routes.go`, `api.ts`), run a doc generation command (`swag init`, `typedoc`) to keep generated documentation in sync with the code the agent just wrote.

5. **Security Scan** — On any file write, run a lightweight secrets scanner (`trufflehog`, `detect-secrets`, `gitleaks`) against the changed path and emit a warning to stderr if credentials or API keys are detected before they can be committed.

### ConfigChange

1. **Config Diff Alert** — On every `crush.json` write, run a diff against the last known-good version stored in `~/.cache/crush/config-baseline.json` and print a human-readable summary of what changed, catching accidental or unauthorized configuration modifications.

2. **Workspace Config Inheritance Validator** — When a workspace-scoped `crush.json` is written, compare it against the global config and flag any keys that override security-sensitive settings (e.g. disabled safety checks, expanded tool permissions), giving the developer a chance to review before the agent acts on them.

3. **Config Version Control Commit** — Automatically run `git add crush.json && git commit -m "config: update crush.json"` in the workspace root whenever the workspace config changes, keeping configuration changes in the repo history alongside code changes.

4. **Environment-Specific Config Sync** — After a config write, push the new config values to a shared secrets manager (e.g. 1Password CLI `op item edit`, AWS SSM `aws ssm put-parameter`) so that CI pipelines and teammates pick up the same settings without manual distribution.

5. **Reload Hook Registry** — When `crush.json` changes, send a signal or HTTP request to any running sidecar processes (e.g. a local proxy, a test watcher daemon) to reload their configuration without requiring a restart, keeping all tools in sync with the updated Crush config.
