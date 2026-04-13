package main

import (
	"errors"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/godbus/dbus/v5"
)

func (m *busStartProcessor) RequestFSAccess (directory bool) (*dbus.Error) {
	readyChan := make(chan bool, 2)
	errChan := make(chan error, 1)
	go func () {
		errChan <- requestFiles(directory, readyChan)
	} ()
	ready := <- readyChan
	if ! ready {
		err := <- errChan
		warn.Println("Could not request access:", err)
		return dbus.MakeFailedError(err)
	}
	return nil

}

func requestFiles(directory bool, ready chan bool) error {
	var errChan = make(chan error, 5)
	var statChan = make(chan uint, 1)
	var resChan = make(chan map[string]dbus.Variant, 1)
	var wg sync.WaitGroup
	id := "portableHelper" + strconv.Itoa(rand.Int())
	conn, err := dbus.SessionBus()
	if err != nil {
		ready <- false
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
	if call.Err != nil {
		ready <- false
		return call.Err
	}
	ready <- true
	status := <- statChan
	result := <- resChan
	switch status {
		case 0:
			debug.Println("Access granted by user")
		case 1:
			warn.Println("File sharing cancelled by user")
			return nil
		case 2:
			warn.Println("The user interaction was ended in some other way")
			ready <- false
			return errors.New("The user interaction was ended in some other way")
		default:
			warn.Println("Unknown response status:", status)
			ready <- false
			return errors.New("Unknown response status " + strconv.Itoa(int(status)))
	}
	var uris []string
	val, ok := result["uris"]
	if ok {
		err := val.Store(&uris)
		if err != nil {
			ready <- false
			return err
		}
		if len(uris) == 0 {
			warn.Println("Did not receive any URI to share")
			return nil
		}
	} else {
		ready <- false
		return errors.New("Did not receive any URI to share")
	}
	for idx, val := range uris {
		uris[idx], _ = strings.CutPrefix(val, "file://")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		ready <- false
		return err
	}
	err = os.MkdirAll(filepath.Join(home, "Shared"), 0700)
	if err != nil {
		ready <- false
		return err
	}
	for _, file := range uris {
		err := os.Symlink(
			file,
			filepath.Join(
				home,
				"Shared",
				filepath.Base(file),
			),
		)
		if err != nil {
			ready <- false
			return err
		}
	}

	for sig := range errChan {
		if sig != nil {
			ready <- false
			return errors.New("Unable to request files: " + sig.Error())
		}
	}
	return nil
}