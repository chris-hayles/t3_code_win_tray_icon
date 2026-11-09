//go:build windows

package attach

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/process"
	"t3tray/internal/cmdquote"
	"t3tray/internal/ownership"
	"t3tray/internal/pairlink"
	"t3tray/internal/paths"
	"t3tray/internal/settings"
	"t3tray/internal/supervisor"
	"t3tray/internal/t3cmdparse"
	"t3tray/internal/winutil"
)

const desktopExe = "T3 Code (Beta).exe"

const confirmText = "T3 Code will close and reopen.\r\n\r\n" +
	"Local environment will be turned off so the desktop stops running its own server.\r\n\r\n" +
	"A pairing prompt will open; confirm the 127.0.0.1 address.\r\n\r\n" +
	"Folder drag-and-drop into the composer is unavailable in this mode."

func ConfirmAttach() (bool, error) {
	return winutil.MessageBoxYesNo("Use this server in T3 Code", confirmText)
}

func RunAttachFlow(sup *supervisor.Supervisor) error {
	yes, err := ConfirmAttach()
	if err != nil || !yes {
		return nil
	}
	desktopServerPID, err := desktopServerBeforeClose(sup)
	if err != nil {
		return err
	}
	if err := closeDesktopUI(); err != nil {
		winutil.MessageBoxOK("T3 Serve Tray", err.Error())
		return err
	}
	if desktopServerPID > 0 {
		if err := waitForPIDExit(desktopServerPID, 20*time.Second); err != nil {
			msg := "The desktop server is still running; not changing settings."
			winutil.MessageBoxOK("T3 Serve Tray", msg)
			return fmt.Errorf(msg)
		}
	}
	if st, _, _ := ownership.ClassifyFile(paths.ServerRuntimeJSON(), winutil.ProcessAlive, sup.IsOwnedPID); st == ownership.Foreign {
		winutil.MessageBoxOK("T3 Serve Tray", "Another T3 server is still using this home.")
		return fmt.Errorf("foreign server")
	}
	if err := settings.MergeDisableLocalEnvironment(paths.DesktopSettingsJSON()); err != nil {
		winutil.MessageBoxOK("T3 Serve Tray", "Could not update desktop settings.")
		return err
	}
	sup.EnsureOwnedOrStart()
	if _, err := sup.WaitForLoopback(30 * time.Second); err != nil {
		winutil.MessageBoxOK("T3 Serve Tray", "Timed out waiting for the tray server on 127.0.0.1.")
		return err
	}
	_, link, err := mintPairLink()
	if err != nil {
		winutil.MessageBoxOK("T3 Serve Tray", err.Error())
		return err
	}
	if err := winutil.ShellExecuteOpen(link); err != nil {
		winutil.MessageBoxOK("T3 Serve Tray", "Could not open the pairing link in T3 Code.")
		return err
	}
	return nil
}

// CopyPairingLink mints a one-time link for the running loopback server,
// copies the pasteable URL, and offers to open it in T3 Code.
func CopyPairingLink(sup *supervisor.Supervisor) error {
	if _, err := os.Stat(paths.T3Cmd()); err != nil {
		winutil.MessageBoxOK("T3 Serve Tray", "Open T3 Code once so the t3 command is installed.")
		return err
	}
	if !loopbackReady() {
		sup.EnsureOwnedOrStart()
		if _, err := sup.WaitForLoopback(30 * time.Second); err != nil {
			winutil.MessageBoxOK("T3 Serve Tray", "The tray server is not running on 127.0.0.1 yet.")
			return err
		}
	}
	pairURL, appLink, err := mintPairLink()
	if err != nil {
		winutil.MessageBoxOK("T3 Serve Tray", err.Error())
		return err
	}
	if err := winutil.SetClipboardText(pairURL); err != nil {
		winutil.MessageBoxOK("T3 Serve Tray", "Could not copy the pairing link.\r\n\r\n"+pairURL)
		return err
	}
	yes, err := winutil.MessageBoxYesNoQuestion("Pairing link",
		"Pairing link copied to the clipboard.\r\n\r\n"+
			"In T3 Code, open Settings → Environments → Add environment and paste it.\r\n\r\n"+
			pairURL+"\r\n\r\n"+
			"Open this link in T3 Code now?")
	if err != nil || !yes {
		return err
	}
	if err := winutil.ShellExecuteOpen(appLink); err != nil {
		winutil.MessageBoxOK("T3 Serve Tray", "Could not open the pairing link. It is still on the clipboard.")
		return err
	}
	return nil
}

