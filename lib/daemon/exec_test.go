package main

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"testing"
	"time"
)

func spawnCmd(errChan chan error, argv0 string, args ...string) () {
	cmd := exec.Command(argv0, args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Env = append(os.Environ(), "PORTABLE_LOGGING=debug")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Pdeathsig: syscall.SIGTERM,
	}
	err := cmd.Run()
	if err != nil {
		errChan <- err
	}
	return
}

func TestPools(t *testing.T) {
	var wg sync.WaitGroup
	var errChan = make(chan error, 16)
	fmt.Println("Testing pools sandbox")
	wg.Go(func() {
		spawnCmd(errChan, "portable-pools", "testbox")
	})
	time.Sleep(1 * time.Second)
	wg.Go(func() {
		spawnCmd(errChan, "portable-pools", "testbox", "--quit")
	})
	go func () {
		wg.Wait()
		close(errChan)
	} ()
	for err := range errChan {
		if err != nil {
			t.Fatal("Failed to execute pools:", err)
		}
	}
}