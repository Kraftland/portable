package main

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"encoding/json"

	godbus "github.com/godbus/dbus/v5"
)

func engageExpose(chann chan map[string]string, conf Config, docsChan chan PassFiles) []string {
	close(chann)
	var bwArgs []string
	var bwArgChan = make(chan []string, 512)
	var portalFiles = make(chan string, 512)
	var wg sync.WaitGroup
	var writeWg sync.WaitGroup

	var pathsChan = make(chan string, 512)
	var consentChan = make(chan bool, 1)
	go func () {
		var paths []string
		for sig := range pathsChan {
			paths = append(paths, sig)
		}
		if len(paths) == 0 {
			return
		}
		consentChan <- questionExpose(paths, conf)
	} ()



	writeWg.Go(func() {
		for sig := range bwArgChan {
			bwArgs = append(bwArgs, sig...)
		}
	})
	writeWg.Go(func() {
		consent := <- consentChan
		if ! consent {
			return
		}
		consentChan <- consent

		files := []string{}
		for sig := range portalFiles {
			files = append(files, sig)
		}
		if len(files) == 0 {
			return
		}
		conn, err := godbus.SessionBus()
		if err != nil {
			pecho("crit", "Could not connect to session bus:", err)
		}
		addFilesToPortal(conn, files, docsChan, conf)
	})


	for pthMap := range chann {
		for k, v := range pthMap {
			ori := k
			dest := v
			wg.Go(func() {
				stat, err := os.Stat(ori)
				if err != nil {
					pecho("warn", "Could not stat path:", err)
				}
				pathsChan <- ori
				if strings.HasPrefix(dest, "ro:") {
					bwArgChan <- []string{
						"--ro-bind",
						ori,
						strings.TrimPrefix(dest, "ro:"),
					}
				} else if strings.HasPrefix(dest, "dev:") {
					bwArgChan <- []string{
						"--dev-bind",
						ori,
						strings.TrimPrefix(dest, "dev:"),
					}
				} else if dest == "null" {
				} else {
					bwArgChan <- []string{
						"--bind",
						ori,
						dest,
					}
				}
				if filepath.IsAbs(ori) && ! stat.IsDir() {
					portalFiles <- ori
				} else {
					pecho("warn", "Skipping descriptor passing: either the path is not absolute, or is a directory")
				}
			})
		}

	}
	wg.Wait()
	close(bwArgChan)
	close(portalFiles)
	close(pathsChan)
	writeWg.Wait()
	if consent := <- consentChan; consent {
		return bwArgs
	} else {
		return []string{}
	}

}

func addFilesToPortal(connBus *godbus.Conn, pathList []string, filesInfo chan PassFiles, config Config) {
	pecho("debug", "Passing", pathList, "via file descriptor")
	//pecho("warn", "Calling Portal to add files:", pathList)
	var filesInfoTmp PassFiles
	filesInfoTmp.FileMap = map[string]string{}
	var busFdList []godbus.UnixFD
	for _, path := range pathList {
		fileObj, err := os.Open(path)
		if err != nil {
			pecho("warn", "Could not open file: " + err.Error())
			continue
		}
		defer fileObj.Close()
		filesInfoTmp.FileMap[path] = "unknown"
		fd := fileObj.Fd()
		busFdList = append(busFdList, godbus.UnixFD(fd))
	}
	type AddDocumentFullData struct {
		PathFDs		[]godbus.UnixFD
		Flags		uint32
		AppID		string
		Permissions	[]string
	}
	var busData	AddDocumentFullData
	busData.AppID = config.Metadata.AppID
	busData.Flags = 1
	busData.PathFDs = busFdList
	busData.Permissions = []string{"read", "write", "grant-permissions"}

	path := "/org/freedesktop/portal/documents"
	pathBus := godbus.ObjectPath(path)

	obj := connBus.Object("org.freedesktop.portal.Documents", pathBus)
	pecho("debug", "Requesting Documents portal for IDs...")
	call := obj.Call("org.freedesktop.portal.Documents.AddFull", 0,
		busData.PathFDs,
		busData.Flags,
		busData.AppID,
		busData.Permissions,
	)
	//<- call.Done
	pecho("debug", "AddFull call done")
	if call.Err != nil {
		pecho("warn", "Could not contact Documents portal: " + call.Err.Error())
	}
	type PortalResponse struct {
		DocIDs		[]string
		ExtraInfo	map[string]godbus.Variant
	}
	var resp PortalResponse
	err := godbus.Store(call.Body, &resp.DocIDs, &resp.ExtraInfo)
	if err != nil {
		pecho("warn", "Could not decode portal response: " + err.Error())
	}
	for idx, docid := range resp.DocIDs {
		filesInfoTmp.FileMap[pathList[idx]] = filepath.Join(
			xdgDir.runtimeDir,
			"/doc/",
			docid,
			filepath.Base(pathList[idx]),
		)
	}
	jsonObj, _ := json.Marshal(filesInfoTmp)
	addEnv("_portableHelperExtraFiles=" + string(jsonObj))
	pecho("debug", "Passed files info: " + string(jsonObj))
	filesInfo <- filesInfoTmp
}