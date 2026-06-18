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
			return "", err
		default:
			return "", err
	}
	str := string(bytes)
	return strings.TrimSuffix(str, "\x00"), nil
}

func handleExecveCalls(notif *seccomp.ScmpNotifReq, fd seccomp.ScmpFd) error {
	// var args []string
	// argsAddr := notif.Data.Args
	// for _, addr := range argsAddr {
	// 	arg, _ := readArgFromMemory(
	// 		int(notif.Pid),
	// 		addr,
	// 	)
	// 	args = append(args, arg)
	// }

	arg, err := readArgFromMemory(int(notif.Pid), notif.Data.Args[0])
	if err != nil {
		warn.Println("Could not read argv0 from memory:", err)
	}

	// if len(args) == 0 {
	// 	return errors.New("Could not read syscall arguments: empty")
	// }
	switch filepath.Base(arg) {
		case "bash":
			debug.Println("PID", notif.Pid, "spawned a bash shell")
		case "chrome-sandbox":
			debug.Println("PID", notif.Pid, "spawned a chrome sandbox")
		default:
			debug.Println(
				"Got execve() from PID",
				notif.Pid,
				"with argument:",
				arg,
				"Deciphered from memory address:",
				notif.Data.Args[0])
	}
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
	return err
}

/*
	This is not a security boundary!
	Specifically, it is very vulnerable to TOCTOU attacks and requires
		manually reading memory.
	See https://lore.kernel.org/all/20260504011207.539408-1-xiyou.wangcong@gmail.com/
		for an upcoming proposal which introduces
		SECCOMP_IOCTL_NOTIF_PIN_ARGS for race-free unotify

*/
func superviseSeccompNotif(fd seccomp.ScmpFd, lockdown bool) {
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
				"System call triggered: PID", notif.Pid,
				"requested", callName,
				"using architecture", notif.Data.Arch.String(),
				"with", notif.Data.Args,
				"which may be problematic",
			)
			var resp seccomp.ScmpNotifResp
			if lockdown {
				debug.Println("Rejecting syscall due to lockdown")
				resp = seccomp.ScmpNotifResp{
					ID:	notif.ID,
					//Error:	int32(syscall.EPERM),
					Error:	0,
				}
			} else {
				// Do nothing now
				resp = seccomp.ScmpNotifResp{
					ID:	notif.ID,
					Error:	0,
					//Val:	0,
					Flags:	seccomp.NotifRespFlagContinue,
				}
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
	var lockdown bool
	switch os.Getenv("_portableLockdown") {
		case "1":
			lockdown = true
	}
	filter, err := seccomp.NewFilter(seccomp.ActAllow)
	if err != nil {
		return
	}
	defer filter.Release()

	err = filter.SetBadArchAction(seccomp.ActAllow)
	if err != nil {
		return
	}

	var rejectRules = []string{
		"reboot",
		"setuid",
		"setgid",
	}

	for _, rule := range rejectRules {
		callID, err := seccomp.GetSyscallFromName(rule)
		if err != nil {
			panic(err)
		}
		addRuleToFilter(filter, callID, seccomp.ActKillThread)
	}

	var notifyRules = []string{
		"ioperm",
		"ptrace",
		// "chroot",
		"mount",
		"umount2",
		"umount",
		"fsmount",
		"fsconfig",
		"fsopen",
		"pivot_root",
		"move_mount",
		"fspick",

		"mount_setattr",
		// "unshare", // TODO: wait for kernel feature SECCOMP_IOCTL_NOTIF_PIN_ARGS
		"setns",
		"execve",
	}

	/* Note that future iterations will include support for
		zypak-like hijacking
	 That is, hijack stat() and Access()
	 	to return st_uid=0, st_mode=S_ISUID|S_IXOTH and 4755
	 And redirect execvp() calls
	 But it may not be necessary because the kernel will act as a security boundary
	*/

	for _, rule := range notifyRules {
		callID, err := seccomp.GetSyscallFromName(rule)
		if err != nil {
			panic(err)
		}
		addRuleToFilter(filter, callID, seccomp.ActNotify)
	}

	err = filter.SetNoNewPrivsBit(true)
	if err != nil {
		return
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
	go superviseSeccompNotif(fd, lockdown)

	return
}