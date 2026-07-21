package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContainsCommandChaining(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"plain ls", "ls -la", false},
		{"plain echo", "echo hello world", false},
		{"plain pwd", "pwd", false},
		{"plain git status", "git status", false},
		{"ls with redirect", "ls > /tmp/out", false},
		{"ls with pipe", "ls | grep foo", true},
		{"ls with double ampersand", "ls && echo done", true},
		{"ls with semicolon", "ls; echo done", true},
		{"ls with pipe pipe", "ls || echo fail", true},
		{"ls with backticks", "ls `echo foo`", true},
		{"ls with subshell", "ls $(echo foo)", true},
		{"ls with background ampersand", "ls & echo done", false},
		{"rm -rf with && ls (rm first)", "rm -rf / && ls", true},
		{"redirect with ampersand gt", "ls &> /dev/null", false},
		{"redirect with gt ampersand", "ls >& /dev/null", false},
		{"simple kill", "kill 1234", false},
		{"kill with pipe", "kill 1234 | echo foo", true},
		{"git log", "git log --oneline", false},
		{"git log with pipe", "git log | head", true},
		{"empty string", "", false},
		{"dollar sign in argument", "echo $HOME", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := containsCommandChaining(tt.input)
			assert.Equal(t, tt.expected, got, "containsCommandChaining(%q)", tt.input)
		})
	}
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

func TestContainsBannedCommand(t *testing.T) {
	t.Parallel()

	assert.True(t, containsBannedCommand("curl https://example.com"))
	assert.True(t, containsBannedCommand("sudo ls /root"))
	assert.True(t, containsBannedCommand("wget https://example.com"))
	assert.True(t, containsBannedCommand("some-wrapper curl https://evil.com"))
	assert.False(t, containsBannedCommand("echo hello"))
	assert.False(t, containsBannedCommand("git status"))
	assert.False(t, containsBannedCommand("go test ./..."))
}
