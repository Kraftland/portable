package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-systemd/v22/dbus"
	godbus "github.com/godbus/dbus/v5"
)

func trayWakeNG(config Config, conn *godbus.Conn) error {
	pecho("debug", "Attempting tray wakeup")
	const cgMnt string = "/sys/fs/cgroup"

	busObj := conn.Object(
		"top.kimiblock.portable." + config.Metadata.AppID,
		"/top/kimiblock/portable/daemon",
	)
	call := busObj.Call("top.kimiblock.portable.Info.GetInfo", 0)
	if call.Err != nil {
		pecho("warn", "Could not obtain instance information: " + call.Err.Error())
		return call.Err
	}
	replyArry := []string{}
	err := call.Store(&replyArry)
	if err != nil {
		pecho("warn", "Could not store D-Bus reply: " + err.Error())
		return err
	}
	var id string
	for _, val := range replyArry {
		str, has := strings.CutPrefix(val, "Instance ID: ")
		if has {
			id = str
			break
		}
	}
	if len(id) == 0 {
		return errors.New("Could not obtain instance ID: reply invalid")
	}
	ctx := context.Background()
	ctxNew, cancelFunc := context.WithTimeout(ctx, 10 *time.Second)
	defer cancelFunc()
	sdConn, err := dbus.NewUserConnectionContext(ctxNew)
	if err != nil {
		return err
	}
	var ret = make(map[string]any)
	ret, err = sdConn.GetAllPropertiesContext(ctx, config.Metadata.FriendlyName + "-" + id + "-dbus.service")
	if err != nil {
		return err
	}
	sdConn.Close()
	var cgPath string
	cgroupPath, ok := ret["ControlGroup"]
	if ok {
		cgPath = parseStr(cgroupPath)
	} else {
		return errors.New("Could not obtain bus control group: reply invalid")
	}
	pecho("debug", "Obtained Bus control group: " + cgPath)

	pidsFile, err := os.OpenFile(
		filepath.Join(cgMnt, cgPath, "cgroup.procs"),
		os.O_RDONLY,
		0700,
	)
	if err != nil {
		return err
	}
	defer pidsFile.Close()

	scanner := bufio.NewScanner(pidsFile)
	var pids []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 {
			continue
		}
		pids = append(pids, line)
	}
	var registeredNotifs []string
	trayObj := conn.Object("org.kde.StatusNotifierWatcher", "/StatusNotifierWatcher")
	call = trayObj.Call("org.freedesktop.DBus.Properties.Get", 0, "org.kde.StatusNotifierWatcher", "RegisteredStatusNotifierItems")
	if call.Err != nil {
		return call.Err
	}
	err = call.Store(&registeredNotifs)
	if err != nil {
		return err
	}
	if len(registeredNotifs) == 0 {
		return errors.New("No registered tray icon")
	}
	busObjUID := conn.Object("org.freedesktop.DBus", "/org/freedesktop/DBus")
	var wg sync.WaitGroup
	for _, notif := range registeredNotifs {
		wg.Go(func() {
			//var newStyle bool
			var name string
			if strings.Contains(notif, "@") {
				//newStyle = true
				sp := strings.Split(notif, "@")
				name = sp[0]
			} else if strings.Contains(notif, "/") {
				sp := strings.Split(notif, "/")
				name = sp[0]
			} else {
				name = notif
			}
			pecho("debug", "Trying tray name " + name)
			call := busObjUID.Call("org.freedesktop.DBus.GetConnectionUnixProcessID", 0, name)
			if call.Err != nil {
				pecho("warn", "Could not get peer PID: " + call.Err.Error())
				return
			}
			var pid int
			err := call.Store(&pid)
			if err != nil {
				pecho("warn", "Could not get peer PID: " + err.Error())
				return
			}
			pidStr := strconv.Itoa(pid)
			if slices.Contains(pids, pidStr) {
				pecho("debug", "Calling Activate on Tray...")
				tray := conn.Object(name, "/StatusNotifierItem")
				call := tray.Call("org.kde.StatusNotifierItem.Activate", 0, int32(1), int32(18))
				if call.Err != nil {
					pecho("warn", "Could not call for tray wakeup: " + call.Err.Error())
					fmt.Println(call.Err)
					return
				}
			} else {
				return
			}
		})
	}

	wg.Wait()

	return nil
}
