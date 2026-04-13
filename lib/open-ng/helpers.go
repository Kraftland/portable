package main

import (
	"errors"
	"io"
	"math/rand"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/godbus/dbus/v5"
)

func saveFile(path string) error {
	var isDir bool
	stat, err := os.Stat(path)
	if err != nil {
		return err
	}
	isDir = stat.IsDir()
	type pattern struct {
		Type		uint32
		Rule		string
	}
	type filter struct {
		Type		string
		List		[]pattern
	}

	var patter = pattern {
		Type:		0,
		Rule:		"*",
	}
	var patterMime = pattern {
		Type:		1,
		Rule:		"text/plain",
	}
	var defaultFilter = filter {
		Type:		"File",
	}
	var inodeFilter = filter {
		Type:		"Directory",
		List:		[]pattern{
			{
				Type:	0,
				Rule:	"*",
			},
			{
				Type:	1,
				Rule:	"inode/directory",
			},
		},
	}
	defaultFilter.List = append(defaultFilter.List, patter)
	defaultFilter.List = append(defaultFilter.List, patterMime)

	type portalResp struct {
		response	uint32
		results		map[string]dbus.Variant
	}
	responseChan := make(chan portalResp, 1)
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return err
	}
	obj := conn.Object(
		"org.freedesktop.portal.Desktop",
		"/org/freedesktop/portal/desktop",
	)
	const parentWindow string = ""
	var opts = make(map[string]dbus.Variant)
	token := "PortableOpen" + strconv.Itoa(rand.Int())
	opts["handle_token"] = dbus.MakeVariant(token)
	opts["accept_label"] = dbus.MakeVariant("OK")
	opts["current_name"] = dbus.MakeVariant(filepath.Base(path))
	opts["modal"] = dbus.MakeVariant(true)
	opts["filters"] = dbus.MakeVariant([]filter{defaultFilter, inodeFilter})
	switch isDir {
		case true:
			opts["current_filter"] = dbus.MakeVariant(inodeFilter)
		case false:
			opts["current_filter"] = dbus.MakeVariant(defaultFilter)
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func () {
		busname := conn.Names()[0]
		absName := strings.ReplaceAll(strings.TrimPrefix(busname, ":"), ".", "_")
		var objPath string = filepath.Join("/org/freedesktop/portal/desktop/request", absName, token)
		logger.Println("Resolved request object path:", objPath)
		err := conn.AddMatchSignal(
			dbus.WithMatchInterface("org.freedesktop.portal.Request"),
			dbus.WithMatchMember("Response"),
			dbus.WithMatchObjectPath(dbus.ObjectPath(objPath)),
			dbus.WithMatchSender("org.freedesktop.portal.Desktop"),
		)
		if err != nil {
			panic(err)
		}
		sigChan := make(chan *dbus.Signal, 8)
		conn.Signal(sigChan)
		wg.Done()
		for sig := range sigChan {
			if sig.Path == dbus.ObjectPath(objPath) && sig.Name == "org.freedesktop.portal.Request.Response" {
				logger.Println("Received response from", objPath)
				var resp portalResp
				resp.results = make(map[string]dbus.Variant)
				err := dbus.Store(sig.Body, &resp.response, &resp.results)
				if err != nil {
					warn.Println("Could not decode results:", err)
					continue
				}
				responseChan <- resp
				break
			}
		}
	} ()
	logger.Println("Base name:", filepath.Base(path))
	opts["directory"] = dbus.MakeVariant(false)
	opts["multiple"] = dbus.MakeVariant(false)
	wg.Wait()
	call := obj.Call(
		"org.freedesktop.portal.FileChooser.SaveFile",
		dbus.FlagAllowInteractiveAuthorization,
		parentWindow,
		"Export",
		opts,
	)
	logger.Println(call)
	if call.Err != nil {
		return call.Err
	}
	resp := <- responseChan
	switch resp.response {
		case 0:
		case 1:
			return nil
		case 2:
			return errors.New("User interaction was ended in some other way")
		default:
			return errors.New("Unexpected Response signal: " + strconv.Itoa(int(resp.response)))
	}

	var uris []string
	val, ok := resp.results["uris"]
	if ok {
		err := val.Store(&uris)
		if err != nil {
			return errors.New("Could not decode uris: " + err.Error())
		}
	} else {
		return errors.New("Could not find uris in response")
	}
	logger.Println("Got URIs:", uris)
	dirPaths := []string{}
	for _, val := range uris {
		escPath, _ := url.PathUnescape(val)
		dirPaths = append(dirPaths, strings.TrimPrefix(escPath, "file://"))
	}

	for idx := range dirPaths {
		wg.Go(func() {
			destFile, err := os.OpenFile(
				filepath.Join(
					dirPaths[idx],
					//filepath.Base(path),
				),
				os.O_WRONLY|os.O_TRUNC|os.O_CREATE,
				0700,
			)
			if err != nil {
				warn.Fatalln("Could not open destination file:", err)
			}
			defer destFile.Close()
			origFile, err := os.Open(
				path,
			)
			if err != nil {
				warn.Fatalln("Could not open origin file:", err)
			}
			defer origFile.Close()
			logger.Println("Streaming", origFile.Name(), "to", destFile.Name())
			_, err = io.Copy(destFile, origFile)
			if err != nil {
				warn.Fatalln("Could not stream file:", err)
			}
		})
	}

	wg.Wait()

	return nil
}

func evalPath(path string) (finalPath string) {
	inputAbs, _ := strings.CutPrefix(path, "file://")
	finalPath, err := filepath.Abs(inputAbs)
	if err != nil {
		warn.Fatalln("Could not get absolute path: " + err.Error())
		return
	}



	logger.Println("Resolved absolute path", finalPath)
	return
}