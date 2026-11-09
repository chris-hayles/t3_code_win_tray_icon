package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type DesktopSettings struct {
	LocalEnvironmentEnabled *bool `json:"localEnvironmentEnabled"`
	// Other keys are preserved via raw merge.
	raw map[string]json.RawMessage
}

// MergeDisableLocalEnvironment reads path, sets localEnvironmentEnabled to false, preserves other keys.
func MergeDisableLocalEnvironment(path string) error {
	data, err := os.ReadFile(path)
	var merged map[string]json.RawMessage
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			merged = map[string]json.RawMessage{}
		} else {
			return err
		}
	} else {
		if err := json.Unmarshal(data, &merged); err != nil {
			return fmt.Errorf("parse desktop-settings: %w", err)
		}
		if merged == nil {
			merged = map[string]json.RawMessage{}
		}
	}
	falseVal := json.RawMessage("false")
	merged["localEnvironmentEnabled"] = falseVal

	out, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "desktop-settings-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		_ = os.Remove(tmpName)
	}()
	if _, err := tmp.Write(out); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// ParseDesktopSettings is used in tests to verify merge behavior without touching real paths.
func ParseDesktopSettings(data []byte) (map[string]json.RawMessage, error) {
	var m map[string]json.RawMessage
	if len(data) == 0 {
		return map[string]json.RawMessage{"localEnvironmentEnabled": json.RawMessage("false")}, nil
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// LocalEnvironmentEnabled reports whether the desktop app will start its own server.
// T3 Code treats a missing file, a missing key, or a null value as enabled. Only an explicit false turns it off.
func LocalEnvironmentEnabled(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return true
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(data, &doc); err != nil || doc == nil {
		return true
	}
	raw, ok := doc["localEnvironmentEnabled"]
	if !ok {
		return true
	}
	// json.Unmarshal into a plain bool leaves it unchanged on null, so decode into a pointer.
	var enabled *bool
	if err := json.Unmarshal(raw, &enabled); err != nil || enabled == nil {
		return true
	}
	return *enabled
}

// ApplyDisableLocal sets the key and returns serialized JSON (compact).
func ApplyDisableLocal(data []byte) ([]byte, error) {
	var merged map[string]json.RawMessage
	if len(data) == 0 {
		merged = map[string]json.RawMessage{}
	} else if err := json.Unmarshal(data, &merged); err != nil {
		return nil, err
	}
	if merged == nil {
		merged = map[string]json.RawMessage{}
	}
	merged["localEnvironmentEnabled"] = json.RawMessage("false")
	return json.Marshal(merged)
}
