package rootfs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

type RootFS struct {
	path        string
	mountTarget string
	procMounted bool
	rootMounted bool
}

func NewRootFS(image string) (*RootFS, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("couldn't determine home directory: %w", err)
	}

	path := filepath.Join(home, ".local", "share", "capsule", "rootfs", image)
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("stat rootfs %q: %w", path, err)
	}

	return &RootFS{
		path:        path,
		mountTarget: path,
	}, nil
}

func (rfs *RootFS) Close() error {
	var errs []string

	if rfs.procMounted {
		if err := syscall.Unmount("/proc", 0); err != nil {
			errs = append(errs, fmt.Sprintf("unmount /proc: %v", err))
		} else {
			rfs.procMounted = false
		}
	}

	if rfs.rootMounted {
		if err := syscall.Unmount(rfs.mountTarget, syscall.MNT_DETACH); err != nil {
			errs = append(errs, fmt.Sprintf("detach rootfs %q at %q: %v", rfs.path, rfs.mountTarget, err))
		} else {
			rfs.rootMounted = false
		}
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}

	return nil
}

func (rfs *RootFS) MountRootFS() (err error) {
	defer func() {
		if err != nil {
			_ = rfs.Close()
		}
	}()

	if err := syscall.Mount("", "/", "", syscall.MS_REC|syscall.MS_PRIVATE, ""); err != nil {
		return fmt.Errorf("couldn't make mount namespace private: %w", err)
	}

	if err := syscall.Mount(rfs.path, rfs.path, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("couldn't bind-mount rootfs %q: %w", rfs.path, err)
	}
	rfs.rootMounted = true

	if err := syscall.Chdir(rfs.path); err != nil {
		return fmt.Errorf("couldn't chdir to rootfs %q: %w", rfs.path, err)
	}
	if err := syscall.Chroot("."); err != nil {
		return fmt.Errorf("couldn't chroot into %q: %w", rfs.path, err)
	}
	rfs.mountTarget = "/"
	if err := syscall.Chdir("/"); err != nil {
		return fmt.Errorf("couldn't chdir to new root: %w", err)
	}

	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		return fmt.Errorf("couldn't mount /proc: %w", err)
	}
	rfs.procMounted = true
	return nil
}
