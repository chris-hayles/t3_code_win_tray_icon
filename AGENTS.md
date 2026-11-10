# AGENTS.md

Windows-only Go tray app that supervises a loopback `t3 serve`. No CGO. The installer is NSIS and per-user.

## Do not

- Run the tray, `t3 serve`, or `t3 pair` against the user's live `~\.t3` while testing.
- Close or kill T3 Code processes.
- Write pairing tokens or pairing URLs to logs, commits, or test output.
- Pass `--lan`. The server is `t3 serve --host 127.0.0.1`.

## Easy to get wrong

- `WindowsCmdQuoted` wraps the command in an extra quote pair because `cmd.exe /c` strips one leading and one trailing quote. `pair --label "This PC"` must stay one argument. A broken quote makes `t3 pair` exit with `Unexpected positional argument`.
- The job object must use `windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION` with `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE`. A smaller struct does not set the flag. Do not allow breakaway. The tray process itself stays out of the job.
- Handles given to `CreateProcess` as stdio must be inheritable. Stdin is `\\.\NUL`. A capture pipe's read end must not be inherited.
- `ShellExecute` of `t3code://pair?...` gets an empty working directory. `filepath.Dir` on that URL is not a directory.
- Missing `desktop-settings.json`, or a missing `localEnvironmentEnabled` key, means the desktop local environment is on. Only JSON `false` hides **Use this server in T3 Code…**.
- Exit stops only the server this tray started. A live pid in `server-runtime.json` that this tray does not own is left alone.
- Do not restart `t3 serve` while a T3 Code update apply is in progress. Detect updater/installer processes under `%LOCALAPPDATA%\t3code-updater` (and the legacy `t3-code-desktop-updater` cache), a recent/changed `~\.t3\runtime\desktop-update-restart` mtime, a first sighting of desktop `--updated` (Electron keeps that flag for the process lifetime—do not renew the pause on later polls), or a `t3.cmd` target exe that does not exist. Pending installer files alone are not enough; leftovers persist after successful updates.
- The desktop link is `t3code://pair?pairingUrl=` plus the query-encoded `http://127.0.0.1:<port>/pair#token=...` URL.

## Icon

`internal/assets/tray.ico` is the T3 Code icon and is the tray, exe, and installer icon. Do not replace it with a generated placeholder. Refresh it with `tools/extracticon` from an installed `T3 Code (Beta).exe`.

## Build

`.\build.ps1` runs `go test ./...`, writes gitignored `cmd/t3tray/rsrc.syso`, builds `dist\t3tray.exe` with `-H windowsgui`, and runs `makensis` when it is on `PATH`. `.github/workflows/installer.yml` runs that script on Windows. A `v*` tag publishes the setup exe; other runs only upload the artifact.

Tests use temporary directories only.
