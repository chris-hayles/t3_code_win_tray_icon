package t3cmdparse

import (
	"os"
	"regexp"
	"strings"
)

var quotedExeRE = regexp.MustCompile(`"([^"]+\.exe)"`)

// DesktopExeFromCmd reads t3.cmd and returns the quoted T3 Code executable path, if present.
func DesktopExeFromCmd(cmdPath string) (string, error) {
	data, err := os.ReadFile(cmdPath)
	if err != nil {
		return "", err
	}
	// First quoted .exe is typically the desktop app path set by T3 Code.
	if m := quotedExeRE.FindStringSubmatch(string(data)); len(m) >= 2 {
		return m[1], nil
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasSuffix(strings.ToLower(line), ".exe") {
			return strings.Trim(line, `"`), nil
		}
	}
	return "", os.ErrNotExist
}
