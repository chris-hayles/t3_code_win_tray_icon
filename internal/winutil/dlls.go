//go:build windows

package winutil

import "golang.org/x/sys/windows"

var (
	user32 = windows.NewLazySystemDLL("user32.dll")
)
