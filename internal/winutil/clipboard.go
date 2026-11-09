//go:build windows

package winutil

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	cfUnicodeText = 13
	gmemMoveable  = 0x0002
)

func SetClipboardText(text string) error {
	user32 := windows.NewLazySystemDLL("user32.dll")
	kernel32 := windows.NewLazySystemDLL("kernel32.dll")
	openClipboard := user32.NewProc("OpenClipboard")
	closeClipboard := user32.NewProc("CloseClipboard")
	emptyClipboard := user32.NewProc("EmptyClipboard")
	setClipboardData := user32.NewProc("SetClipboardData")
	globalAlloc := kernel32.NewProc("GlobalAlloc")
	globalFree := kernel32.NewProc("GlobalFree")
	globalLock := kernel32.NewProc("GlobalLock")
	globalUnlock := kernel32.NewProc("GlobalUnlock")

	utf16, err := windows.UTF16FromString(text)
	if err != nil {
		return err
	}
	n := len(utf16) * int(unsafe.Sizeof(utf16[0]))
	r, _, e := openClipboard.Call(0)
	if r == 0 {
		return fmt.Errorf("OpenClipboard: %v", e)
	}
	defer closeClipboard.Call()
	emptyClipboard.Call()
	h, _, e := globalAlloc.Call(gmemMoveable, uintptr(n))
	if h == 0 {
		return fmt.Errorf("GlobalAlloc: %v", e)
	}
	ptr, _, e := globalLock.Call(h)
	if ptr == 0 {
		return fmt.Errorf("GlobalLock: %v", e)
	}
	dest := unsafe.Slice((*uint16)(*(*unsafe.Pointer)(unsafe.Pointer(&ptr))), len(utf16))
	copy(dest, utf16)
	globalUnlock.Call(h)
	r, _, e = setClipboardData.Call(cfUnicodeText, h)
	if r == 0 {
		globalFree.Call(h)
		return fmt.Errorf("SetClipboardData: %v", e)
	}
	return nil
}
