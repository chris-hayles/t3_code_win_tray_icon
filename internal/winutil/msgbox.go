//go:build windows

package winutil

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

var procMessageBoxW = user32.NewProc("MessageBoxW")

const (
	mbYesNo        = 0x00000004
	mbIconWarning  = 0x00000030
	mbIconQuestion = 0x00000020
	mbOK           = 0x00000000
	idYes          = 6
)

func MessageBoxYesNo(title, text string) (bool, error) {
	return messageBoxYesNo(title, text, mbYesNo|mbIconWarning)
}

func MessageBoxYesNoQuestion(title, text string) (bool, error) {
	return messageBoxYesNo(title, text, mbYesNo|mbIconQuestion)
}

func messageBoxYesNo(title, text string, flags uintptr) (bool, error) {
	t16, _ := windows.UTF16PtrFromString(title)
	x16, _ := windows.UTF16PtrFromString(text)
	ret, _, err := procMessageBoxW.Call(
		0,
		uintptr(unsafe.Pointer(x16)),
		uintptr(unsafe.Pointer(t16)),
		flags,
	)
	if ret == 0 {
		return false, fmt.Errorf("MessageBox: %v", err)
	}
	return ret == idYes, nil
}

func MessageBoxOK(title, text string) {
	t16, _ := windows.UTF16PtrFromString(title)
	x16, _ := windows.UTF16PtrFromString(text)
	procMessageBoxW.Call(
		0,
		uintptr(unsafe.Pointer(x16)),
		uintptr(unsafe.Pointer(t16)),
		mbOK|mbIconWarning,
	)
}
