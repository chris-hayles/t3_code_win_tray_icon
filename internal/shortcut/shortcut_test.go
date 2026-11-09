package shortcut

import (
	"path/filepath"
	"testing"
)

func TestStartupPath(t *testing.T) {
	base := filepath.Join(`C:\Users\x\AppData\Roaming`)
	got := StartupPath(base)
	want := filepath.Join(base, `Microsoft`, `Windows`, `Start Menu`, `Programs`, `Startup`, `T3 Serve Tray.lnk`)
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