func mintPairLink() (string, string, error) {
	pairArgs := cmdquote.WindowsCmdQuoted(paths.T3Cmd(), `pair --label "This PC"`)
	out, code, err := winutil.RunHiddenCapture(pairArgs, paths.UserProfile())
	if err != nil {
		return "", "", fmt.Errorf("could not run t3 pair")
	}
	if code != 0 {
		return "", "", fmt.Errorf("t3 pair failed (exit %d). %s", code, pairErrorDetail(string(out)))
	}
	pairURL, err := pairlink.ParsePairingURLLine(string(out))
	if err != nil {
		return "", "", fmt.Errorf("t3 pair did not print a pairing link")
	}
	appLink, err := pairlink.BuildAppLink(pairURL)
	if err != nil {
		return "", "", fmt.Errorf("could not build the desktop pairing link")
	}
	return pairURL, appLink, nil
}

func pairErrorDetail(out string) string {
	detail := ""
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.EqualFold(line, "ERROR") {
			continue
		}
		if strings.Contains(strings.ToLower(line), "token=") {
			continue
		}
		detail = line
	}
	if detail == "" {
		return "No pairing link was printed."
	}
	if len(detail) > 300 {
		detail = detail[:300]
	}
	return detail
}

func loopbackReady() bool {
	rt, err := ownership.ReadRuntime(paths.ServerRuntimeJSON())
	if err != nil || rt.PID == nil || !winutil.ProcessAlive(*rt.PID) {
		return false
	}
	return rt.Host == "127.0.0.1" || rt.Host == "localhost"
}

func desktopServerBeforeClose(sup *supervisor.Supervisor) (int, error) {
	rt, err := ownership.ReadRuntime(paths.ServerRuntimeJSON())
	if err != nil {
		return 0, nil
	}
	if rt.PID == nil {
		return 0, nil
	}
	pid := *rt.PID
	if !winutil.ProcessAlive(pid) {
		return 0, nil
	}
	if sup.IsOwnedPID(pid) {
		return 0, nil
	}
	ok, err := isDesktopServeProcess(pid)
	if err != nil || !ok {
		return 0, err
	}
	return pid, nil
}

func isDesktopServeProcess(pid int) (bool, error) {
	p, err := process.NewProcess(int32(pid))
	if err != nil {
		return false, err
	}
	name, err := p.Name()
	if err != nil {
		return false, err
	}
	if !strings.EqualFold(name, desktopExe) {
		return false, nil
	}
	cmd, err := p.Cmdline()
	if err != nil {
		return false, err
	}
	return strings.Contains(cmd, "bin.mjs"), nil
}

func closeDesktopUI() error {
	procs, err := process.Processes()
	if err != nil {
		return err
	}
	var uiPIDs []int32
	for _, p := range procs {
		name, err := p.Name()
		if err != nil || !strings.EqualFold(name, desktopExe) {
			continue
		}
		cmd, err := p.Cmdline()
		if err != nil {
			continue
		}
		if strings.Contains(cmd, "bin.mjs") {
			continue
		}
		uiPIDs = append(uiPIDs, p.Pid)
	}
	for _, pid := range uiPIDs {
		_ = winutil.PostCloseToProcessWindows(pid)
	}
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		alive := false
		for _, pid := range uiPIDs {
			if winutil.ProcessAlive(int(pid)) {
				alive = true
				break
			}
		}
		if !alive {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("T3 Code did not close within 20 seconds")
}

func waitForPIDExit(pid int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !winutil.ProcessAlive(pid) {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("timeout")
}

func OpenT3Code() error {
	exe := paths.T3CodeFallbackExe()
	if p, err := t3cmdparse.DesktopExeFromCmd(paths.T3Cmd()); err == nil && p != "" {
		exe = p
	}
	return winutil.OpenDocument(exe)
}
