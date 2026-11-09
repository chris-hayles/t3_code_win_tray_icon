# T3 Serve Tray

Per-user Windows tray app that keeps a loopback `t3 serve` process running at `127.0.0.1` using your installed T3 CLI (`%USERPROFILE%\.t3\bin\t3.cmd`). It does **not** pass `--lan`. Quitting the tray stops only the server tree it started (via a Windows Job Object); it does not kill unrelated T3 servers.

## What it does

- On startup, runs `t3.cmd serve --host 127.0.0.1` under hidden `cmd.exe`, with automatic restart and exponential backoff.
- Tray menu: status, **Open T3 Code**, **Copy pairing link**, **Use this server in T3 Code…** (shown only while the desktop local environment is on; this is the only action that edits desktop settings), restart (when this tray owns the server), open logs, **Run at startup**, and **Exit**.
- Logs live in `%LOCALAPPDATA%\T3ServeTray\` (`t3-serve.log` rolls at 5 MiB; supervisor log is separate). Pairing tokens/URLs are not written to logs.

### Copy pairing link

Copies a one-time `http://127.0.0.1/pair#token=…` link for the running tray server. In T3 Code, open **Settings → Environments → Add environment** and paste it. The dialog can also open that link in T3 Code. The link expires after a few minutes; run the command again for a new one.

### Use this server in T3 Code…

After you confirm the dialog:

- T3 Code will close and reopen.
- Local environment is turned off in `desktop-settings.json` so the desktop stops running its own server.
- A pairing prompt opens for the `127.0.0.1` server.
- **Folder drag-and-drop into the composer is unavailable** while using the tray-managed loopback server.

The NSIS installer does **not** change T3 Code settings. The tray icon is the T3 Code icon, stored at `internal/assets/tray.ico`.

## Install

Build `dist\T3ServeTray-setup.exe`, then run it. The setup is per-user and does not ask for administrator rights. It installs to `%LOCALAPPDATA%\Programs\T3ServeTray`, adds a Startup shortcut, and leaves `%LOCALAPPDATA%\T3ServeTray` logs and `~\.t3` in place on uninstall.

## Build

```powershell
.\build.ps1
```

`build.ps1` runs `go test ./...`, generates `cmd\t3tray\rsrc.syso` from the icon (that file is not committed), builds `dist\t3tray.exe` with `-H windowsgui`, and runs `makensis` on `installer\T3ServeTray.nsi` when `makensis` is on `PATH`.

Manual steps:

```powershell
go test ./...
go build -ldflags "-H windowsgui" -o dist\t3tray.exe ./cmd/t3tray
makensis installer\T3ServeTray.nsi
```

## Release

Pushes to `main` and pull requests build the installer and upload it as an Actions artifact. Pushing a tag such as `v1.0.0` also publishes `T3ServeTray-setup.exe` on that GitHub release.

## Tests

```powershell
go test ./...
```

Unit tests use temporary directories only.

## Requirements

- Windows 10+
- Go 1.22+
- T3 CLI installed by T3 Code at `%USERPROFILE%\.t3\bin\t3.cmd`
- Optional: NSIS 3 for the installer
