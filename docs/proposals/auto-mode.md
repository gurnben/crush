# Auto Mode for Crush

**Status**: Implemented
**Date**: July 2026

---

## Problem

Crush offers two permission extremes:

| Mode | Flag | Behavior |
|---|---|---|
| **Manual** | (default) | Prompts the user for every non-safe tool call |
| **Yolo** | `--yolo` / `-y` | Approves everything, no guardrails |

Research from Anthropic shows users approve ~93% of all permission
prompts, leading to "approval fatigue" where users rubber-stamp without
reading. Meanwhile, yolo mode provides zero protection.

**Goal**: Add an `auto` permission mode that uses a combination of
static rules and a classifier model to gate tool calls, catching
genuinely dangerous actions while letting safe ones run freely. This is
the same approach used by Claude Code's auto mode and (optionally)
Codex CLI's Guardian sub-agent.

---

## Design Principles

1. **Rules first, model second.** The classifier model is only invoked
   for the ~5-10% of tool calls that fall into the ambiguous middle
   ground. Crush's existing `safeCommands` allowlist, `bannedCommands`
   blocklist, and new static dangerous-pattern matching handle the
   clear-cut cases with zero latency.

2. **Fail closed.** If the classifier is unavailable (network error,
   rate limit, model failure), the action escalates to the user prompt,
   never silently allowed.

3. **Reasoning-blind.** The classifier does not see the agent's own
   reasoning or tool results. It sees only: tool call name, tool call
   input parameters, and the working directory context. This prevents
   the agent from talking the classifier into bad decisions and blocks
   prompt injection via tool outputs.

4. **Deny-and-continue.** Classifier denials return the reason to the
   agent as a tool error (`StopTurn = false`) so the agent can retry
   with a safer approach. Human denials set `StopTurn = true` and end
   the turn. This makes false positives survivable.

5. **Hooks cannot bypass safety.** PreToolUse hooks that return
   `decision=allow` still work as a fast path for the permission
   prompt, but the auto-mode classifier's static deny checks run
   before hook approvals in the permission waterfall. Additionally,
   the bash tool's pre-flight dangerous-command and banned-command
   checks run after hook input rewriting but before any permission
   check, creating an absolute barrier.

---

## Architecture

### Full Decision Pipeline

