package pull

import (
	"archive/tar"
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestClientIsAuthenticated(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		name   string
		token  authTokenResponse
		expect bool
	}{
		{
			name:   "missing token",
			token:  authTokenResponse{},
			expect: false,
		},
		{
			name: "missing issued at",
			token: authTokenResponse{
				Token:     "abc",
				ExpiresIn: 60,
			},
			expect: false,
		},
		{
			name: "non positive expiry",
			token: authTokenResponse{
				Token:    "abc",
				IssuedAt: now.Format(time.RFC3339),
			},
			expect: false,
		},
		{
			name: "invalid issued at",
			token: authTokenResponse{
				Token:     "abc",
				ExpiresIn: 60,
				IssuedAt:  "not-a-time",
			},
			expect: false,
		},
		{
			name: "expired token",
			token: authTokenResponse{
				Token:     "abc",
				ExpiresIn: 60,
				IssuedAt:  now.Add(-2 * time.Minute).Format(time.RFC3339),
			},
			expect: false,
		},
		{
			name: "valid token",
			token: authTokenResponse{
				Token:     "abc",
				ExpiresIn: 60,
				IssuedAt:  now.Add(-30 * time.Second).Format(time.RFC3339),
			},
			expect: true,
		},
		{
			name: "token inside refresh buffer",
			token: authTokenResponse{
				Token:     "abc",
				ExpiresIn: 60,
				IssuedAt:  now.Add(-(60*time.Second - authRefreshBuffer + time.Second)).Format(time.RFC3339),
			},
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{token: tt.token}

			if got := client.IsAuthenticated(); got != tt.expect {
				t.Fatalf("IsAuthenticated() = %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestResolveRootfsPath(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "tmp", "capsule-rootfs")

	got, err := resolveRootfsPath(root, "etc/passwd")
	if err != nil {
		t.Fatalf("resolveRootfsPath() unexpected error: %v", err)
	}

	want := filepath.Join(root, "etc", "passwd")
	if got != want {
		t.Fatalf("resolveRootfsPath() = %q, want %q", got, want)
	}
}

func TestExtractLayerEntry(t *testing.T) {
	rootfsDir := t.TempDir()

	dirHdr := &tar.Header{
		Name:     "etc",
		Typeflag: tar.TypeDir,
		Mode:     0o755,
		ModTime:  time.Now(),
	}
	if err := extractLayerEntry(rootfsDir, dirHdr, bytes.NewReader(nil)); err != nil {
		t.Fatalf("extract dir: %v", err)
	}

	fileHdr := &tar.Header{
		Name:     "etc/hostname",
		Typeflag: tar.TypeReg,
		Mode:     0o644,
		Size:     int64(len("capsule\n")),
		ModTime:  time.Now(),
	}
	if err := extractLayerEntry(rootfsDir, fileHdr, bytes.NewReader([]byte("capsule\n"))); err != nil {
		t.Fatalf("extract file: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(rootfsDir, "etc", "hostname"))
	if err != nil {
		t.Fatalf("read extracted file: %v", err)
	}
	if string(data) != "capsule\n" {
		t.Fatalf("extracted file content = %q, want %q", string(data), "capsule\n")
	}

	symlinkHdr := &tar.Header{
		Name:     "bin/sh",
		Typeflag: tar.TypeSymlink,
		Linkname: "/bin/busybox",
		Mode:     0o777,
		ModTime:  time.Now(),
	}
	if err := extractLayerEntry(rootfsDir, symlinkHdr, bytes.NewReader(nil)); err != nil {
		t.Fatalf("extract symlink: %v", err)
	}

	linkTarget, err := os.Readlink(filepath.Join(rootfsDir, "bin", "sh"))
	if err != nil {
		t.Fatalf("read extracted symlink: %v", err)
	}
	if linkTarget != "/bin/busybox" {
		t.Fatalf("symlink target = %q, want %q", linkTarget, "/bin/busybox")
	}

	hardLinkHdr := &tar.Header{
		Name:     "etc/hostname.copy",
		Typeflag: tar.TypeLink,
		Linkname: "etc/hostname",
		Mode:     0o644,
		ModTime:  time.Now(),
	}
	if err := extractLayerEntry(rootfsDir, hardLinkHdr, bytes.NewReader(nil)); err != nil {
		t.Fatalf("extract hardlink: %v", err)
	}

	copyData, err := os.ReadFile(filepath.Join(rootfsDir, "etc", "hostname.copy"))
	if err != nil {
		t.Fatalf("read extracted hardlink: %v", err)
	}
	if string(copyData) != "capsule\n" {
		t.Fatalf("hardlink content = %q, want %q", string(copyData), "capsule\n")
	}
}

func TestExtractLayerEntryOverwritesExistingFile(t *testing.T) {
	rootfsDir := t.TempDir()

	target := filepath.Join(rootfsDir, "etc", "hostname")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("create parent dir: %v", err)
	}
	if err := os.WriteFile(target, []byte("old\n"), 0o644); err != nil {
		t.Fatalf("seed existing file: %v", err)
	}

	hdr := &tar.Header{
		Name:     "etc/hostname",
		Typeflag: tar.TypeReg,
		Mode:     0o644,
		Size:     int64(len("new\n")),
		ModTime:  time.Now(),
	}

	if err := extractLayerEntry(rootfsDir, hdr, bytes.NewReader([]byte("new\n"))); err != nil {
		t.Fatalf("extractLayerEntry() unexpected error: %v", err)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read overwritten file: %v", err)
	}
	if string(data) != "new\n" {
		t.Fatalf("overwritten file content = %q, want %q", string(data), "new\n")
	}
}
