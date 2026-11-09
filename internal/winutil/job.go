//go:build windows

package winutil

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

type Job struct {
	handle windows.Handle
}

func NewKillOnCloseJob() (*Job, error) {
	h, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return nil, err
	}
	var info windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	_, err = windows.SetInformationJobObject(
		h,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
	)
	if err != nil {
		windows.CloseHandle(h)
		return nil, err
	}
	return &Job{handle: h}, nil
}

func (j *Job) AssignProcess(process windows.Handle) error {
	return windows.AssignProcessToJobObject(j.handle, process)
}

func (j *Job) Close() error {
	if j.handle == 0 {
		return nil
	}
	return windows.CloseHandle(j.handle)
}

func (j *Job) Handle() windows.Handle {
	return j.handle
}

func (j *Job) TerminateTree() error {
	if j.handle == 0 {
		return nil
	}
	// Closing the job handle kills the process tree when kill-on-close is set.
	if err := windows.CloseHandle(j.handle); err != nil {
		return fmt.Errorf("close job: %w", err)
	}
	j.handle = 0
	return nil
}
