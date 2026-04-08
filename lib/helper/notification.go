package main

import (
	"math/rand"
	"strconv"
	"sync"
	"slices"
	"github.com/godbus/dbus/v5"
)

type notification struct {
	notif	map[string]dbus.Variant
}

type button struct {
	label	string
	action	string
	target	dbus.Variant
	purpose	string
}

// Calls the AddNotification method, waits if a button is present
func addNotif(notif notification, waitActions []string) error {
	var errChan = make(chan error, 16)
	id := "portable" + strconv.Itoa(rand.Int())
	conn, err := dbus.SessionBus()
	if err != nil {
		return err
	}
	portalObj := conn.Object(
		"org.freedesktop.portal.Desktop",
		"/org/freedesktop/portal/desktop",
	)

	var wg sync.WaitGroup
	var listenReady = make(chan bool, 1)
	val, ok := notif.notif["buttons"]
	if ok {
		var buttons []button
		err := val.Store(&buttons)
		if err != nil {
			return err
		}
		wg.Go(func() {
			if len(waitActions) == 0 {
				return
			}
			err := conn.AddMatchSignal(
				dbus.WithMatchInterface("org.freedesktop.portal.Notification"),
				dbus.WithMatchMember("ActionInvoked"),
				dbus.WithMatchObjectPath("/org/freedesktop/portal/desktop"),
			)
			if err != nil {
				errChan <- err
			}
			sigChan := make(chan *dbus.Signal, 512)
			conn.Signal(sigChan)
			listenReady <- true
			for sig := range sigChan {
				if sig.Name == "org.freedesktop.portal.Notification.ActionInvoked" && sig.Path == "/org/freedesktop/portal/desktop" {
					var idReceived string
					var action string
					var parms []dbus.Variant
					err := dbus.Store(sig.Body, &idReceived, &action, &parms)
					if err != nil {
						continue
					}
					if idReceived != id {
						continue
					}
					if slices.Contains(waitActions, action) {
						return
					}
				}
			}
		})
	}
	<- listenReady
	call := portalObj.Call(
		"org.freedesktop.portal.Notification.AddNotification",
		dbus.FlagAllowInteractiveAuthorization,
		id,
		notif.notif,
	)
	if call.Err != nil {
		return call.Err
	}

	for sig := range errChan {
		if sig != nil {
			return sig
		}
	}


	return nil
}