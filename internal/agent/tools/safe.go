package tools

import (
	"runtime"
	"slices"
	"strings"
)

var safeCommands = []string{
	// Bash builtins and core utils
	"cal",
	"date",
	"df",
	"du",
	"echo",
	"env",
	"free",
	"groups",
	"hostname",
	"id",
	"kill",
	"killall",
	"ls",
	"nice",
	"nohup",
	"printenv",
	"ps",
	"pwd",
	"set",
	"time",
	"timeout",
	"top",
	"type",
	"uname",
	"unset",
	"uptime",
	"whatis",
	"whereis",
	"which",
	"whoami",

	// Git
	"git blame",
	"git branch",
	"git config --get",
	"git config --list",
	"git describe",
	"git diff",
	"git grep",
	"git log",
	"git ls-files",
	"git ls-remote",
	"git remote",
	"git rev-parse",
	"git shortlog",
	"git show",
	"git status",
	"git tag",
}

var chainingMetacharacters = []string{
	";",
	"|",
	"&&",
	"$(",
	"`",
}

// containsCommandChaining reports whether s contains shell metacharacters
// that enable command chaining or substitution.
func containsCommandChaining(s string) bool {
	return slices.ContainsFunc(chainingMetacharacters, func(c string) bool {
		return strings.Contains(s, c)
	})
}

// containsBannedCommand reports whether the command string contains any
// of the banned commands. This checks both segment-first-word positions
// (where commands typically appear) and all words in the command (to
// catch cases where hooks may rewrite command structure). Used in auto
// mode to enforce the banned command list before the permission service
// (and before hooks can bypass it).
// containsBannedCommand reports whether the command string contains any
// of the banned commands. This checks every word in the command against
// the banned set to catch commands regardless of how they are wrapped
// or prefixed by hooks. Used in auto mode to enforce the banned command
// list before the permission service.
func containsBannedCommand(cmd string) bool {
	bannedSet := make(map[string]struct{}, len(bannedCommands))
	for _, b := range bannedCommands {
		bannedSet[b] = struct{}{}
	}

	for _, word := range strings.Fields(cmd) {
		lower := strings.ToLower(word)
		if _, ok := bannedSet[lower]; ok {
			return true
		}
	}
	return false
}

// DangerousPatterns are command substrings that are always denied in
// auto mode without reaching the LLM classifier. These are
// high-confidence patterns where no context can make them safe for
// auto-approval.
var DangerousPatterns = []string{
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

	for _, pattern := range DangerousPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}

	return false
}

func init() {
	if runtime.GOOS == "windows" {
		safeCommands = append(
			safeCommands,
			// Windows-specific commands
			"ipconfig",
			"nslookup",
			"ping",
			"systeminfo",
			"tasklist",
			"where",
		)
	}
}
