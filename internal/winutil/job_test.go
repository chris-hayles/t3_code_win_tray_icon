//go:build windows

package winutil

import (
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestNewKillOnCloseJobSetsLimit(t *testing.T) {
	job, err := NewKillOnCloseJob()
	if err != nil {
		t.Fatal(err)
	}
	defer job.Close()

	var info windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
	err = windows.QueryInformationJobObject(
		job.Handle(),
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if info.BasicLimitInformation.LimitFlags&windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE == 0 {
		t.Fatalf("kill-on-close not set, flags=%#x", info.BasicLimitInformation.LimitFlags)
	}
	if info.BasicLimitInformation.LimitFlags&windows.JOB_OBJECT_LIMIT_BREAKAWAY_OK != 0 {
		t.Fatal("breakaway is allowed")
	}
}
