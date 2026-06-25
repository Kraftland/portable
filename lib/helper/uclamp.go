package main

import (
	"os"
)

func writeUclampVal() error {
	rawVal := os.Getenv("_portableUclampMax")
	if len(rawVal) == 0 {
		return nil
	}
	return os.WriteFile(
		"/sys/fs/cgroup/cpu.uclamp.max",
		[]byte(rawVal),
		0700,
	)
}