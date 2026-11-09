//go:build windows

package winutil

import "golang.org/x/sys/windows"

func CloseProcessHandle(h uintptr) {
	if h != 0 {
		_ = windows.CloseHandle(windows.Handle(h))
	}
}
