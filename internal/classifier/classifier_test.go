package classifier

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClassifyByRules(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		toolName           string
		action             string
		params             any
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
			name:     "bash safe command goes to classifier",
			toolName: "bash",
			action:   "execute",
			params:   map[string]string{"command": "npm test"},
			expected: VerdictClassify,
		},
		{
			name:     "bash rm -rf is denied by static rules",
			toolName: "bash",
			action:   "execute",
			params:   map[string]string{"command": "rm -rf /tmp/something"},
			expected: VerdictDeny,
		},
		{
			name:     "bash git push --force is denied by static rules",
			toolName: "bash",
			action:   "execute",
			params:   map[string]string{"command": "git push --force origin main"},
			expected: VerdictDeny,
		},
		{
			name:     "bash git push -f is denied by static rules",
			toolName: "bash",
			action:   "execute",
			params:   map[string]string{"command": "git push -f origin main"},
			expected: VerdictDeny,
		},
		{
			name:     "bash chmod 777 is denied by static rules",
			toolName: "bash",
			action:   "execute",
			params:   map[string]string{"command": "chmod 777 /tmp"},
			expected: VerdictDeny,
		},
		{
			name:     "bash git reset --hard is denied",
			toolName: "bash",
			action:   "execute",
			params:   map[string]string{"command": "git reset --hard HEAD~3"},
			expected: VerdictDeny,
		},
		{
			name:     "bash nil params goes to classifier",
			toolName: "bash",
			action:   "execute",
			params:   nil,
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
			got := ClassifyByRules(tt.toolName, tt.action, tt.params, tt.path, tt.workingDir, tt.trustProjectWrites)
			assert.Equal(t, tt.expected, got, "ClassifyByRules(%q, %q, %v, %q, %q, %v)",
				tt.toolName, tt.action, tt.params, tt.path, tt.workingDir, tt.trustProjectWrites)
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

func TestIsDangerousCommand(t *testing.T) {
	t.Parallel()

	dangerous := []string{
		"rm -rf /tmp/foo",
		"rm -fr /home/user",
		"git push --force origin main",
		"git push -f origin main",
		"git reset --hard HEAD~3",
		"git clean -fd",
		"chmod 777 /tmp",
		"chmod -R 755 /var",
		"dd if=/dev/zero of=/dev/sda",
		"RM -RF /tmp/foo",
		"pip install requests",
		"pip install --user requests",
		"pip3 install flask",
		"npm install --global prettier",
		"npm install -g eslint",
		"cargo install ripgrep",
		"gem install rails",
		"go install golang.org/x/tools/...",
		"brew install jq",
	}
	for _, cmd := range dangerous {
		assert.True(t, IsDangerousCommand(cmd), "expected dangerous: %q", cmd)
	}

	safe := []string{
		"git push origin main",
		"git status",
		"npm test",
		"go build ./...",
		"mkdir -p /tmp/test",
		"cat /etc/hostname",
		"chmod 644 myfile.txt",
		"ls -la",
	}
	for _, cmd := range safe {
		assert.False(t, IsDangerousCommand(cmd), "expected safe: %q", cmd)
	}
}

func TestExtractBashCommand(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "echo hello", extractBashCommand(map[string]string{"command": "echo hello"}))
	assert.Equal(t, "", extractBashCommand(nil))
	assert.Equal(t, "", extractBashCommand(map[string]string{"other": "value"}))

	type bashParams struct {
		Command string `json:"command"`
	}
	assert.Equal(t, "git status", extractBashCommand(bashParams{Command: "git status"}))
}
