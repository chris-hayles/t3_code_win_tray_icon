//go:build windows

package winutil

import (
	"fmt"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	shell32           = windows.NewLazySystemDLL("shell32.dll")
	procShellExecuteW = shell32.NewProc("ShellExecuteW")
)

const swShowNormal = 1

func ShellExecuteOpen(target string) error {
	// A pairing URL such as t3code://pair?... is not a filesystem path.
	// filepath.Dir would turn it into a directory ShellExecute rejects.
	dir := ""
	if filepath.IsAbs(target) {
		dir = filepath.Dir(target)
	}
	target16, err := windows.UTF16PtrFromString(target)
	if err != nil {
		return err
	}
	var op16 *uint16
	if op, err := windows.UTF16PtrFromString("open"); err == nil {
		op16 = op
	}
	var dir16 *uint16
	if dir != "" {
		dir16, _ = windows.UTF16PtrFromString(dir)
	}
	ret, _, _ := procShellExecuteW.Call(
		0,
		uintptr(unsafe.Pointer(op16)),
		uintptr(unsafe.Pointer(target16)),
		0,
		uintptr(unsafe.Pointer(dir16)),
		swShowNormal,
	)
	if ret <= 32 {
		return fmt.Errorf("ShellExecute failed: %d", ret)
	}
	return nil
}

func OpenExplorerDir(dir string) error {
	return ShellExecuteOpen(dir)
}

func OpenDocument(path string) error {
	if _, err := os.Stat(path); err != nil {
		return err
	}
	return ShellExecuteOpen(path)
}
