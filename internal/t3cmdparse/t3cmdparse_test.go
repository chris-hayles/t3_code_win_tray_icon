package t3cmdparse

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDesktopExeFromCmd(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "t3.cmd")
	content := `@echo off
set T3_DESKTOP="C:\Apps\T3 Code (Beta).exe"
"%T3_DESKTOP%" %*
`
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := DesktopExeFromCmd(p)
	if err != nil {
		t.Fatal(err)
	}
	if got != `C:\Apps\T3 Code (Beta).exe` {
		t.Fatalf("got %q", got)
	}
}
