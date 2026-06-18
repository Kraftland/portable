package main

import (
	"github.com/seccomp/libseccomp-golang"
)

func createSeccompFilter() error {
	filter, err := seccomp.NewFilter(seccomp.ActAllow)
	if err != nil {
		return err
	}
	defer filter.Release()
	err = filter.Load()
	if err != nil {
		return err
	}
	return nil
}