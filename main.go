package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

func fatalf(code int, format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(code)
}

func main() {
	if len(os.Args) < 3 {
		fatalf(1, "expected at least 3 args but got %q", os.Args)
	}
	switch cmd := os.Args[1]; cmd {
	case "run":
		run()
	case "child":
		child()
	default:
		fatalf(1, "unknown command %q", cmd)
	}
}

func run() {
	if len(os.Args) < 3 {
		fatalf(1, "did not provide a program to run")
	}

	// this reruns the **same** binary. So we run this this
	// main.go file but with the name child instead of run
	// the first pass sets up the new namespace and settings
	// the second pass (child) will use that namespace and set
	// the new hostname
	cmd := exec.Command("/proc/self/exe", append([]string{"child"}, os.Args[2:]...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.SysProcAttr = &syscall.SysProcAttr{
		// create a new namespace and a new user here
		Cloneflags: syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWUSER |
			syscall.CLONE_NEWNS |
			syscall.CLONE_NEWPID,
		UidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      os.Geteuid(),
				Size:        1,
			},
		},
		GidMappingsEnableSetgroups: false,
		GidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      os.Getegid(),
				Size:        1,
			},
		},
	}

	if err := cmd.Run(); err != nil {
		fatalf(1, "running command %s returned error: %q", os.Args[2], err)
	}
}

func child() {
	if len(os.Args) < 3 {
		fatalf(1, "did not provide a program to run")
	}
	cmd := exec.Command(os.Args[2], os.Args[3:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := syscall.Sethostname([]byte("capsule")); err != nil {
		fatalf(1, "couldn't set hostname: %v", err)
	}

	if err := syscall.Mount("", "/", "", syscall.MS_REC|syscall.MS_PRIVATE, ""); err != nil {
		fatalf(1, "couldn't make mount namespace private: %v", err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		fatalf(1, "couldn't determine home directory: %v", err)
	}

	rootfs := filepath.Join(home, ".local", "share", "capsule", "rootfs", "busybox")
	if err := syscall.Mount(rootfs, rootfs, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		fatalf(1, "couldn't bind-mount rootfs %q: %v", rootfs, err)
	}

	if err := syscall.Chdir(rootfs); err != nil {
		fatalf(1, "couldn't chdir to rootfs %q: %v", rootfs, err)
	}
	if err := syscall.Chroot("."); err != nil {
		fatalf(1, "couldn't chroot into %q: %v", rootfs, err)
	}
	if err := syscall.Chdir("/"); err != nil {
		fatalf(1, "couldn't chdir to new root: %v", err)
	}

	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		fatalf(1, "couldn't mount /proc: %v", err)
	}
	defer syscall.Unmount("/proc", 0)

	fmt.Printf("%q\n", os.Args)
	fmt.Printf("pid: %d, ppid: %d\n", os.Getpid(), os.Getppid())

	if err := cmd.Run(); err != nil {
		fatalf(1, "running command %s returned error: %q", os.Args[2], err)
	}
}
