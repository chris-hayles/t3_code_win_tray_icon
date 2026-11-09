//go:build windows

package winutil

import (
	"golang.org/x/sys/windows"
)

// TrySingleInstance returns a handle that must be kept open, or nil if another instance exists.
func TrySingleInstance(name string) (windows.Handle, bool) {
	name16, _ := windows.UTF16PtrFromString(`Local\` + name)
	h, err := windows.CreateMutex(nil, false, name16)
	if err != nil {
		if err == windows.ERROR_ALREADY_EXISTS {
			if h != 0 {
				windows.CloseHandle(h)
			}
			return 0, false
		}
		return 0, false
	}
	if windows.GetLastError() == windows.ERROR_ALREADY_EXISTS {
		windows.CloseHandle(h)
		return 0, false
	}
	return h, true
}
