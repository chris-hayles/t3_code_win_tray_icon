//go:build windows

package shortcut

import (
	"os"
	"path/filepath"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"t3tray/internal/paths"
)

func CreateStartupShortcut(exePath string) error {
	if exePath == "" {
		var err error
		exePath, err = os.Executable()
		if err != nil {
			return err
		}
	}
	target := paths.StartupShortcut()
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	ole.CoInitialize(0)
	defer ole.CoUninitialize()
	unknown, err := oleutil.CreateObject("WScript.Shell")
	if err != nil {
		return err
	}
	defer unknown.Release()
	shell, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return err
	}
	defer shell.Release()
	shortcut, err := oleutil.CallMethod(shell, "CreateShortcut", target)
	if err != nil {
		return err
	}
	sc := shortcut.ToIDispatch()
	defer sc.Release()
	if _, err := oleutil.PutProperty(sc, "TargetPath", exePath); err != nil {
		return err
	}
	if _, err := oleutil.PutProperty(sc, "WorkingDirectory", filepath.Dir(exePath)); err != nil {
		return err
	}
	_, err = oleutil.CallMethod(sc, "Save")
	return err
}

func ToggleStartupShortcut(enabled bool) error {
	target := paths.StartupShortcut()
	if enabled {
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		return CreateStartupShortcut(exe)
	}
	return Remove(target)
}
