//go:build windows

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/getlantern/systray"
	"t3tray/internal/assets"
	"t3tray/internal/attach"
	"t3tray/internal/paths"
	"t3tray/internal/settings"
	"t3tray/internal/shortcut"
	"t3tray/internal/supervisor"
	"t3tray/internal/winutil"
)

func main() {
	mtx, ok := winutil.TrySingleInstance("T3ServeTray")
	if !ok {
		os.Exit(0)
	}
	defer winutil.CloseProcessHandle(uintptr(mtx))

	sup, err := supervisor.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "supervisor: %v\n", err)
		os.Exit(1)
	}
	defer sup.CloseLogs()
	sup.CheckStartupOwnership()
	sup.StartLoop()

	systray.Run(func() { onReady(sup) }, func() {
		sup.StopOwned()
	})
}

func onReady(sup *supervisor.Supervisor) {
	systray.SetIcon(assets.TrayIcon)
	systray.SetTitle("T3 Serve Tray")
	systray.SetTooltip("T3 Serve Tray")

	mStatus := systray.AddMenuItem("Starting…", "Server status")
	mStatus.Disable()
	mOpen := systray.AddMenuItem("Open T3 Code", "")
	mPair := systray.AddMenuItem("Copy pairing link", "Copy a link for the desktop app")
	mAttach := systray.AddMenuItem("Use this server in T3 Code…", "")
	mRestart := systray.AddMenuItem("Restart server", "")
	mLogs := systray.AddMenuItem("Open logs", "")
	mStartup := systray.AddMenuItem("Run at startup", "")
	if shortcut.Exists(paths.StartupShortcut()) {
		mStartup.Check()
	}
	systray.AddSeparator()
	mExit := systray.AddMenuItem("Exit", "")

	go func() {
		attachShown := true
		for {
			st, label := sup.CurrentStatus()
			mStatus.SetTitle(label)
			if st == supervisor.StatusRunning {
				mRestart.Enable()
			} else {
				mRestart.Disable()
			}
			needAttach := settings.LocalEnvironmentEnabled(paths.DesktopSettingsJSON())
			if needAttach != attachShown {
				if needAttach {
					mAttach.Show()
				} else {
					mAttach.Hide()
				}
				attachShown = needAttach
			}
			time.Sleep(2 * time.Second)
		}
	}()

	go menuLoop(mOpen, func() { _ = attach.OpenT3Code() })
	go menuLoop(mPair, func() { _ = attach.CopyPairingLink(sup) })
	go menuLoop(mAttach, func() { _ = attach.RunAttachFlow(sup) })
	go menuLoop(mRestart, func() { sup.RestartOwned() })
	go menuLoop(mLogs, func() { _ = winutil.OpenExplorerDir(paths.TrayDataDir()) })
	go menuLoop(mStartup, func() {
		if mStartup.Checked() {
			_ = shortcut.ToggleStartupShortcut(false)
			mStartup.Uncheck()
		} else {
			if err := shortcut.ToggleStartupShortcut(true); err == nil {
				mStartup.Check()
			}
		}
	})
	go menuLoop(mExit, func() {
		sup.StopOwned()
		systray.Quit()
	})
}

func menuLoop(item *systray.MenuItem, fn func()) {
	for range item.ClickedCh {
		fn()
	}
}
