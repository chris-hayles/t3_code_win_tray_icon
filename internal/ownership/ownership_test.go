package ownership

import (
	"os"
	"path/filepath"
	"testing"
)

func writeRuntime(t *testing.T, dir string, pid int) string {
	t.Helper()
	p := filepath.Join(dir, "server-runtime.json")
	body := `{"pid":` + itoa(pid) + `,"host":"127.0.0.1","port":3773,"origin":"http://127.0.0.1:3773"}`
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

func TestClassify_Absent(t *testing.T) {
	dir := t.TempDir()
	p := writeRuntime(t, dir, 1234)
	st, _, _ := ClassifyFile(p, func(pid int) bool { return false }, func(pid int) bool { return false })
	if st != Absent {
		t.Fatal(st)
	}
}

func TestClassify_Owned(t *testing.T) {
	dir := t.TempDir()
	p := writeRuntime(t, dir, 42)
	st, _, _ := ClassifyFile(p, func(pid int) bool { return pid == 42 }, func(pid int) bool { return true })
	if st != Owned {
		t.Fatal(st)
	}
}

func TestClassify_Foreign(t *testing.T) {
	dir := t.TempDir()
	p := writeRuntime(t, dir, 99)
	st, _, _ := ClassifyFile(p, func(pid int) bool { return true }, func(pid int) bool { return false })
	if st != Foreign {
		t.Fatal(st)
	}
}

func TestClassify_MissingPID(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "server-runtime.json")
	os.WriteFile(p, []byte(`{"host":"127.0.0.1"}`), 0o644)
	st, _, _ := ClassifyFile(p, func(int) bool { return true }, func(int) bool { return true })
	if st != Absent {
		t.Fatal(st)
	}
}
