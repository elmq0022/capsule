package cgroups

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type CGroup struct {
	cgroupDir string
}

func NewCGroup() (*CGroup, error) {
	cgroupDir, err := createCgroupDir()
	if err != nil {
		return nil, err
	}
	return &CGroup{cgroupDir: cgroupDir}, nil
}

func (c *CGroup) Close() error {
	return os.RemoveAll(c.cgroupDir)
}

func createCgroupDir() (string, error) {
	// assumes cgroup v2

	f, err := os.Open("/proc/self/cgroup")
	if err != nil {
		return "", fmt.Errorf("couldn't open /proc/self/cgroup: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return "", fmt.Errorf("scan /proc/self/cgroup: %w", err)
		}
		return "", fmt.Errorf("/proc/self/cgroup was empty")
	}

	line := scanner.Text()

	if !strings.HasPrefix(line, "0::/") {
		return "", fmt.Errorf("expected cgroup v2 but got %q", line)
	}
	path := strings.TrimPrefix(line, "0::")
	parent := filepath.Dir(filepath.Join("/sys/fs/cgroup/", path))
	cgroupDir := filepath.Join(parent, fmt.Sprintf("capsule-%d", os.Getpid()))

	if err := os.Mkdir(cgroupDir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir %q: %w", cgroupDir, err)
	}

	return cgroupDir, nil
}

func (c *CGroup) writeCgroupValue(file, value string) error {
	if err := os.WriteFile(filepath.Join(c.cgroupDir, file), []byte(value), 0o644); err != nil {
		return fmt.Errorf("could not write value %q to file %q: %w", value, file, err)
	}
	return nil
}

func (c *CGroup) SetPidsMax(value int) error {
	return c.writeCgroupValue("pids.max", strconv.Itoa(value))
}

func (c *CGroup) SetPidsMaxUnlimited() error {
	return c.writeCgroupValue("pids.max", "max")
}

func (c *CGroup) SetMemoryMax(value int) error {
	return c.writeCgroupValue("memory.max", strconv.Itoa(value))
}

func (c *CGroup) SetMemoryMaxUnlimited() error {
	return c.writeCgroupValue("memory.max", "max")
}

func (c *CGroup) AttachPID(value int) error {
	return c.writeCgroupValue("cgroup.procs", strconv.Itoa(value))
}
