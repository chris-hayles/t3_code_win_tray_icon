package logroll

import (
	"os"
	"path/filepath"
)

const DefaultMaxSize = 5 * 1024 * 1024

// MaybeRotate renames path to path+".1" when size exceeds maxBytes.
func MaybeRotate(path string, maxBytes int64) error {
	if maxBytes <= 0 {
		maxBytes = DefaultMaxSize
	}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if info.Size() < maxBytes {
		return nil
	}
	backup := path + ".1"
	_ = os.Remove(backup)
	return os.Rename(path, backup)
}

// AppendDir ensures the log directory exists.
func EnsureDir(dir string) error {
	return os.MkdirAll(dir, 0o755)
}

// OpenAppend opens path for append, rotating if needed.
func OpenAppend(path string) (*os.File, error) {
	if err := EnsureDir(filepath.Dir(path)); err != nil {
		return nil, err
	}
	if err := MaybeRotate(path, DefaultMaxSize); err != nil {
		return nil, err
	}
	return os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
}
