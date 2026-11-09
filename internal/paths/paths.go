package paths

import (
	"os"
	"path/filepath"
)

func UserProfile() string {
	return os.Getenv("USERPROFILE")
}

func LocalAppData() string {
	return os.Getenv("LOCALAPPDATA")
}

func AppData() string {
	return os.Getenv("APPDATA")
}

func T3Cmd() string {
	return filepath.Join(UserProfile(), ".t3", "bin", "t3.cmd")
}

func ServerRuntimeJSON() string {
	return filepath.Join(UserProfile(), ".t3", "userdata", "server-runtime.json")
}

func DesktopSettingsJSON() string {
	return filepath.Join(UserProfile(), ".t3", "userdata", "desktop-settings.json")
}

func TrayDataDir() string {
	return filepath.Join(LocalAppData(), "T3ServeTray")
}

func ServeLog() string {
	return filepath.Join(TrayDataDir(), "t3-serve.log")
}

func SupervisorLog() string {
	return filepath.Join(TrayDataDir(), "supervisor.log")
}

func StartupShortcut() string {
	return filepath.Join(AppData(), "Microsoft", "Windows", "Start Menu", "Programs", "Startup", "T3 Serve Tray.lnk")
}

func T3CodeFallbackExe() string {
	return filepath.Join(LocalAppData(), "Programs", "t3code", "T3 Code (Beta).exe")
}

const ServeArgs = "serve --host 127.0.0.1"
