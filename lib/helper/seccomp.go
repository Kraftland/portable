package main

import (
	"bufio"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/seccomp/libseccomp-golang"
)

func readArgFromMemory(pid int, addr uint64) (string, error) {
	if addr == 0 {
		return "", errors.New(
			"Could not read argument: Null pointer passed",
		)
	}
	path := filepath.Join("/proc", strconv.Itoa(pid), "mem")
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	_, err = file.Seek(int64(addr), io.SeekStart)
	if err != nil {
		return "", err
	}
	reader := bufio.NewReader(file)
	bytes, err := reader.ReadBytes(0)
	switch err {
		case nil:
		case io.EOF:
		default:
			return "", err
	}
	str := string(bytes)
	return strings.TrimSuffix(str, "\x00"), nil
}

func handleExecveCalls(notif *seccomp.ScmpNotifReq, fd seccomp.ScmpFd) error {
	var args []string
	argsAddr := notif.Data.Args
	for _, addr := range argsAddr {
		arg, _ := readArgFromMemory(
			int(notif.Pid),
			addr,
		)
		args = append(args, arg)
	}
	if len(args) == 0 {
		return errors.New("Could not read syscall arguments: empty")
	}
	switch filepath.Base(args[0]) {
		case "bash":
			debug.Println("PID", notif.Pid, "spawned a bash shell")
		case "chrome-sandbox":
			debug.Println("PID", notif.Pid, "spawned a chrome sandbox")
		default:
			debug.Println("Got execve() with arguments:", args)
	}
	// Do nothing now
	var resp = seccomp.ScmpNotifResp{
		ID:	notif.ID,
		Error:	0,
		Val:	0,
		Flags:	seccomp.NotifRespFlagContinue,
	}
	err := seccomp.NotifRespond(
		fd,
		&resp,
	)
	return err
}

func superviseSeccompNotif(fd seccomp.ScmpFd) {
	var wg sync.WaitGroup
	for {
		notif, err := seccomp.NotifReceive(fd)
		if err != nil {
			warn.Println(
				"Could not receive seccomp signal:",
				err,
			)
			terminateNotify <- 1
			return
		}
		wg.Go(func() {
			switch notif.Data.Syscall {
				case syscall.SYS_EXECVE:
					err := handleExecveCalls(notif, fd)
					if err != nil {
						warn.Println(
							"Could not handle",
							"execve():",
							err,
						)
					}
					return
				default:
			}

			callName, err := notif.Data.Syscall.GetName()
			if err != nil {
				callName = strconv.Itoa(int(notif.Data.Syscall))
			}

			warn.Println(
				"System call triggered:", callName,
				"from PID", notif.Pid,
				"using architecture", notif.Data.Arch.String(),
				"calling", notif.Data.Args,
				"which may be problematic",
			)

			// Do nothing now
			var resp = seccomp.ScmpNotifResp{
				ID:	notif.ID,
				Error:	0,
				Val:	0,
				Flags:	seccomp.NotifRespFlagContinue,
			}

			err = seccomp.NotifRespond(
				fd,
				&resp,
			)
			if err != nil {
				warn.Println(
					"Could not respond seccomp signal:",
					err,
				)
			}
		})
	}
}

// Panics on err
func addRuleToFilter(f *seccomp.ScmpFilter, call seccomp.ScmpSyscall, act seccomp.ScmpAction) {
	err := f.AddRule(call, act)
	if err != nil {
		panic(err)
	}
}

func createSeccompFilter() (err error) {
	filter, err := seccomp.NewFilter(seccomp.ActAllow)
	if err != nil {
		return
	}
	defer filter.Release()

	var notifyRules = []seccomp.ScmpSyscall{
		syscall.SYS_KILL,
		syscall.SYS_IOPERM,
		syscall.SYS_REBOOT,
		syscall.SYS_SETUID,
		syscall.SYS_SETGID,
		syscall.SYS_PTRACE,
		syscall.SYS_CHROOT,
		syscall.SYS_MOUNT,
		syscall.SYS_BIND,
		syscall.SYS_EXECVE,
	}

	for _, rule := range notifyRules {
		addRuleToFilter(filter, rule, seccomp.ActNotify)
	}

	err = filter.Precompute()
	if err != nil {
		return
	}
	err = filter.Load()
	if err != nil {
		return
	}
	fd, err := filter.GetNotifFd()
	if err != nil {
		return
	}
	go superviseSeccompNotif(fd)

	return
}