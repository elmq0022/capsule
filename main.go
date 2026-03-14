package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/elmq0022/capsule/cgroups"
	"github.com/elmq0022/capsule/namespaces"
	"github.com/elmq0022/capsule/rootfs"
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
	case "pull":
		// todo pull the image
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
	namespaces.SetNamespaces(cmd)

	cgroup, err := cgroups.NewCGroup()
	if err != nil {
		fatalf(1, "could not create cgroup: %v", err)
	}

	defer func() {
		if err := cgroup.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "cleanup cgroup: %v\n", err)
		}
	}()

	if err := cgroup.SetPidsMax(10); err != nil {
		fatalf(1, "%w", err)
	}

	if err := cgroup.SetMemoryMax(67108864); err != nil {
		fatalf(1, "%w", err)
	}

	if err := cmd.Start(); err != nil {
		fatalf(1, "running command %s returned error: %q", os.Args[2], err)
	}

	if err := cgroup.AttachPID(cmd.Process.Pid); err != nil {
		fatalf(1, "couldn't attach child pid %d to cgroup: %w", cmd.Process.Pid, err)
	}

	if err := cmd.Wait(); err != nil {
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

	rfs, err := rootfs.NewRootFS("busybox")
	if err != nil {
		fatalf(1, "couldn't prepare rootfs: %v", err)
	}
	if err := rfs.MountRootFS(); err != nil {
		fatalf(1, "%v", err)
	}
	defer func() {
		if err := rfs.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "cleanup rootfs: %v\n", err)
		}
	}()

	fmt.Printf("%q\n", os.Args)
	fmt.Printf("pid: %d, ppid: %d\n", os.Getpid(), os.Getppid())

	if err := cmd.Run(); err != nil {
		fatalf(1, "running command %s returned error: %q", os.Args[2], err)
	}
}