```
LLM produces tool call
  │
  ▼
┌─────────────────────────────────────────────┐
│ 1. hookedTool.Run()                         │
│                                             │
│ PreToolUse hooks fire (user-defined scripts) │
│ - May rewrite tool input (call.Input)       │
│ - May return decision=allow (stamps ctx)    │
│ - May return decision=deny (blocks call)    │
│                                             │
│ Inner tool.Run() called with (potentially   │
│ rewritten) input                            │
└──────────────────┬──────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────┐
│ 2. Bash Tool Pre-Flight (bash only)         │
│                                             │
│ a. Safe-command check (safe.go)             │
│    echo, git status, ls, pwd, etc.          │
│    → isSafeReadOnly = true → SKIP ALL       │
│                                             │
│ b. Auto-mode dangerous pattern check        │
│    IsDangerousCommand(params.Command)       │
│    rm -rf, --force, chmod 777, pip install… │
│    → ClassifierDeniedResponse (StopTurn=    │
│      false, agent retries)                  │
│                                             │
│ c. Auto-mode banned command check           │
│    containsBannedCommand(params.Command)    │
│    curl, wget, ssh, sudo, apt, etc.         │
│    → ClassifierDeniedResponse               │
│                                             │
│ Steps b-c are an absolute barrier that      │
│ hooks cannot bypass, even if the hook       │
│ rewrote the command and returned allow.     │
└──────────────────┬──────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────┐
│ 3. Permission Service Waterfall             │
│                                             │
│ a. Yolo skip (--yolo)                       │
│    → approve instantly                      │
│                                             │
│ b. Config allowlist                         │
│    (permissions.allowed_tools)              │
│    → approve instantly                      │
│                                             │
│ c. AUTO-MODE CLASSIFIER                     │
│    ┌─────────────────────────────────────┐  │
│    │ Tier 1: Static Rules (rules.go)     │  │
│    │                                     │  │
│    │ Read-only tool? → ALLOW             │  │
│    │ In-project file write? → ALLOW      │  │
│    │ Bash dangerous pattern? → DENY      │  │
│    │ Ambiguous? → CLASSIFY (Tier 2)      │  │
│    └──────────────┬──────────────────────┘  │
│                   │ CLASSIFY                 │
│    ┌──────────────▼──────────────────────┐  │
│    │ Tier 2: LLM Classifier (2-stage)    │  │
│    │                                     │  │
│    │ Stage 1: Fast filter                │  │
│    │   max_tokens=256, temperature=0     │  │
│    │   → ALLOW? done                     │  │
│    │   → DENY/ESCALATE? Stage 2          │  │
│    │                                     │  │
│    │ Stage 2: Chain-of-thought review    │  │
│    │   Full reasoning about risk         │  │
│    │   → ALLOW / DENY / ESCALATE         │  │
│    └─────────────────────────────────────┘  │
│                                             │
│    ALLOW → approve, notify UI               │
│    DENY  → deny, record denial,             │
│            check circuit breaker            │
│    ESCALATE → fall through to d             │
│                                             │
│ d. Hook pre-approval                        │
│    (decision=allow from PreToolUse hook)     │
│    → approve instantly                      │
│    Catches safe commands that hooks          │
│    approved but the classifier escalated on │
│                                             │
│ e. Session auto-approve (non-interactive)   │
│    → approve instantly                      │
│                                             │
│ f. Session permission cache                 │
│    (previous "Allow for session" match)     │
│    → approve instantly                      │
│                                             │
│ g. USER PROMPT                              │
│    Permission dialog:                       │
│    Allow / Allow for Session / Deny         │
│    Allow → StopTurn=false                   │
│    Deny  → StopTurn=true (turn ends)        │
└──────────────────┬──────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────┐
│ 4. Shell Execution (final barrier)          │
│                                             │
│ blockFuncs() — shell interpreter level      │
│ CommandsBlocker (bannedCommands exact match) │
│ ArgumentsBlocker (subcommand+flag combos)   │
│ → Hard error if matched                     │
└─────────────────────────────────────────────┘
```

### Why the ordering matters

- **Step 2a before 2b**: Safe commands (`echo`, `git status`) bypass
  everything with zero latency. No classifier call, no permission
  request.
- **Step 2b-c before step 3**: The bash tool's dangerous-command and
  banned-command checks run after hooks rewrite tool input but before
  the permission service. Hooks that return `decision=allow` cannot
  bypass these checks. This is the absolute safety floor.
- **Step 3c before step 3d**: The auto-mode classifier runs before
  hook approvals in the permission waterfall. This prevents a hook
  `decision=allow` from short-circuiting past the classifier's static
  deny list. If the classifier returns ALLOW, the call is approved
  immediately. If it returns ESCALATE, hook approvals at step 3d
  still work as a fast path for commands hooks legitimately approved.
- **Step 3g deny vs classifier deny**: Classifier denials use
  `StopTurn=false` (deny-and-continue) so the agent retries with a
  safer approach. User denials use `StopTurn=true` to end the turn,
  respecting explicit human refusal.

---

## Static Rule Classification (Tier 1)

### Read-Only Tools (Always Approve)

| Tool | Rationale |
|---|---|
| `view`, `ls`, `glob`, `grep` | Read-only filesystem operations |
| `lsp_diagnostics`, `lsp_references`, `lsp_symbols`, `lsp_definition`, `lsp_call_hierarchy` | Read-only code intelligence |
| `sourcegraph` | Read-only code search |
| `crush_info`, `crush_logs` | Read-only application state |
| `job_output`, `job_kill`, `lsp_restart` | Low-risk process management |
| `todos`, `question` | No side effects |
| `list_mcp_resources`, `read_mcp_resource` | Read-only MCP access |
| `fetch`, `agentic_fetch` | Read-only HTTP requests |

Any tool with `action == "read"` or `action == "list"` is also
auto-approved regardless of tool name.

### In-Project Write Tools (Approve When `trust_project_writes` Is True)

| Tool | Condition |
|---|---|
| `edit`, `multiedit`, `write` | `fsext.HasPrefix(path, workingDir)` |
| `lsp_rename`, `lsp_replace_symbol` | `fsext.HasPrefix(path, workingDir)` |

Out-of-project file writes go to the LLM classifier.

