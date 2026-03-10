package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Printf("Expected 3 args but only got %q", os.Args)
		os.Exit(1)
	}
	switch cmd := os.Args[1]; cmd {
	case "run":
		run()
	case "child":
		child()
	default:
		fmt.Printf("arg was %s\n", cmd)
	}
}

func run() {
	if len(os.Args) < 3 {
		fmt.Println("did not provide a program to run")
		return
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
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWUSER,
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
		fmt.Printf(
			"running command %s returned error: %q\n",
			os.Args[2],
			err,
		)
		os.Exit(1)
	}
}

func child() {
	if len(os.Args) < 3 {
		fmt.Println("did not provide a program to run")
		return
	}
	cmd := exec.Command(os.Args[2], os.Args[3:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := syscall.Sethostname([]byte("capsule")); err != nil {
		fmt.Printf("couldn't set hostname: %v", err)
		os.Exit(1)
	}

	fmt.Printf("%q\n", os.Args)
	err := cmd.Run()

	if err != nil {
		fmt.Printf(
			"running command %s returned error: %q\n",
			os.Args[1],
			err,
		)
		os.Exit(1)
	}
}
