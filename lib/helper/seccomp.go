package main

import (
	"sync"
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
			debug.Println(
				"Got syscall", notif.Data.Syscall,
				"from PID", notif.Pid,
				"using architecture", notif.Data.Arch.String(),
				"calling", notif.Data.Args,
			)

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
			if err != nil {
				warn.Println(
					"Could not respond seccomp signal:",
					err,
				)
			}
		})
	}
}

func createSeccompFilter() (err error) {
	filter, err := seccomp.NewFilter(seccomp.ActAllow)
	if err != nil {
		return
	}
	defer filter.Release()

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