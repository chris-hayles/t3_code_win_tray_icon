//go:build windows

package winutil

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	procEnumWindows              = user32.NewProc("EnumWindows")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procPostMessageW             = user32.NewProc("PostMessageW")
)

const wmClose = 0x0010

func PostCloseToProcessWindows(pid int32) bool {
	var sent bool
	cb := windows.NewCallback(func(hwnd uintptr, _ uintptr) uintptr {
		var winPID uint32
		procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&winPID)))
		if int32(winPID) == pid {
			if isTopLevel(hwnd) {
				procPostMessageW.Call(hwnd, wmClose, 0, 0)
				sent = true
			}
		}
		return 1
	})
	procEnumWindows.Call(cb, 0)
	return sent
}

func isTopLevel(hwnd uintptr) bool {
	procGetParent := user32.NewProc("GetParent")
	p, _, _ := procGetParent.Call(hwnd)
	return p == 0
}
