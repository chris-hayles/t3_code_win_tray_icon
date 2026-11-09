package cmdquote

import (
	"strings"
	"testing"
)

func TestWindowsCmdQuoted_SpacesInPath(t *testing.T) {
	got := WindowsCmdQuoted(`C:\Users\Me\My Docs\.t3\bin\t3.cmd`, "serve --host 127.0.0.1")
	if !strings.Contains(got, `"C:\Users\Me\My Docs\.t3\bin\t3.cmd"`) {
		t.Fatalf("batch path not quoted: %q", got)
	}
	if !strings.Contains(got, "serve --host 127.0.0.1") {
		t.Fatalf("args missing: %q", got)
	}
	if !strings.HasPrefix(got, `/d /c `) {
		t.Fatalf("missing /d /c prefix: %q", got)
	}
}

func TestWindowsCmdQuoted_FixedServeArgs(t *testing.T) {
	batch := `C:\Users\test\.t3\bin\t3.cmd`
	got := WindowsCmdQuoted(batch, "serve --host 127.0.0.1")
	want := `/d /c ""` + batch + `" serve --host 127.0.0.1"`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestWindowsCmdQuoted_LabelWithSpace(t *testing.T) {
	got := WindowsCmdQuoted(`C:\Users\test\.t3\bin\t3.cmd`, `pair --label "This PC"`)
	want := `/d /c ""C:\Users\test\.t3\bin\t3.cmd" pair --label "This PC""`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
