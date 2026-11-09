package cmdquote

import (
	"strings"
)

// WindowsCmdQuoted invokes cmd.exe /d /c with a single argument that runs the
// batch file and its arguments. The batch path is quoted when needed.
func WindowsCmdQuoted(batchPath, batchArgs string) string {
	quotedBatch := `"` + strings.ReplaceAll(batchPath, `"`, "") + `"`
	command := quotedBatch
	if args := strings.TrimSpace(batchArgs); args != "" {
		command += " " + args
	}
	// cmd.exe /c strips one leading and one trailing quote. The extra pair
	// leaves the batch path and quoted arguments, including "This PC", intact.
	return `/d /c "` + command + `"`
}
