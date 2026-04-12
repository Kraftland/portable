package main

import (
	"errors"
	"math/rand"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/godbus/dbus/v5"
)

func requestFiles(directory bool) error {
	var errChan = make(chan error, 5)
	var statChan = make(chan uint, 1)
	var resChan = make(chan map[string]dbus.Variant, 1)
	var wg sync.WaitGroup
	id := "portableHelper" + strconv.Itoa(rand.Int())
	conn, err := dbus.SessionBus()
	if err != nil {
		return err
	}
	wg.Add(1)
	go func () {
		sigChan := make(chan *dbus.Signal, 512)
		busname := conn.Names()[0]
		absName := strings.ReplaceAll(strings.TrimPrefix(busname, ":"), ".", "_")
		var objPath string = filepath.Join("/org/freedesktop/portal/desktop/request", absName, id)
		err := conn.AddMatchSignal(
			dbus.WithMatchInterface("org.freedesktop.portal.Request"),
			dbus.WithMatchMember("Response"),
			dbus.WithMatchObjectPath(dbus.ObjectPath(objPath)),
			dbus.WithMatchSender("org.freedesktop.portal.Desktop"),
		)
		if err != nil {
			errChan <- err
			wg.Done()
			return
		}
		conn.Signal(sigChan)
		wg.Done()
		for sig := range sigChan {
			if sig.Path == dbus.ObjectPath(objPath) && sig.Name == "org.freedesktop.portal.Request.Response" {
			} else {
				continue
			}
			debug.Println("Got incoming response signal")
			var stat uint
			var res dbus.Variant
			err := dbus.Store(sig.Body, &stat, &res)
			if err != nil {
				errChan <- err
				return
			}
			var resDecoded map[string]dbus.Variant
			err = res.Store(&resDecoded)
			if err != nil {
				errChan <- err
				return
			}
			statChan <- stat
			resChan <- resDecoded
		}
	} ()
	portalObj := conn.Object("org.freedesktop.portal.Desktop", "/org/freedesktop/portal/desktop")
	options := make(map[string]dbus.Variant)
	options["handle_token"] = dbus.MakeVariant(id)
	options["multiple"] = dbus.MakeVariant(true)
	options["directory"] = dbus.MakeVariant(directory)
	const parentWindow = ""
	wg.Wait()
	call := portalObj.Call(
		"org.freedesktop.portal.FileChooser.OpenFile",
		dbus.FlagAllowInteractiveAuthorization,
		parentWindow,
		"Import files to sandbox",
		options,
	)


	for sig := range errChan {
		if sig != nil {
			return errors.New("Unable to request files: " + sig.Error())
		}
	}
	return nil
}