package main

import (
	"github.com/coreos/go-systemd/v22/dbus"
	"sync"
	godbus "github.com/godbus/dbus/v5"
	"context"
	"os"
)

// Sending integers to stopSignal will cause the whole program to exit with such code
func stopAppWorker(conn *dbus.Conn, sdCancelFunc func(), sdContext context.Context, busconn *godbus.Conn, stopSignal chan int, config Config) {
	sig := <- stopSignal
	pecho("debug", "Received a quit request from channel")
	var wg sync.WaitGroup

	for {
		select {
			case sig := <- stopFuncChan:
				sigFunc := sig
				wg.Go(sigFunc)
				continue
			default:
				pecho("debug", "Successfully drained stopFuncChan")
		}
		break
	}
	wg.Go(func() {
		if busconn == nil {
			pecho("warn", "Race detected: bus already terminated")
			return
		}
		reply, err := busconn.ReleaseName("top.kimiblock.portable." + config.Metadata.AppID)
		if err != nil {
			pecho("warn", "Could not request bus to release name:", err)
			return
		}
		switch reply {
			case godbus.ReleaseNameReplyReleased:
				pecho("debug", "Successfully released bus name")
			default:
				pecho("warn", "Could not release D-Bus name: " + reply.String())
		}
	})
	wg.Go(func() {
		doCleanUnit(conn, sdCancelFunc, sdContext, config)
	})
	wg.Wait()
	os.Exit(sig)
}