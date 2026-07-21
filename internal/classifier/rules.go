// Package classifier implements the auto-mode safety classifier that
// gates tool calls using a combination of static rules and an LLM
// classifier model.
package classifier

import (
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
// Populated from tools.ReadOnlyToolNames() at Service construction
// time to avoid import cycles.
var readOnlyTools map[string]bool

// inProjectWriteTools are tools that modify files and can be
// auto-approved when the target path is inside the working directory.
// Populated from tools.InProjectWriteToolNames() at Service
// construction time.
var inProjectWriteTools map[string]bool

// lowRiskTools are tools that manage Crush's own state and can be
// auto-approved without classification. Populated from
// tools.LowRiskToolNames() at Service construction time.
var lowRiskTools map[string]bool

// InitToolCategories sets the tool category maps from the canonical
// lists defined in the tools package. This must be called before
// ClassifyByRules is used. It exists to break the import cycle
// between classifier and tools.
func InitToolCategories(readOnly, inProjectWrite, lowRisk []string) {
	readOnlyTools = toSet(readOnly)
	inProjectWriteTools = toSet(inProjectWrite)
	lowRiskTools = toSet(lowRisk)
}

func toSet(names []string) map[string]bool {
	m := make(map[string]bool, len(names))
	for _, n := range names {
		m[n] = true
	}
	return m
}

// ClassifyByRules applies static Tier 1 rules to decide whether a tool
// call should be auto-approved, sent to the classifier model, or
// immediately denied. It returns VerdictClassify when the static rules
// cannot make a decision.
func ClassifyByRules(toolName, action, path, workingDir string, trustProjectWrites bool) Verdict {
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

	// Low-risk tools that manage Crush's own state.
	if lowRiskTools[toolName] {
		return VerdictAllow
	}

	// download brings external content to local disk; the classifier
	// should evaluate the URL and destination.
	if toolName == "download" {
		return VerdictClassify
	}

	// bash commands are the most variable-risk tool. The bash tool
	// already handles safe-command detection and dangerous-pattern
	// matching before reaching the permission service, so anything
	// that gets here is not in the safe-command or dangerous-pattern
	// lists and needs LLM classification.
	if toolName == "bash" {
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
