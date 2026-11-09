$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot

go mod tidy
go test ./...

New-Item -ItemType Directory -Force -Path dist | Out-Null
go run github.com/akavel/rsrc@v0.10.2 -arch amd64 -ico internal/assets/tray.ico -o cmd/t3tray/rsrc.syso
go build -ldflags "-H windowsgui" -o dist\t3tray.exe ./cmd/t3tray

$makensis = Get-Command makensis -ErrorAction SilentlyContinue
if ($null -ne $makensis) {
  & makensis installer\T3ServeTray.nsi
} else {
  Write-Host "makensis not on PATH; skipped T3ServeTray-setup.exe (installer script is ready)."
}
