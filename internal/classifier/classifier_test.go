package classifier

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	// Initialize tool categories for tests since the canonical lists
	// live in the tools package which we can't import here.
	InitToolCategories(
		[]string{"view", "ls", "glob", "grep", "lsp_diagnostics", "lsp_references",
			"lsp_symbols", "lsp_definition", "lsp_call_hierarchy", "sourcegraph",
			"crush_info", "crush_logs", "job_output", "todos", "question",
			"list_mcp_resources", "read_mcp_resource", "fetch", "agentic_fetch"},
		[]string{"edit", "multiedit", "write", "lsp_rename", "lsp_replace_symbol"},
		[]string{"job_kill", "lsp_restart"},
	)
	m.Run()
}

func TestClassifyByRules(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		toolName           string
		action             string
		path               string
		workingDir         string
		trustProjectWrites bool
		expected           Verdict
	}{
		{
			name:     "read-only tool is always allowed",
			toolName: "view",
			action:   "read",
			expected: VerdictAllow,
		},
		{
			name:     "ls tool is always allowed",
			toolName: "ls",
			action:   "list",
			expected: VerdictAllow,
		},
		{
			name:     "glob tool is always allowed",
			toolName: "glob",
			action:   "",
			expected: VerdictAllow,
		},
		{
			name:     "grep tool is always allowed",
			toolName: "grep",
			action:   "",
			expected: VerdictAllow,
		},
		{
			name:     "fetch is always allowed",
			toolName: "fetch",
			action:   "fetch",
			expected: VerdictAllow,
		},
		{
			name:     "sourcegraph is always allowed",
			toolName: "sourcegraph",
			action:   "",
			expected: VerdictAllow,
		},
		{
			name:     "crush_info is always allowed",
			toolName: "crush_info",
			action:   "",
			expected: VerdictAllow,
		},
		{
			name:     "question is always allowed",
			toolName: "question",
			action:   "",
			expected: VerdictAllow,
		},
		{
			name:     "todos is always allowed",
			toolName: "todos",
			action:   "",
			expected: VerdictAllow,
		},
		{
			name:     "list_mcp_resources is always allowed",
			toolName: "list_mcp_resources",
			action:   "list",
			expected: VerdictAllow,
		},
		{
			name:     "read_mcp_resource is always allowed",
			toolName: "read_mcp_resource",
			action:   "read",
			expected: VerdictAllow,
		},
		{
			name:     "generic read action is allowed",
			toolName: "unknown_tool",
			action:   "read",
			expected: VerdictAllow,
		},
		{
			name:     "generic list action is allowed",
			toolName: "unknown_tool",
			action:   "list",
			expected: VerdictAllow,
		},
		{
			name:               "in-project edit is allowed when trusting project writes",
			toolName:           "edit",
			action:             "write",
			path:               "/project/src/main.go",
			workingDir:         "/project",
			trustProjectWrites: true,
			expected:           VerdictAllow,
		},
		{
			name:               "out-of-project edit goes to classifier",
			toolName:           "edit",
			action:             "write",
			path:               "/other/dir/file.go",
			workingDir:         "/project",
			trustProjectWrites: true,
			expected:           VerdictClassify,
		},
		{
			name:               "in-project edit goes to classifier when not trusting",
			toolName:           "edit",
			action:             "write",
			path:               "/project/src/main.go",
			workingDir:         "/project",
			trustProjectWrites: false,
			expected:           VerdictClassify,
		},
		{
			name:               "in-project write tool is allowed",
			toolName:           "write",
			action:             "write",
			path:               "/project/new_file.go",
			workingDir:         "/project",
			trustProjectWrites: true,
			expected:           VerdictAllow,
		},
		{
			name:               "in-project multiedit is allowed",
			toolName:           "multiedit",
			action:             "write",
			path:               "/project/src/main.go",
			workingDir:         "/project",
			trustProjectWrites: true,
			expected:           VerdictAllow,
		},
		{
			name:               "in-project lsp_rename is allowed",
			toolName:           "lsp_rename",
			action:             "",
			path:               "/project/src/main.go",
			workingDir:         "/project",
			trustProjectWrites: true,
			expected:           VerdictAllow,
		},
		{
			name:     "bash goes to classifier",
			toolName: "bash",
			action:   "execute",
			expected: VerdictClassify,
		},
		{
			name:     "download goes to classifier",
			toolName: "download",
			action:   "download",
			expected: VerdictClassify,
		},
		{
			name:     "MCP tool goes to classifier",
			toolName: "mcp_github_create_issue",
			action:   "execute",
			expected: VerdictClassify,
		},
		{
			name:     "unknown tool goes to classifier",
			toolName: "something_new",
			action:   "execute",
			expected: VerdictClassify,
		},
		{
			name:     "job_kill is always allowed",
			toolName: "job_kill",
			action:   "",
			expected: VerdictAllow,
		},
		{
			name:     "lsp_restart is always allowed",
			toolName: "lsp_restart",
			action:   "",
			expected: VerdictAllow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ClassifyByRules(tt.toolName, tt.action, tt.path, tt.workingDir, tt.trustProjectWrites)
			assert.Equal(t, tt.expected, got, "ClassifyByRules(%q, %q, %q, %q, %v)",
				tt.toolName, tt.action, tt.path, tt.workingDir, tt.trustProjectWrites)
		})
	}
}

func TestParseStage1(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected Verdict
	}{
		{"ALLOW", VerdictAllow},
		{"allow", VerdictAllow},
		{"  ALLOW  ", VerdictAllow},
		{"DENY", VerdictDeny},
		{"deny", VerdictDeny},
		{"ESCALATE", VerdictEscalate},
		{"escalate", VerdictEscalate},
		{"", VerdictEscalate},
		{"gibberish", VerdictEscalate},
		{"The action is ALLOW", VerdictAllow},
		{"I think DENY is appropriate", VerdictDeny},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got := ParseStage1(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestParseStage2(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		input          string
		expectedVerict Verdict
		expectReason   bool
	}{
		{
			name:           "valid allow JSON",
			input:          `{"verdict": "ALLOW", "reason": "standard test command"}`,
			expectedVerict: VerdictAllow,
			expectReason:   true,
		},
		{
			name:           "valid deny JSON",
			input:          `{"verdict": "DENY", "reason": "modifies system files"}`,
			expectedVerict: VerdictDeny,
			expectReason:   true,
		},
		{
			name:           "valid escalate JSON",
			input:          `{"verdict": "ESCALATE", "reason": "ambiguous intent"}`,
			expectedVerict: VerdictEscalate,
			expectReason:   true,
		},
		{
			name:           "unparseable output",
			input:          "I cannot determine the safety",
			expectedVerict: VerdictEscalate,
		},
		{
			name:           "empty output",
			input:          "",
			expectedVerict: VerdictEscalate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ParseStage2(tt.input)
			assert.Equal(t, tt.expectedVerict, got.Verdict)
			if tt.expectReason {
				assert.NotEmpty(t, got.Reason)
			}
		})
	}
}

func TestVerdictString(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "ALLOW", VerdictAllow.String())
	assert.Equal(t, "DENY", VerdictDeny.String())
	assert.Equal(t, "ESCALATE", VerdictEscalate.String())
	assert.Equal(t, "CLASSIFY", VerdictClassify.String())
}
