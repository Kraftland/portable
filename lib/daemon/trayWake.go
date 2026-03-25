package main

import (
	"bufio"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/coreos/go-systemd/dbus"
	"github.com/coreos/go-systemd/v22/dbus"
	godbus "github.com/godbus/dbus/v5"
)

func trayWakeNG(config Config, conn *godbus.Conn) error {
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
	sdConn := dbus.NewUserConnectionContext(ctxNew)
	var ret = make(map[string]interface{})
	ret, err = sdConn.GetUnitProperties()
	cancelFunc()
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
	scanner := bufio.NewScanner(pidsFile)
	var pids []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 {
			continue
		}
		pids = append(pids, line)
	}
}
