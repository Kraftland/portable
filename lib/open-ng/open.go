package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"github.com/godbus/dbus/v5"
)

var (
	linkRegexp	=	regexp.MustCompile(`[a-zA-Z][a-zA-Z0-9+.-]*://[^\s/$.?#].[^\s]*`);
	docMount	=	filepath.Join("/run/user", strconv.Itoa(os.Getuid()), "doc")
)

func openPath(path string, showItem bool) {
	modPath := evalPath(path)

	if len(modPath) == 0 {
		warn.Fatalln("Failed to resolve path")
		return
	}

	logger.Println("evalPath returned path", modPath)

	stat, err := os.Stat(modPath)
	if err != nil {
		warn.Fatalln("Could not open file or directory", err)
	}
	switch stat.IsDir() {
		case true:
			err := openDirectoryPortal(modPath)
			if err != nil {
				warn.Println("Could not call Portal to open a directory:", err)
			} else {
				return
			}
		case false:
			switch showItem {
				case false:
					succ := openFilePortal(modPath)
					if ! succ {
						warn.Println("Could not open file using OpenFile, falling back...")
					} else {
						return
					}
			}
			err := openDirectoryPortal(modPath)
			if err != nil {
				warn.Println("Could not call Portal to open a directory:", err)
			}
	}

	conn, err := dbus.SessionBus()
	if err != nil {
		warn.Fatalln("Could not connect to session bus: " + err.Error())
		return
	}
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