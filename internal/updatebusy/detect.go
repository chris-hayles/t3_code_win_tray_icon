package updatebusy

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"t3tray/internal/t3cmdparse"
)

const (
	// MarkerGrace is how long after desktop-update-restart is touched we keep pausing.
	MarkerGrace = 2 * time.Minute
	// SettleAfterInstaller is extra quiet time after the last installer/busy signal.
	SettleAfterInstaller = 30 * time.Second
	// UpdatedRelaunchGrace is applied once when a desktop --updated process is first seen.
	// Electron keeps --updated for the life of that process, so later polls must not renew.
	UpdatedRelaunchGrace = 30 * time.Second
	// PollWhileBusy is the sleep used by the supervisor while updates are in progress.
	PollWhileBusy = 2 * time.Second
)

// Proc is a process snapshot used by detection (testable without live Windows APIs).
type Proc struct {
	PID     int32
	Name    string
	ExePath string
	CmdLine string
}

// Paths supplies filesystem locations. Nil fields fall back to env-based defaults.
type Paths struct {
	LocalAppData string
	UserProfile  string
	T3Cmd        string
}

// Detector decides whether the tray should delay restarting t3 serve.
type Detector struct {
	Now   func() time.Time
	Stat  func(string) (os.FileInfo, error)
	List  func() ([]Proc, error)
	Paths Paths

	lastMarkerMtime time.Time
	pauseUntil      time.Time
	// seenUpdated tracks desktop --updated process identities already granted a grace window.
	seenUpdated map[string]struct{}
}

// Busy reports whether serve restarts should wait, with a short reason for logs/UI.
func (d *Detector) Busy() (bool, string) {
	if d == nil {
		return false, ""
	}
	now := time.Now()
	if d.Now != nil {
		now = d.Now()
	}
	stat := os.Stat
	if d.Stat != nil {
		stat = d.Stat
	}

	if reason := d.cliTargetMissing(stat); reason != "" {
		return true, reason
	}

	if reason := d.installerBusy(); reason != "" {
		d.extendPause(now, SettleAfterInstaller)
		return true, reason
	}

	if reason := d.markerBusy(now, stat); reason != "" {
		return true, reason
	}

	if !d.pauseUntil.IsZero() && now.Before(d.pauseUntil) {
		return true, "waiting for T3 Code update to settle"
	}
	return false, ""
}

func (d *Detector) extendPause(now time.Time, grace time.Duration) {
	until := now.Add(grace)
	if until.After(d.pauseUntil) {
		d.pauseUntil = until
	}
}

func (d *Detector) localAppData() string {
	if d.Paths.LocalAppData != "" {
		return d.Paths.LocalAppData
	}
	return os.Getenv("LOCALAPPDATA")
}

func (d *Detector) userProfile() string {
	if d.Paths.UserProfile != "" {
		return d.Paths.UserProfile
	}
	return os.Getenv("USERPROFILE")
}

func (d *Detector) t3CmdPath() string {
	if d.Paths.T3Cmd != "" {
		return d.Paths.T3Cmd
	}
	return filepath.Join(d.userProfile(), ".t3", "bin", "t3.cmd")
}

func (d *Detector) markerPath() string {
	return filepath.Join(d.userProfile(), ".t3", "runtime", "desktop-update-restart")
}

func updaterCacheDirs() []string {
	return []string{"t3code-updater", "t3-code-desktop-updater"}
}

func (d *Detector) cliTargetMissing(stat func(string) (os.FileInfo, error)) string {
	cmdPath := d.t3CmdPath()
	if _, err := stat(cmdPath); err != nil {
		return ""
	}
	exe, err := t3cmdparse.DesktopExeFromCmd(cmdPath)
	if err != nil || exe == "" {
		return ""
	}
	if _, err := stat(exe); err != nil {
		return "t3.cmd target missing; open T3 Code once"
	}
	return ""
}

