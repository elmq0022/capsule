package namespaces

import (
	"os"
	"os/exec"
	"syscall"
)

func SetNamespaces(cmd *exec.Cmd) {
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
}
