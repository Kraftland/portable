package main

import (
	"math/rand"
	"os"

	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/godbus/dbus/v5"
)

var (
	linkRegexp	=	regexp.MustCompile(`[a-zA-Z][a-zA-Z0-9+.-]*://[^\s/$.?#].[^\s]*`);
	docMount	=	filepath.Join("/run/user", strconv.Itoa(os.Getuid()), "doc")
)

func openPath(path string, showItem bool) {
	modPath := evalPath(path)

	logger.Println("evalPath returned path", modPath)

	if len(modPath) == 0 {
		warn.Fatalln("Failed to resolve path")
		return
	}

	succ := openPathPortal(modPath, showItem)
	if succ == true {
		return
	}

	succ = openPathPortal(modPath, true)
	if succ == true {
		return
	}

	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		warn.Fatalln("Could not connect to session bus: " + err.Error())
		return
	}
	defer conn.Close()
	logger.Println("Calling FileManager1 for path: " + modPath)
	pathSlice := []string{"file://" + modPath}
	fileManager1Obj := conn.Object("org.freedesktop.FileManager1", "/org/freedesktop/FileManager1")
	fileManager1Obj.Call(
		"org.freedesktop.FileManager1.ShowItems",
		0,
		pathSlice,
		os.Getenv("appID"),
	)
}

func openPathPortal(path string, showItem bool) (success bool) {
	var wg sync.WaitGroup
	fd, err := os.Open(path)
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
		warn.Fatalln("Could not add document to Portal Store:", call.Err)
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
	optMap["writable"] = dbus.MakeVariant(true)
	optMap["ask"] = dbus.MakeVariant(true)
	var busMethod string
	switch showItem {
		case true:
			busMethod = "OpenDirectory"
		case false:
			busMethod = "OpenFile"
	}
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
		"org.freedesktop.portal.OpenURI."+ busMethod,
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
			os.Exit(0)
			return true
		case 1:
			logger.Println("Interaction cancelled")
			os.Exit(0)
			return true
		case 2:
			logger.Println("User interaction was ended in some other way")
			return false
		default:
			warn.Println("Unexpected Response signal:", res)
			return false
	}
}

func main () {
	rawCmdArgs := os.Args
	logger.Println("Received command line open request:", rawCmdArgs)
	if os.Args[1] == "--show-item" {
		logger.Println("Enabled compatibility mode for --show-item")
		totalLength := len(os.Args)
		var loopCounter uint = 2
		for {
			if loopCounter > uint(totalLength) {
				logger.Println("Could not resolve path")
				break
			}
			_, err := os.Stat(os.Args[loopCounter])
			if err != nil {
				logger.Println("Invalid argument: " + os.Args[loopCounter])
			} else {
				openPath(os.Args[loopCounter], true)
				break
			}
			loopCounter++
		}
	} else if strings.Contains(os.Args[1], "file://") == false && linkRegexp.Match([]byte(os.Args[1])) {
		logger.Println("Got a link")
		err := OpenURI(os.Args[1])
		if err != nil {
			warn.Fatalln("Could not open link:", err)
		}
	} else {
		openPath(os.Args[1], false)
	}
}