func (d *Detector) markerBusy(now time.Time, stat func(string) (os.FileInfo, error)) string {
	info, err := stat(d.markerPath())
	if err != nil {
		return ""
	}
	mt := info.ModTime()
	if d.lastMarkerMtime.IsZero() {
		d.lastMarkerMtime = mt
		// Stale leftover markers must not block forever on first observation.
		if now.Sub(mt) < MarkerGrace && now.Sub(mt) >= 0 {
			d.extendPause(now, MarkerGrace-now.Sub(mt))
			return "T3 Code update marker is recent"
		}
		return ""
	}
	if mt.After(d.lastMarkerMtime) {
		d.lastMarkerMtime = mt
		d.extendPause(mt, MarkerGrace)
		return "T3 Code update marker changed"
	}
	if !d.pauseUntil.IsZero() && now.Before(d.pauseUntil) {
		return "waiting for T3 Code update to settle"
	}
	return ""
}

func (d *Detector) installerBusy() string {
	if d.List == nil {
		return ""
	}
	procs, err := d.List()
	if err != nil {
		return ""
	}
	caches := updaterCacheDirs()
	local := strings.ToLower(filepath.Clean(d.localAppData()))
	liveUpdated := make(map[string]struct{})
	now := d.nowOr()
	for _, p := range procs {
		name := strings.ToLower(filepath.Base(p.Name))
		if name == "" {
			name = strings.ToLower(filepath.Base(p.ExePath))
		}
		path := strings.ToLower(filepath.Clean(p.ExePath))
		cmd := strings.ToLower(p.CmdLine)

		if isUpdaterInstallerName(name) {
			if pathUnderUpdaterCache(path, local, caches) || pathUnderUpdaterCache(cmd, local, caches) {
				return "T3 Code updater/installer is running"
			}
		}
		if strings.HasPrefix(name, "t3-code-") && strings.HasSuffix(name, ".exe") {
			return "T3 Code installer is running"
		}
		if strings.Contains(cmd, "--updated") && looksLikeT3Desktop(name, path, cmd) {
			key := updatedProcKey(p)
			liveUpdated[key] = struct{}{}
			if d.seenUpdated == nil {
				d.seenUpdated = map[string]struct{}{}
			}
			if _, seen := d.seenUpdated[key]; seen {
				// Same post-update desktop process is still alive; do not renew the pause.
				continue
			}
			d.seenUpdated[key] = struct{}{}
			d.extendPause(now, UpdatedRelaunchGrace)
			return "T3 Code is relaunching after an update"
		}
	}
	// Drop identities that exited so a later update can pause again.
	for key := range d.seenUpdated {
		if _, ok := liveUpdated[key]; !ok {
			delete(d.seenUpdated, key)
		}
	}
	return ""
}

func updatedProcKey(p Proc) string {
	if p.PID > 0 {
		return "pid:" + strconv.FormatInt(int64(p.PID), 10)
	}
	return "cmd:" + strings.ToLower(p.ExePath) + "\n" + strings.ToLower(p.CmdLine)
}

func (d *Detector) nowOr() time.Time {
	if d.Now != nil {
		return d.Now()
	}
	return time.Now()
}

func isUpdaterInstallerName(name string) bool {
	return name == "installer.exe" || strings.HasPrefix(name, "t3-code-")
}

func pathUnderUpdaterCache(path, localAppData string, caches []string) bool {
	if path == "" || localAppData == "" {
		return false
	}
	for _, cache := range caches {
		root := strings.ToLower(filepath.Clean(filepath.Join(localAppData, cache)))
		if path == root || strings.HasPrefix(path, root+string(filepath.Separator)) {
			return true
		}
		// Command lines may use mixed separators.
		if strings.Contains(path, strings.ToLower(cache)) {
			return true
		}
	}
	return false
}

func looksLikeT3Desktop(name, path, cmd string) bool {
	if strings.Contains(name, "t3 code") || strings.Contains(path, "t3-code-desktop") || strings.Contains(cmd, "t3-code-desktop") {
		return true
	}
	return strings.Contains(cmd, "t3 code")
}
