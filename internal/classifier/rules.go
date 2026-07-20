// Package classifier implements the auto-mode safety classifier that
// gates tool calls using a combination of static rules and an LLM
// classifier model.
package classifier

import (
	"encoding/json"
	"strings"

	"github.com/charmbracelet/crush/internal/fsext"
)

// Verdict is the outcome of a classification decision.
type Verdict int

const (
	// VerdictAllow means the action is safe to execute.
	VerdictAllow Verdict = iota
	// VerdictDeny means the action is clearly dangerous.
	VerdictDeny
	// VerdictEscalate means the action is ambiguous and needs human review.
	VerdictEscalate
	// VerdictClassify means the static rules could not decide and the
	// LLM classifier should be invoked.
	VerdictClassify
)

func (v Verdict) String() string {
	switch v {
	case VerdictAllow:
		return "ALLOW"
	case VerdictDeny:
		return "DENY"
	case VerdictEscalate:
		return "ESCALATE"
	case VerdictClassify:
		return "CLASSIFY"
	default:
		return "UNKNOWN"
	}
}

// readOnlyTools are tools whose actions never modify state.
var readOnlyTools = map[string]bool{
	"view":               true,
	"ls":                 true,
	"glob":               true,
	"grep":               true,
	"lsp_diagnostics":    true,
	"lsp_references":     true,
	"lsp_symbols":        true,
	"lsp_definition":     true,
	"lsp_call_hierarchy": true,
	"sourcegraph":        true,
	"crush_info":         true,
	"crush_logs":         true,
	"job_output":         true,
	"todos":              true,
	"question":           true,
	"list_mcp_resources": true,
	"read_mcp_resource":  true,
	"fetch":              true,
	"agentic_fetch":      true,
}

// inProjectWriteTools are tools that modify files and can be
// auto-approved when the target path is inside the working directory.
var inProjectWriteTools = map[string]bool{
	"edit":               true,
	"multiedit":          true,
	"write":              true,
	"lsp_rename":         true,
	"lsp_replace_symbol": true,
}

// ClassifyByRules applies static Tier 1 rules to decide whether a tool
// call should be auto-approved, sent to the classifier model, or
// immediately denied. It returns VerdictClassify when the static rules
// cannot make a decision.
func ClassifyByRules(toolName, action string, params any, path, workingDir string, trustProjectWrites bool) Verdict {
	if readOnlyTools[toolName] {
		return VerdictAllow
	}

	if action == "read" || action == "list" {
		return VerdictAllow
	}

	if trustProjectWrites && inProjectWriteTools[toolName] {
		if isInProject(path, workingDir) {
			return VerdictAllow
		}
	}

	// job_kill is low-risk; it only kills background shells that Crush
	// spawned.
	if toolName == "job_kill" {
		return VerdictAllow
	}

	// lsp_restart is safe; it only restarts language servers.
	if toolName == "lsp_restart" {
		return VerdictAllow
	}

	// download brings external content to local disk; the classifier
	// should evaluate the URL and destination.
	if toolName == "download" {
		return VerdictClassify
	}

	// bash commands are the most variable-risk tool. The bash tool
	// already handles safe-command detection before reaching the
	// permission service, so anything that gets here is not in the
	// safe-command list. Check for statically dangerous patterns
	// before falling through to the classifier model.
	if toolName == "bash" {
		if cmd := extractBashCommand(params); cmd != "" {
			if IsDangerousCommand(cmd) {
				return VerdictDeny
			}
		}
		return VerdictClassify
	}

	// MCP tools are opaque third-party tools. Always classify.
	if strings.HasPrefix(toolName, "mcp_") {
		return VerdictClassify
	}

	// Any unrecognized tool goes to the classifier.
	return VerdictClassify
}

// isInProject reports whether path is inside workingDir.
func isInProject(path, workingDir string) bool {
	if path == "" || workingDir == "" {
		return false
	}
	return fsext.HasPrefix(path, workingDir)
}

// dangerousPatterns are command substrings that are always denied
// without reaching the LLM classifier. These are high-confidence
// patterns where no context can make them safe for auto-approval.
var dangerousPatterns = []string{
	"rm -rf",
	"rm -fr",
	"--force",
	"--no-verify",
	"push --force",
	"push -f",
	"rebase --onto",
	"reset --hard",
	"clean -fd",
	"clean -dfx",
	"chmod 777",
	"chmod -r",
	"chown ",
	"mkfs",
	"dd if=",
	"> /dev/",
	"truncate ",
	"shred ",
	":(){ :|:",

	// Package install patterns that bypass project-level dependency
	// management (global/user installs).
	"pip install",
	"pip3 install",
	"npm install --global",
	"npm install -g",
	"pnpm add --global",
	"pnpm add -g",
	"yarn global add",
	"cargo install",
	"gem install",
	"go install",
	"brew install",
}

// IsDangerousCommand checks whether a shell command matches any
// statically known dangerous pattern that should never be
// auto-approved.
func IsDangerousCommand(cmd string) bool {
	lower := strings.ToLower(strings.TrimSpace(cmd))

	for _, pattern := range dangerousPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}

	return false
}

// extractBashCommand extracts the "command" field from bash tool
// params. Params may be a struct with a Command field, a map, or raw
// JSON bytes.
func extractBashCommand(params any) string {
	if params == nil {
		return ""
	}

	// Try struct with Command field (BashPermissionsParams).
	type hasCommand struct {
		Command string `json:"command"`
	}
	data, err := json.Marshal(params)
	if err != nil {
		return ""
	}
	var c hasCommand
	if err := json.Unmarshal(data, &c); err != nil {
		return ""
	}
	return c.Command
}
