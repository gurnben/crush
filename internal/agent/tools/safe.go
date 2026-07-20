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
func containsBannedCommand(cmd string) bool {
	bannedSet := make(map[string]struct{}, len(bannedCommands))
	for _, b := range bannedCommands {
		bannedSet[b] = struct{}{}
	}

	// Check every word in the command against the banned set. This is
	// intentionally broad to catch commands regardless of how they are
	// wrapped or prefixed by hooks.
	for _, word := range strings.Fields(cmd) {
		lower := strings.ToLower(word)
		if _, ok := bannedSet[lower]; ok {
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
