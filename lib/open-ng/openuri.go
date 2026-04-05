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

func OpenURI(uri string) error {
	logger.Println("Calling OpenURI for opening link:", uri)
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return err
	}
	portalObj := conn.Object(
		"org.freedesktop.portal.Desktop",
		"/org/freedesktop/portal/desktop",
	)
	busname := conn.Names()[0]
	absName := strings.ReplaceAll(strings.TrimPrefix(busname, ":"), ".", "_")
	var resp = make(chan uint32, 1)
	inId := "portableOpen" + strconv.Itoa(rand.Int())
	var objPath string = filepath.Join("/org/freedesktop/portal/desktop/request", absName, inId)
	var wg sync.WaitGroup
	wg.Add(1)
	go func () {
		err := conn.AddMatchSignal(
			dbus.WithMatchInterface("org.freedesktop.portal.Request"),
			dbus.WithMatchMember("Response"),
			dbus.WithMatchObjectPath(dbus.ObjectPath(objPath)),
			dbus.WithMatchSender("org.freedesktop.portal.Desktop"),
		)
		if err != nil {
			panic(err)
		}
		sigChan := make(chan *dbus.Signal, 512)
		conn.Signal(sigChan)
		wg.Done()
		for sig := range sigChan {
			logger.Println("Incoming signal")
			var v dbus.Variant
			var stat uint32
			if sig.Path == dbus.ObjectPath(objPath) && sig.Name == "org.freedesktop.portal.Request.Response" {
				err := dbus.Store(sig.Body, &stat, &v)
				if err != nil {
					logger.Fatalln("Could not store bus reply:", err)
				}
				resp <- stat
				break
			}
		}
		defer conn.RemoveSignal(sigChan)
	} ()
	const parentWindow string = ""
	var optMap = make(map[string]dbus.Variant)
	optMap["handle_token"] = dbus.MakeVariant(inId)
	optMap["writable"] = dbus.MakeVariant(true)
	optMap["ask"] = dbus.MakeVariant(true)
	call := portalObj.Call(
		"org.freedesktop.portal.OpenURI.OpenURI",
		0,
		parentWindow,
		uri,
		optMap,
	)
	if call.Err != nil {
		return call.Err
	}
	res := <- resp
	logger.Println("Got response on request")
	switch res {
		case 0:
			os.Exit(0)
			return nil
		case 1:
			logger.Println("Interaction cancelled")
			os.Exit(0)
			return nil
		case 2:
			logger.Println("User interaction was ended in some other way")
			os.Exit(0)
			return errors.New("User interaction was ended in some other way")
		default:
			logger.Println("Unexpected Response signal:", res)
			return errors.New("Unexpected Response signal: " + strconv.Itoa(int(res)))
	}
}