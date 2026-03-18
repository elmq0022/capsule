package rootfs

import (
	"errors"
	"fmt"
	"os"
	"path"
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

const defaultPathEnv = "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"

func DefaultPathEnv() string {
	return defaultPathEnv
}

func NewRootFS(repo, tag string) (*RootFS, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("couldn't determine home directory: %w", err)
	}

	path := filepath.Join(home, ".local", "share", "capsule", "rootfs", repo, tag)
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("stat rootfs %q: %w", path, err)
	}

	return &RootFS{
		path:        path,
		mountTarget: path,
	}, nil
}

func (rfs *RootFS) ResolveCommand(command string, pathEnv string) (string, error) {
	return resolveCommand(rfs.path, command, pathEnv)
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

	if err := ensureMountPoint("/proc", 0o555); err != nil {
		return fmt.Errorf("couldn't prepare /proc mountpoint: %w", err)
	}
	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		return fmt.Errorf("couldn't mount /proc: %w", err)
	}
	rfs.procMounted = true
	return nil
}

func ensureMountPoint(target string, perm os.FileMode) error {
	info, err := os.Stat(target)
	if err == nil {
		if !info.IsDir() {
			return fmt.Errorf("%s exists but is not a directory", target)
		}
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.MkdirAll(target, perm)
}

func resolveCommand(root, command, pathEnv string) (string, error) {
	if strings.TrimSpace(command) == "" {
		return "", fmt.Errorf("command must not be empty")
	}

	if strings.Contains(command, "/") {
		candidate := cleanContainerPath(command)
		if err := ensureExecutable(root, candidate); err != nil {
			return "", fmt.Errorf("command %q not found in image: %w", candidate, err)
		}
		return candidate, nil
	}

	searchPath := pathEnv
	if strings.TrimSpace(searchPath) == "" {
		searchPath = defaultPathEnv
	}

	for _, dir := range strings.Split(searchPath, ":") {
		if strings.TrimSpace(dir) == "" {
			continue
		}
		candidate := cleanContainerPath(path.Join(dir, command))
		if err := ensureExecutable(root, candidate); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("command %q not found in PATH %q", command, searchPath)
}

func ensureExecutable(root, candidate string) error {
	resolved, err := resolveInRoot(root, candidate)
	if err != nil {
		return err
	}

	info, err := os.Stat(resolved)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("%q is a directory", candidate)
	}
	if info.Mode()&0o111 == 0 {
		return fmt.Errorf("%q is not executable", candidate)
	}
	return nil
}

func resolveInRoot(root, candidate string) (string, error) {
	parts := strings.Split(strings.TrimPrefix(cleanContainerPath(candidate), "/"), "/")
	if len(parts) == 1 && parts[0] == "" {
		return root, nil
	}

	resolved := make([]string, 0, len(parts))
	symlinks := 0
	for i := 0; i < len(parts); i++ {
		part := parts[i]
		if part == "" || part == "." {
			continue
		}
		if part == ".." {
			if len(resolved) > 0 {
				resolved = resolved[:len(resolved)-1]
			}
			continue
		}

		current := filepath.Join(append([]string{root}, resolved...)...)
		next := filepath.Join(current, part)
		info, err := os.Lstat(next)
		if err != nil {
			return "", err
		}

		if info.Mode()&os.ModeSymlink != 0 {
			symlinks++
			if symlinks > 40 {
				return "", fmt.Errorf("too many symlinks while resolving %q", candidate)
			}

			target, err := os.Readlink(next)
			if err != nil {
				return "", err
			}

			base := "/" + path.Join(resolved...)
			targetPath := target
			if !path.IsAbs(targetPath) {
				targetPath = path.Join(base, targetPath)
			}

			remaining := strings.Join(parts[i+1:], "/")
			merged := cleanContainerPath(targetPath)
			if remaining != "" {
				merged = cleanContainerPath(path.Join(merged, remaining))
			}

			parts = strings.Split(strings.TrimPrefix(merged, "/"), "/")
			resolved = resolved[:0]
			i = -1
			continue
		}

		if i < len(parts)-1 && !info.IsDir() {
			containerDir := "/" + path.Join(append(resolved, part)...)
			return "", fmt.Errorf("%q is not a directory", containerDir)
		}

		resolved = append(resolved, part)
	}

	return filepath.Join(append([]string{root}, resolved...)...), nil
}

func cleanContainerPath(p string) string {
	return path.Clean("/" + strings.TrimSpace(p))
}
