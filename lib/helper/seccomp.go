package main

import (
	"strconv"
	"sync"
	"syscall"

	"github.com/seccomp/libseccomp-golang"
)

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