//go:build windows

package winutil

import (
	"io"
	"os"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	createNoWindow   = 0x08000000
	createSuspended  = 0x00000004
	createUnicodeEnv = 0x00000400
)

// StartHiddenCmdWithJob runs cmd suspended, assigns to job, resumes primary thread.
func StartHiddenCmdWithJob(cmdArgs, workDir string, job *Job, stdout, stderr *os.File) (pid int, proc windows.Handle, err error) {
	cmdLine := `"` + comspec() + `" ` + cmdArgs
	cmdLine16, err := windows.UTF16PtrFromString(cmdLine)
	if err != nil {
		return 0, 0, err
	}
	var wd16 *uint16
	if workDir != "" {
		wd16, _ = windows.UTF16PtrFromString(workDir)
	}
	var si windows.StartupInfo
	si.Cb = uint32(unsafe.Sizeof(si))
	si.Flags = windows.STARTF_USESHOWWINDOW | windows.STARTF_USESTDHANDLES
	si.ShowWindow = windows.SW_HIDE
	nul, err := openInheritableNUL()
	if err != nil {
		return 0, 0, err
	}
	defer windows.CloseHandle(nul)
	si.StdInput = nul
	if stdout != nil {
		si.StdOutput = windows.Handle(stdout.Fd())
		if err := makeInheritable(si.StdOutput); err != nil {
			return 0, 0, err
		}
	}
	if stderr != nil {
		si.StdErr = windows.Handle(stderr.Fd())
		if err := makeInheritable(si.StdErr); err != nil {
			return 0, 0, err
		}
	}
	if si.StdOutput == 0 {
		si.StdOutput = nul
	}
	if si.StdErr == 0 {
		si.StdErr = nul
	}
	flags := uint32(createNoWindow | createUnicodeEnv | createSuspended)
	var pi windows.ProcessInformation
	err = windows.CreateProcess(nil, cmdLine16, nil, nil, true, flags, nil, wd16, &si, &pi)
	if err != nil {
		return 0, 0, err
	}
	if job != nil {
		if err := job.AssignProcess(pi.Process); err != nil {
			windows.TerminateProcess(pi.Process, 1)
			windows.CloseHandle(pi.Process)
			windows.CloseHandle(pi.Thread)
			return 0, 0, err
		}
	}
	k32 := windows.NewLazySystemDLL("kernel32.dll")
	resumeThread := k32.NewProc("ResumeThread")
	ret, _, e := resumeThread.Call(uintptr(pi.Thread))
	if ret == ^uintptr(0) {
		windows.TerminateProcess(pi.Process, 1)
		windows.CloseHandle(pi.Process)
		windows.CloseHandle(pi.Thread)
		return 0, 0, e
	}
	windows.CloseHandle(pi.Thread)
	return int(pi.ProcessId), pi.Process, nil
}

func ProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(h)
	var code uint32
	if err := windows.GetExitCodeProcess(h, &code); err != nil {
		return false
	}
	const stillActive = 259
	return code == stillActive
}

func IsPIDInJob(pid int, job *Job) bool {
	if job == nil || job.handle == 0 || pid <= 0 {
		return false
	}
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(h)
	var result int32
	if err := isProcessInJob(h, job.handle, &result); err != nil {
		return false
	}
	return result != 0
}

func isProcessInJob(process, job windows.Handle, result *int32) error {
	k32 := windows.NewLazySystemDLL("kernel32.dll")
	proc := k32.NewProc("IsProcessInJob")
	r, _, e := proc.Call(uintptr(process), uintptr(job), uintptr(unsafe.Pointer(result)))
	if r == 0 {
		return e
	}
	return nil
}

// RunHiddenCapture runs hidden cmd.exe with args (the `/d /c ...` portion) and captures stdout+stderr.
func RunHiddenCapture(cmdArgs, workDir string) ([]byte, int, error) {
	cmdLine := `"` + comspec() + `" ` + cmdArgs
	cmdLine16, err := windows.UTF16PtrFromString(cmdLine)
	if err != nil {
		return nil, -1, err
	}
	var wd16 *uint16
	if workDir != "" {
		wd16, _ = windows.UTF16PtrFromString(workDir)
	}
	var sa windows.SecurityAttributes
	sa.Length = uint32(unsafe.Sizeof(sa))
	sa.InheritHandle = 1
	var rd, wr windows.Handle
	if err := windows.CreatePipe(&rd, &wr, &sa, 0); err != nil {
		return nil, -1, err
	}
	// Only the write end should be inherited. The child holding the read end deadlocks the capture.
	if err := windows.SetHandleInformation(rd, windows.HANDLE_FLAG_INHERIT, 0); err != nil {
		windows.CloseHandle(rd)
		windows.CloseHandle(wr)
		return nil, -1, err
	}
	readFile := os.NewFile(uintptr(rd), "r")
	writeHandle := wr
	nul, err := openInheritableNUL()
	if err != nil {
		readFile.Close()
		windows.CloseHandle(writeHandle)
		return nil, -1, err
	}
	defer windows.CloseHandle(nul)
	var si windows.StartupInfo
	si.Cb = uint32(unsafe.Sizeof(si))
	si.Flags = windows.STARTF_USESHOWWINDOW | windows.STARTF_USESTDHANDLES
	si.ShowWindow = windows.SW_HIDE
	si.StdOutput = writeHandle
	si.StdErr = writeHandle
	si.StdInput = nul
	var pi windows.ProcessInformation
	flags := uint32(createNoWindow | createUnicodeEnv)
	err = windows.CreateProcess(nil, cmdLine16, nil, nil, true, flags, nil, wd16, &si, &pi)
	windows.CloseHandle(writeHandle)
	if err != nil {
		readFile.Close()
		return nil, -1, err
	}
	windows.CloseHandle(pi.Thread)
	out, _ := io.ReadAll(readFile)
	readFile.Close()
	windows.WaitForSingleObject(pi.Process, windows.INFINITE)
	var code uint32
	windows.GetExitCodeProcess(pi.Process, &code)
	windows.CloseHandle(pi.Process)
	return out, int(code), nil
}

func ExeBaseMatches(cmdLine, exeName string) bool {
	return strings.Contains(strings.ToLower(cmdLine), strings.ToLower(exeName))
}

func comspec() string {
	if p := os.Getenv("COMSPEC"); p != "" {
		return p
	}
	return `C:\Windows\System32\cmd.exe`
}

func makeInheritable(h windows.Handle) error {
	if h == 0 {
		return nil
	}
	return windows.SetHandleInformation(h, windows.HANDLE_FLAG_INHERIT, windows.HANDLE_FLAG_INHERIT)
}

func openInheritableNUL() (windows.Handle, error) {
	p, err := windows.UTF16PtrFromString(`\\.\NUL`)
	if err != nil {
		return 0, err
	}
	h, err := windows.CreateFile(p, windows.GENERIC_READ|windows.GENERIC_WRITE, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, 0, 0)
	if err != nil {
		return 0, err
	}
	if err := makeInheritable(h); err != nil {
		windows.CloseHandle(h)
		return 0, err
	}
	return h, nil
}
