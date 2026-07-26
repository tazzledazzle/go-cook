package uat

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func moduleRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(file), "..", "..")
}

func minimalTorrentPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(moduleRoot(t), "testdata", "minimal.torrent")
}

func runGoTest(t *testing.T, pkg string, extraArgs ...string) {
	t.Helper()
	args := append([]string{"test", pkg, "-count=1"}, extraArgs...)
	cmd := exec.Command("go", args...)
	cmd.Dir = moduleRoot(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go test %s failed: %v\n%s", pkg, err, out)
	}
}

func runGoTestAllExceptUAT(t *testing.T, extraArgs ...string) {
	t.Helper()
	listCmd := exec.Command("go", "list", "./...")
	listCmd.Dir = moduleRoot(t)
	out, err := listCmd.Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" || strings.Contains(line, "/internal/uat") {
			continue
		}
		args := append([]string{"test", line, "-count=1"}, extraArgs...)
		cmd := exec.Command("go", args...)
		cmd.Dir = moduleRoot(t)
		combined, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("go test %s failed: %v\n%s", line, err, combined)
		}
	}
}

func fileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file %s: %v", path, err)
	}
}
