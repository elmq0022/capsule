package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/elmq0022/capsule/cgroups"
	"github.com/elmq0022/capsule/namespaces"
	"github.com/elmq0022/capsule/pull"
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
		pullImage()
	case "run":
		run()
	case "child":
		child()
	default:
		fatalf(1, "unknown command %q", cmd)
	}
}

func parseImageRef(image, tag string) (string, string) {
	repo := strings.TrimSpace(image)
	refTag := strings.TrimSpace(tag)
	if repo == "" {
		fatalf(1, "image must not be empty")
	}
	if refTag == "" {
		fatalf(1, "tag must not be empty")
	}
	if !strings.Contains(repo, "/") {
		repo = "library/" + repo
	}
	return repo, refTag
}

func pullImage() {
	if len(os.Args) != 4 {
		fatalf(1, "usage: %s pull <image> <tag>", os.Args[0])
	}

	repo, tag := parseImageRef(os.Args[2], os.Args[3])
	client := pull.NewClient(repo)
	if err := client.Pull(tag); err != nil {
		fatalf(1, "pull %s:%s: %v", repo, tag, err)
	}
}

func run() {
	if len(os.Args) < 5 {
		fatalf(1, "usage: %s run <image> <tag> <command> [args...]", os.Args[0])
	}

	repo, tag := parseImageRef(os.Args[2], os.Args[3])

	// this reruns the **same** binary. So we run this this
	// main.go file but with the name child instead of run
	// the first pass sets up the new namespace and settings
	// the second pass (child) will use that namespace and set
	// the new hostname
	cmd := exec.Command("/proc/self/exe", append([]string{"child", repo, tag}, os.Args[4:]...)...)
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
		fatalf(1, "running command %s returned error: %q", os.Args[4], err)
	}

	if err := cgroup.AttachPID(cmd.Process.Pid); err != nil {
		fatalf(1, "couldn't attach child pid %d to cgroup: %w", cmd.Process.Pid, err)
	}

	if err := cmd.Wait(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				os.Exit(status.ExitStatus())
			}
		}
		fatalf(1, "waiting for child process failed: %v", err)
	}
}

func child() {
	if len(os.Args) < 5 {
		fatalf(1, "usage: %s child <image> <tag> <command> [args...]", os.Args[0])
	}
	repo, tag := parseImageRef(os.Args[2], os.Args[3])

	if err := syscall.Sethostname([]byte("capsule")); err != nil {
		fatalf(1, "couldn't set hostname: %v", err)
	}

	rfs, err := rootfs.NewRootFS(repo, tag)
	if err != nil {
		fatalf(1, "couldn't prepare rootfs: %v", err)
	}
	pathEnv := rootfs.DefaultPathEnv()
	if err := os.Setenv("PATH", pathEnv); err != nil {
		fatalf(1, "set PATH: %v", err)
	}
	commandPath, err := rfs.ResolveCommand(os.Args[4], pathEnv)
	if err != nil {
		fatalf(1, "resolve command %q in %s:%s: %v", os.Args[4], repo, tag, err)
	}
	if err := rfs.MountRootFS(); err != nil {
		fatalf(1, "%v", err)
	}
	defer func() {
		if err := rfs.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "cleanup rootfs: %v\n", err)
		}
	}()

	cmd := exec.Command(commandPath, os.Args[5:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fatalf(1, "running command %s returned error: %q", os.Args[4], err)
	}
}
