package main

import (
	"sync"
	"github.com/godbus/dbus/v5"
	"os"
	"strconv"
	"strings"
	"math/rand"
	"path/filepath"
)

// Adds a file to the document Portal then opens it, does not handle directories!
func openFilePortal(path string) (success bool) {
	var wg sync.WaitGroup
	fd, err := os.Open(
		path,
	)
	if err != nil {
		warn.Fatalln("Could not open path:", err)
	}
	busConn, err := dbus.SessionBus()
	if err != nil {
		panic(err)
	}
	portalObj := busConn.Object(
		"org.freedesktop.portal.Desktop",
		"/org/freedesktop/portal/desktop",
	)
	docObj := busConn.Object(
		"org.freedesktop.portal.Documents",
		"/org/freedesktop/portal/documents",
	)
	call := docObj.Call(
		"org.freedesktop.portal.Documents.Add",
		0,
		dbus.UnixFD(fd.Fd()),
		true, // reuse_existing
		false, // persistent
	)
	if call.Err != nil {
		warn.Println("Could not add document to Portal Store:", call.Err)
		return false
	}
	var docId string
	err = call.Store(
		&docId,
	)
	if err != nil {
		warn.Fatalln("Could not store Document Portal reply:", err)
	} else {
		logger.Println("Got Document ID:", docId)
	}
	call = docObj.Call(
		"org.freedesktop.portal.Documents.GrantPermissions",
		0,
		docId,
		os.Getenv("appID"),
		[]string{"read", "write", "delete"},
	)
	if call.Err != nil {
		warn.Println("Could not grant permissions:", call.Err)
	}
	wg.Go(func() {
		err := fd.Close()
		if err != nil {
			warn.Println("Could not close file descriptor:", err)
		}
	})
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
			var v dbus.Variant
			var stat uint32
			if sig.Path == dbus.ObjectPath(objPath) && sig.Name == "org.freedesktop.portal.Request.Response" {
				logger.Println("Incoming signal from", sig.Path)
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
	call = docObj.Call(
		"org.freedesktop.portal.Documents.GetMountPoint",
		0,
	)
	if call.Err != nil {
		warn.Fatalln("Call to Document Portal failed:", call.Err)
	}
	var mntRaw []byte
	var mnt string
	err = call.Store(&mntRaw)
	if err != nil {
		warn.Fatalln("Could not store Document Portal reply:", err)
	} else {
		mnt = strings.TrimRight(
			string(mntRaw),
			"\x00",
		)
		logger.Println("Got document mount point:", mnt)
	}
	fd, err = os.Open(
		filepath.Join(
			mnt,
			docId,
			filepath.Base(path),
		),
	)
	if err != nil {
		warn.Fatalln("Could not open file:", err)
	}
	call = portalObj.Call(
		"org.freedesktop.portal.OpenURI.OpenFile",
		0,
		parentWindow,
		dbus.UnixFD(fd.Fd()),
		optMap,
	)
	if call.Err != nil {
		logger.Println("Call to Portal failed:", call.Err)
		return false
	} else {
		logger.Println("Called Portal:", call.Body)
	}
	res := <- resp
	logger.Println("Got response from Portal:", res)
	wg.Wait()
	switch res {
		case 0:
			return true
		case 1:
			warn.Println("Interaction cancelled")
			return true
		case 2:
			logger.Println("User interaction was ended in some other way")
			return false
		default:
			warn.Println("Unexpected Response signal:", res)
			return false
	}
}