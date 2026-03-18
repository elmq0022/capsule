package rootfs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveCommandFindsSymlinkedBinary(t *testing.T) {
	root := t.TempDir()

	if err := os.MkdirAll(filepath.Join(root, "bin"), 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "bin", "busybox"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write busybox: %v", err)
	}
	if err := os.Symlink("/bin/busybox", filepath.Join(root, "bin", "sh")); err != nil {
		t.Fatalf("symlink sh: %v", err)
	}

	got, err := resolveCommand(root, "/bin/sh", "")
	if err != nil {
		t.Fatalf("resolveCommand() unexpected error: %v", err)
	}
	if got != "/bin/sh" {
		t.Fatalf("resolveCommand() = %q, want %q", got, "/bin/sh")
	}
}

func TestResolveCommandSearchesPath(t *testing.T) {
	root := t.TempDir()

	if err := os.MkdirAll(filepath.Join(root, "usr", "bin"), 0o755); err != nil {
		t.Fatalf("mkdir usr/bin: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "usr", "bin", "echo"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write echo: %v", err)
	}

	got, err := resolveCommand(root, "echo", "/usr/bin:/bin")
	if err != nil {
		t.Fatalf("resolveCommand() unexpected error: %v", err)
	}
	if got != "/usr/bin/echo" {
		t.Fatalf("resolveCommand() = %q, want %q", got, "/usr/bin/echo")
	}
}

func TestResolveCommandReportsMissingBinary(t *testing.T) {
	root := t.TempDir()

	if err := os.MkdirAll(filepath.Join(root, "bin"), 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}

	_, err := resolveCommand(root, "/bin/bash", "")
	if err == nil {
		t.Fatal("resolveCommand() error = nil, want error")
	}
	if !strings.Contains(err.Error(), `command "/bin/bash" not found in image`) {
		t.Fatalf("resolveCommand() error = %q, want missing command message", err)
	}
}

func TestEnsureMountPointCreatesMissingDirectory(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "proc")

	if err := ensureMountPoint(target, 0o555); err != nil {
		t.Fatalf("ensureMountPoint() unexpected error: %v", err)
	}

	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("stat target: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("target is not a directory")
	}
}

func TestEnsureMountPointRejectsFile(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "proc")

	if err := os.WriteFile(target, []byte("capsule"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	err := ensureMountPoint(target, 0o555)
	if err == nil {
		t.Fatal("ensureMountPoint() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "is not a directory") {
		t.Fatalf("ensureMountPoint() error = %q, want not-a-directory error", err)
	}
}
