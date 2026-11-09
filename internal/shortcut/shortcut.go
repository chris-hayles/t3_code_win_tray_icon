package shortcut

import (
	"errors"
	"os"
	"path/filepath"
)

// StartupPath returns the per-user Startup shortcut path for tests and callers.
func StartupPath(appData string) string {
	if appData == "" {
		appData = os.Getenv("APPDATA")
	}
	return filepath.Join(appData, "Microsoft", "Windows", "Start Menu", "Programs", "Startup", "T3 Serve Tray.lnk")
}

func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func Remove(path string) error {
	err := os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
