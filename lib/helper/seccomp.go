package main

import (
	"syscall"

	"github.com/seccomp/libseccomp-golang"
)

func createSeccompFilter() error {
	filter, err := seccomp.NewFilter(seccomp.ActAllow)
	if err != nil {
		return err
	}
	defer filter.Release()

	err = filter.AddRule(
		syscall.SYS_EXECVE,
		seccomp.ActNotify,
	)
	if err != nil {
		return err
	}

	err = filter.Load()
	if err != nil {
		return err
	}
	return nil
}