### Bash Pre-Flight Checks (Absolute Barriers)

These run in the bash tool itself, after hook input rewriting but
before the permission service. They cannot be bypassed.

**Dangerous patterns** (substring match, case-insensitive):

```
rm -rf, rm -fr, --force, --no-verify, push --force, push -f,
rebase --onto, reset --hard, clean -fd, clean -dfx, chmod 777,
chmod -r, chown, mkfs, dd if=, > /dev/, truncate, shred,
:(){ :|:, pip install, pip3 install, npm install --global,
npm install -g, pnpm add --global, pnpm add -g, yarn global add,
cargo install, gem install, go install, brew install
```

**Banned commands** (first word of any shell segment matches
`bannedCommands` list): `curl`, `wget`, `ssh`, `sudo`, `apt`,
`dnf`, `pacman`, and ~50 more from the existing shell blocklist.
The check scans all words in the command to catch cases where hooks
rewrite command structure.

### Tools That Go to the LLM Classifier

| Tool | Why |
|---|---|
| `bash` (non-safe, non-dangerous) | Variable risk |
| `download` | Downloads external content to local disk |
| `mcp_*` (any MCP tool) | Opaque third-party tools |
| Any unrecognized tool | Conservative default |

---

## LLM Classifier (Tier 2)

### Model Selection

The classifier defaults to the user's `small` model with constrained
parameters (`temperature=0`, `max_tokens=256`). An explicit
`classifier` model type in `crush.json` overrides this.

```jsonc
{
  "models": {
    "large": { "provider": "anthropic", "model": "claude-sonnet-4-20250514" },
    "small": { "provider": "anthropic", "model": "claude-haiku-4-20250414" },
    "classifier": { "provider": "anthropic", "model": "claude-haiku-4-20250414" }
  }
}
```

Resolution order: explicit `classifier` → `small` → `large`.

The classifier model can also be selected via the model chooser dialog
(Ctrl+L → Tab to the "Classifier" tab).

### Two-Stage Pipeline

**Stage 1 (Fast Filter)**: Sends the tool name, action, parameters
JSON, and working directory to the classifier with a policy template.
Expects a single-word response: ALLOW, DENY, or ESCALATE. If ALLOW,
the call proceeds immediately. If DENY or ESCALATE, Stage 2 fires.

**Stage 2 (Chain-of-Thought Review)**: Sends the same context plus
the Stage 1 verdict. The classifier reasons step-by-step and returns
a JSON object with `verdict` and `reason` fields. If DENY, the reason
is returned to the agent. If ESCALATE, falls through to hook approval
or user prompt. If unparseable, defaults to ESCALATE (fail-closed).

### Configurable Timeout

Default: 5 seconds. If the classifier model doesn't respond in time,
the action escalates to the user prompt.

---

## Deny-and-Continue

When the classifier or static rules deny a tool call, the denial
reason is returned to the agent as a tool error with `StopTurn=false`.
The agent's turn continues and it can retry with a safer approach.

```go
func NewClassifierDeniedResponse(reason string) fantasy.ToolResponse {
    msg := fmt.Sprintf(
        "Action blocked by auto-mode safety classifier: %s\n\n"+
            "Find a safer alternative approach. Do not attempt to "+
            "circumvent this restriction.",
        reason,
    )
    return fantasy.NewTextErrorResponse(msg)
}
```

This contrasts with user denials (`NewPermissionDeniedResponse`) which
set `StopTurn=true` and end the agent's turn.

---

## Circuit Breaker

The classifier tracks consecutive and total denials per session:

| Threshold | Default | Configurable |
|---|---|---|
| Consecutive denials | 3 | `auto.max_consecutive_denials` |
| Total denials | 20 | `auto.max_total_denials` |

When either threshold is exceeded, auto mode pauses for the remainder
of the session and the classifier returns `VerdictEscalate` for all
subsequent calls, falling back to manual prompts.

Any successful approval (classifier or user) resets the consecutive
counter. The total counter never resets within a session.

---

## Activation

### CLI Flags

```bash
crush              # manual mode (default)
crush --auto       # auto mode
crush -a           # short form
crush --yolo       # yolo mode (existing)
crush -y           # short form
```

`--auto` and `--yolo` are mutually exclusive.

### Command Palette

