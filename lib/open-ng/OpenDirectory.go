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

func openDirectoryPortal(path string) error {
	var wg sync.WaitGroup
	fd, err := os.Open(
		path,
	)
	if err != nil {
		warn.Fatalln("Could not open path:", err)
	}
	defer fd.Close()
	busConn, err := dbus.SessionBus()
	if err != nil {
		panic(err)
	}
	portalObj := busConn.Object(
		"org.freedesktop.portal.Desktop",
		"/org/freedesktop/portal/desktop",
	)
	busname := busConn.Names()[0]
	absName := strings.ReplaceAll(strings.TrimPrefix(busname, ":"), ".", "_")
	var resp = make(chan uint32, 1)
	inId := "portableOpen" + strconv.Itoa(rand.Int())
	var objPath string = filepath.Join("/org/freedesktop/portal/desktop/request", absName, inId)

	wg.Add(1)

	go func () {
		err := busConn.AddMatchSignal(
			dbus.WithMatchInterface("org.freedesktop.portal.Request"),
			dbus.WithMatchMember("Response"),
			dbus.WithMatchObjectPath(dbus.ObjectPath(objPath)),
			dbus.WithMatchSender("org.freedesktop.portal.Desktop"),
		)
		if err != nil {
			panic(err)
		}
		sigChan := make(chan *dbus.Signal, 512)
		busConn.Signal(sigChan)
		wg.Done()
		for sig := range sigChan {
			logger.Println("Incoming signal")
			var v dbus.Variant
			var stat uint32
			if sig.Path == dbus.ObjectPath(objPath) && sig.Name == "org.freedesktop.portal.Request.Response" {
				err := dbus.Store(sig.Body, &stat, &v)
				if err != nil {
					warn.Fatalln("Could not store bus reply:", err)
				}
				resp <- stat
				break
			}
		}
		defer busConn.RemoveSignal(sigChan)
	} ()

	const parentWindow string = ""
	var optMap = make(map[string]dbus.Variant)
	optMap["handle_token"] = dbus.MakeVariant(inId)
	//optMap["writable"] = dbus.MakeVariant(true)
	optMap["ask"] = dbus.MakeVariant(true)
	wg.Wait()
	call := portalObj.Call(
		"org.freedesktop.portal.OpenURI.OpenFile",
		0,
		parentWindow,
		dbus.UnixFD(fd.Fd()),
		optMap,
	)
	if call.Err != nil {
		logger.Println("Call to Portal failed:", call.Err)
		return errors.New("Call to Portal failed: " + call.Err.Error())
	}
	res := <- resp
	logger.Println("Got response from Portal:", res)
	wg.Wait()
	switch res {
		case 0:
			os.Exit(0)
			return nil
		case 1:
			warn.Println("Interaction cancelled")
			return nil
		case 2:
			warn.Println("User interaction was ended in some other way")
			return errors.New("User interaction was ended in some other way")
		default:
			warn.Println("Unexpected Response signal:", res)
			return errors.New("Unexpected Response signal: " + strconv.Itoa(int(res)))
	}
}