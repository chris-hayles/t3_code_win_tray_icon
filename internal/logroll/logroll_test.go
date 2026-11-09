package logroll

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMaybeRotate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "t3-serve.log")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := MaybeRotate(path, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("original should be gone")
	}
	if _, err := os.Stat(path + ".1"); err != nil {
		t.Fatal("backup missing")
	}
}

func TestMaybeRotate_UnderLimit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "log")
	os.WriteFile(path, []byte("small"), 0o644)
	if err := MaybeRotate(path, 1000); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path + ".1"); !os.IsNotExist(err) {
		t.Fatal("should not rotate")
	}
}
