package cihook

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStubHookWritesPipelineFile(t *testing.T) {
	hook := NewStubHook()
	dir := filepath.Join(t.TempDir(), "example-service")

	ref, err := hook.TriggerPipeline(dir, "go")
	if err != nil {
		t.Fatalf("TriggerPipeline() error = %v", err)
	}

	if _, err := os.Stat(ref); err != nil {
		t.Fatalf("expected pipeline file at %q, stat error = %v", ref, err)
	}

	data, err := os.ReadFile(ref)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", ref, err)
	}
	if !strings.Contains(string(data), "language: go") {
		t.Errorf("pipeline content = %q, want it to contain %q", data, "language: go")
	}
}
