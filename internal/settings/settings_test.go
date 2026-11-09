package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestApplyDisableLocal_MissingFile(t *testing.T) {
	out, err := ApplyDisableLocal(nil)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if string(m["localEnvironmentEnabled"]) != "false" {
		t.Fatalf("got %s", m["localEnvironmentEnabled"])
	}
}

func TestApplyDisableLocal_PreservesKeys(t *testing.T) {
	in := []byte(`{"localEnvironmentEnabled":true,"mainWindowBounds":{"x":1}}`)
	out, err := ApplyDisableLocal(in)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if m["localEnvironmentEnabled"] != false {
		t.Fatalf("localEnvironmentEnabled=%v", m["localEnvironmentEnabled"])
	}
	bounds, ok := m["mainWindowBounds"].(map[string]interface{})
	if !ok || bounds["x"] != float64(1) {
		t.Fatalf("mainWindowBounds missing: %v", m)
	}
}

func TestApplyDisableLocal_AlreadyFalse(t *testing.T) {
	in := []byte(`{"localEnvironmentEnabled":false,"foo":"bar"}`)
	out, err := ApplyDisableLocal(in)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]interface{}
	json.Unmarshal(out, &m)
	if m["localEnvironmentEnabled"] != false || m["foo"] != "bar" {
		t.Fatalf("%v", m)
	}
}

func TestLocalEnvironmentEnabled(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "desktop-settings.json")
	if !LocalEnvironmentEnabled(path) {
		t.Fatal("missing file should be enabled")
	}
	if err := os.WriteFile(path, []byte(`{"mainWindowBounds":{"x":1}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if !LocalEnvironmentEnabled(path) {
		t.Fatal("missing key should be enabled")
	}
	if err := os.WriteFile(path, []byte(`{"localEnvironmentEnabled":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if !LocalEnvironmentEnabled(path) {
		t.Fatal("true should be enabled")
	}
	if err := os.WriteFile(path, []byte(`{"localEnvironmentEnabled":false,"mainWindowBounds":{"x":1}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if LocalEnvironmentEnabled(path) {
		t.Fatal("explicit false should be disabled")
	}
}

func TestMergeDisableLocalEnvironment_AtomicWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "desktop-settings.json")
	if err := os.WriteFile(path, []byte(`{"localEnvironmentEnabled":true,"k":2}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := MergeDisableLocalEnvironment(path); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	if m["localEnvironmentEnabled"] != false {
		t.Fatal(m)
	}
	if m["k"] != float64(2) {
		t.Fatal(m)
	}
}

func TestLocalEnvironmentEnabled_NullIsEnabled(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "desktop-settings.json")
	if err := os.WriteFile(path, []byte(`{"localEnvironmentEnabled":null,"k":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if !LocalEnvironmentEnabled(path) {
		t.Fatal("null should be enabled")
	}
}
