$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot

# ErrorActionPreference does not turn a non-zero native exit code into an error, so check each one.
function Invoke-Native {
  param(
    [Parameter(Mandatory = $true)][string]$Name,
    [Parameter(Mandatory = $true)][scriptblock]$Command
  )
  & $Command
  if ($LASTEXITCODE -ne 0) {
    Write-Error "$Name failed with exit code $LASTEXITCODE"
    exit $LASTEXITCODE
  }
}

Invoke-Native "go mod tidy" { go mod tidy }
Invoke-Native "go vet" { go vet ./... }
Invoke-Native "go test" { go test ./... }

New-Item -ItemType Directory -Force -Path dist | Out-Null
Invoke-Native "rsrc" { go run github.com/akavel/rsrc@v0.10.2 -arch amd64 -ico internal/assets/tray.ico -o cmd/t3tray/rsrc.syso }
Invoke-Native "go build" { go build -ldflags "-H windowsgui" -o dist\t3tray.exe ./cmd/t3tray }

$makensis = Get-Command makensis -ErrorAction SilentlyContinue
if ($null -ne $makensis) {
  Invoke-Native "makensis" { & makensis installer\T3ServeTray.nsi }
} else {
  Write-Host "makensis not on PATH; skipped T3ServeTray-setup.exe (installer script is ready)."
}