"Toggle Auto Mode" in the command palette (Ctrl+K). When enabled:
- Shows an `" A "` prompt icon (primary color) and "Auto mode"
  placeholder text
- Toggling auto mode on automatically disables yolo mode
- Builds the classifier model on demand if not already initialized

### crush.json

```jsonc
{
  "permissions": {
    "mode": "auto",
    "allowed_tools": ["bash:execute"],
    "auto": {
      "classifier_timeout": 5,
      "max_consecutive_denials": 3,
      "max_total_denials": 20,
      "trust_project_writes": true
    }
  }
}
```

CLI flags override the `mode` setting in the config file.

---

## Implementation

### New Package: `internal/classifier/`

| File | Purpose |
|---|---|
| `rules.go` | Verdict type, static rule engine, dangerous pattern matching, banned command extraction |
| `prompt.go` | Stage 1 and Stage 2 prompt templates, output parsers |
| `classifier.go` | Service struct, two-stage LLM pipeline, circuit breaker state |
| `classifier_test.go` | Unit tests for rules, parsing, dangerous command detection |

### Modified Packages

| Package | Changes |
|---|---|
| `internal/config` | `SelectedModelTypeClassifier`, `PermissionsMode` enum, `AutoModeConfig` struct, classifier model resolution |
| `internal/permission` | Classifier integration in `Request()` waterfall, `RequestResult` type, `AutoMode()`/`SetAutoMode()`/`SetClassifier()` on Service interface |
| `internal/agent/tools` | `NewClassifierDeniedResponse`, `containsBannedCommand`, `IsDangerousCommand` pre-flight in bash tool |
| `internal/agent` | `classifier_model.go` (builds LLM from config), `hooked_tool.go` (permissions plumbing), `coordinator.go` (permissions to hook wrapper) |
| `internal/app` | `EnsureClassifier()` method, auto-mode permission service creation |
| `internal/cmd` | `--auto` flag, mutual exclusivity with `--yolo` |
| `internal/proto` | `AutoMode` field on Workspace |
| `internal/workspace` | `PermissionAutoMode()`/`PermissionSetAutoMode()` on Workspace interface |
| `internal/ui/dialog` | `ActionToggleAutoMode`, command palette item, model chooser Classifier tab |
| `internal/ui/model` | `toggleAutoMode()`, `autoPromptFunc`, auto-mode placeholder text |
| `internal/ui/styles` | Auto-mode prompt styles (`PromptAutoIcon*`, `PromptAutoDots*`) |

---

## Test Results

Validated with a 10-command test suite:

| # | Command | Result |
|---|---|---|
| 1 | `echo "hello from auto mode"` | Auto-approved (safe-command list) |
| 2 | `git status` | Auto-approved (safe-command list) |
| 3 | `cat /etc/hostname` | Auto-approved (classifier/hook) |
| 4 | `rm -rf /tmp/...` | Classifier-denied (static pattern) |
| 5 | `chmod 777 /tmp` | Classifier-denied (static pattern) |
| 6 | `git push --force origin main` | Classifier-denied (static pattern) |
| 7 | `curl https://example.com` | Classifier-denied (banned command) |
| 8 | `pip install --user requests` | Classifier-denied (static pattern) |
| 9 | `go test ./...` | Auto-approved (classifier/hook) |
| 10 | `sudo ls /root` | Classifier-denied (banned command) |

All deny-and-continue responses allowed the agent to continue its
turn and attempt safer alternatives.

---

## Future Work

1. **Classifier decision caching.** Cache per-session classifier
   approvals keyed by the full command string to avoid re-classifying
   identical commands.

2. **User intent context.** Send the last 1-2 user messages to the
   classifier so it can assess whether a risky command aligns with
   what the user asked for.

3. **Telemetry.** Track classifier decisions (allow/deny/escalate),
   false positive rate (user overrides classifier denial), and latency
   impact for prompt tuning.

4. **`acceptEdits` mode.** A lighter variant that auto-approves file
   edits but prompts for bash. This is already achievable via
   `trust_project_writes: true` + manual mode, but a dedicated
   flag would simplify the UX.

5. **Client/remote protocol support.** `PermissionAutoMode` and
   `PermissionSetAutoMode` are stubbed in `ClientWorkspace` pending
   the server-side protocol extension.
