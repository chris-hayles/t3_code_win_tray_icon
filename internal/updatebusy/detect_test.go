package updatebusy

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBusy_InstallerUnderUpdaterCache(t *testing.T) {
	local := t.TempDir()
	d := &Detector{
		Now: func() time.Time { return time.Unix(1_700_000_000, 0) },
		Paths: Paths{
			LocalAppData: local,
			UserProfile:  t.TempDir(),
		},
		List: func() ([]Proc, error) {
			return []Proc{{
				Name:    "installer.exe",
				ExePath: filepath.Join(local, "t3code-updater", "installer.exe"),
			}}, nil
		},
		Stat: func(string) (os.FileInfo, error) { return nil, os.ErrNotExist },
	}
	busy, reason := d.Busy()
	if !busy {
		t.Fatal("expected busy while updater installer runs")
	}
	if reason == "" {
		t.Fatal("expected reason")
	}
}

func TestBusy_T3CodeSetupExeName(t *testing.T) {
	d := &Detector{
		Now:   func() time.Time { return time.Unix(1_700_000_000, 0) },
		Paths: Paths{LocalAppData: t.TempDir(), UserProfile: t.TempDir()},
		List: func() ([]Proc, error) {
			return []Proc{{Name: "T3-Code-0.1.7-x64.exe", ExePath: `C:\Temp\T3-Code-0.1.7-x64.exe`}}, nil
		},
		Stat: func(string) (os.FileInfo, error) { return nil, os.ErrNotExist },
	}
	busy, _ := d.Busy()
	if !busy {
		t.Fatal("expected busy for T3-Code setup process")
	}
}

func TestBusy_IgnoresStalePendingFilesAlone(t *testing.T) {
	// pending\ leftovers after a finished update must not block by themselves.
	local := t.TempDir()
	_ = os.MkdirAll(filepath.Join(local, "t3code-updater", "pending"), 0o755)
	_ = os.WriteFile(filepath.Join(local, "t3code-updater", "pending", "T3-Code-0.1.7-x64.exe"), []byte("x"), 0o644)
	d := &Detector{
		Now:   func() time.Time { return time.Unix(1_700_000_000, 0) },
		Paths: Paths{LocalAppData: local, UserProfile: t.TempDir()},
		List:  func() ([]Proc, error) { return nil, nil },
		Stat:  func(string) (os.FileInfo, error) { return nil, os.ErrNotExist },
	}
	busy, _ := d.Busy()
	if busy {
		t.Fatal("pending installer files alone must not mark busy")
	}
}

func TestBusy_MarkerChangeThenSettle(t *testing.T) {
	home := t.TempDir()
	marker := filepath.Join(home, ".t3", "runtime", "desktop-update-restart")
	if err := os.MkdirAll(filepath.Dir(marker), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(marker, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	stale := time.Unix(1_700_000_000, 0).Add(-24 * time.Hour)
	if err := os.Chtimes(marker, stale, stale); err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1_700_000_000, 0)
	d := &Detector{
		Now:   func() time.Time { return now },
		Paths: Paths{LocalAppData: t.TempDir(), UserProfile: home},
		List:  func() ([]Proc, error) { return nil, nil },
	}

	busy, _ := d.Busy()
	if busy {
		t.Fatal("stale marker on first observation must not block")
	}

	fresh := now.Add(-10 * time.Second)
	if err := os.Chtimes(marker, fresh, fresh); err != nil {
		t.Fatal(err)
	}
	busy, reason := d.Busy()
	if !busy {
		t.Fatalf("expected busy after marker change, reason=%q", reason)
	}

	now = now.Add(MarkerGrace + time.Second)
	busy, _ = d.Busy()
	if busy {
		t.Fatal("expected clear after marker grace elapsed")
	}
}

func TestBusy_MissingDesktopTarget(t *testing.T) {
	home := t.TempDir()
	cmd := filepath.Join(home, ".t3", "bin", "t3.cmd")
	if err := os.MkdirAll(filepath.Dir(cmd), 0o755); err != nil {
		t.Fatal(err)
	}
	missingExe := filepath.Join(home, "T3 Code (Nightly).exe")
	body := "@echo off\r\nif exist \"" + missingExe + "\" goto run\r\nexit /b 127\r\n:run\r\n\"" + missingExe + "\" server.asar %*\r\n"
	if err := os.WriteFile(cmd, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	d := &Detector{
		Now:   func() time.Time { return time.Unix(1_700_000_000, 0) },
		Paths: Paths{LocalAppData: t.TempDir(), UserProfile: home, T3Cmd: cmd},
		List:  func() ([]Proc, error) { return nil, nil },
	}
	busy, reason := d.Busy()
	if !busy {
		t.Fatal("expected busy when t3.cmd target exe is missing")
	}
	if reason == "" {
		t.Fatal("expected reason")
	}
}

func TestBusy_UpdatedRelaunch(t *testing.T) {
	d := &Detector{
		Now:   func() time.Time { return time.Unix(1_700_000_000, 0) },
		Paths: Paths{LocalAppData: t.TempDir(), UserProfile: t.TempDir()},
		List: func() ([]Proc, error) {
			return []Proc{{
				Name:    "T3 Code (Beta).exe",
				ExePath: `C:\Users\x\AppData\Local\Programs\t3-code-desktop\T3 Code (Beta).exe`,
				CmdLine: `"C:\Users\x\AppData\Local\Programs\t3-code-desktop\T3 Code (Beta).exe" --updated`,
			}}, nil
		},
		Stat: func(string) (os.FileInfo, error) { return nil, os.ErrNotExist },
	}
	busy, _ := d.Busy()
	if !busy {
		t.Fatal("expected busy during --updated relaunch")
	}
}
