package rootfs

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

type RootFS struct {
	image string
}

func NewRootFS(image string) *RootFS {
	return &RootFS{image: image}
}

func (rfs *RootFS) Close() {
	syscall.Unmount("/proc", 0)
}

func (rfs *RootFS) MountRootFS() error {
	if err := syscall.Mount("", "/", "", syscall.MS_REC|syscall.MS_PRIVATE, ""); err != nil {
		return fmt.Errorf("couldn't make mount namespace private: %w", err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("couldn't determine home directory: %v", err)
	}

	rootfs := filepath.Join(home, ".local", "share", "capsule", "rootfs", rfs.image)
	if err := syscall.Mount(rootfs, rootfs, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("couldn't bind-mount rootfs %q: %v", rootfs, err)
	}

	if err := syscall.Chdir(rootfs); err != nil {
		return fmt.Errorf("couldn't chdir to rootfs %q: %v", rootfs, err)
	}
	if err := syscall.Chroot("."); err != nil {
		return fmt.Errorf("couldn't chroot into %q: %v", rootfs, err)
	}
	if err := syscall.Chdir("/"); err != nil {
		return fmt.Errorf("couldn't chdir to new root: %v", err)
	}

	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		return fmt.Errorf("couldn't mount /proc: %v", err)
	}
	return nil
}
