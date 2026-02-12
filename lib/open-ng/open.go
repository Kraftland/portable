package main

import (
	"fmt"
	"log"
	"os"
	"os/user"
	"strings"
	"github.com/rymdport/portal/openuri"
	"path/filepath"
	"regexp"
	"github.com/KarpelesLab/reflink"
	"github.com/godbus/dbus/v5"
)

var (
	linkRegexp	=	regexp.MustCompile(`[a-zA-Z][a-zA-Z0-9+.-]*://[^\s/$.?#].[^\s]*`);
)

func openPath(path string, showItem bool) {
	modPath, _ := evalPath(path)

	log.Println("evalPath returned path", modPath)

	if len(modPath) == 0 {
		log.Fatalln("Failed to resolve path")
		return
	}

	stat, err := os.Stat(modPath)
	if err != nil {
		log.Fatalln("Could not stat path: " + err.Error())
		return
	}

	isDir := stat.IsDir()

	if showItem == false {
		succ := openPathPortal(modPath, isDir)
		if succ == true {
			return
		}
	}

	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		log.Fatalln("Could not connect to session bus: " + err.Error())
		return
	}
	defer conn.Close()
	log.Println("Calling FileManager1 for path: " + modPath)
	pathSlice := []string{"file://" + modPath}
	fileManager1Obj := conn.Object("org.freedesktop.FileManager1", "/org/freedesktop/FileManager1")
	fileManager1Obj.Call(
		"org.freedesktop.FileManager1.ShowItems",
		0,
		pathSlice,
		os.Getenv("appID"),
	)
}

func evalPath(path string) (finalPath string, modified bool) {
	inputAbs, err := filepath.Abs(path)
	if err != nil {
		log.Fatalln("Could not get absolute path: " + err.Error())
		return
	}

	inputAbs, _ = strings.CutPrefix(path, "file://")

	log.Println("Resolved absolute path", inputAbs)


	sandboxHome, err := filepath.Abs(os.Getenv("HOME"))
	if err != nil {
		log.Fatalln("Could not get home path: " + err.Error())
		return
	}

	var userName string
	userInfo, err := user.Current()
	if err != nil {
		log.Println("Could not get current user name")
	} else {
		userName = userInfo.Username
	}

	if inputAbs == sandboxHome {
		finalPath = sandboxHome
		return
	} else if strings.HasPrefix(inputAbs, filepath.Join("/home", userName)) {
		if strings.Contains(inputAbs, sandboxHome) == false {
			finalPath = sandboxHome
			return
		}
	} else if inputAbs == "/home" {
		finalPath = sandboxHome
		return
	}
	if strings.HasPrefix(inputAbs, sandboxHome) {
		modified = false
		finalPath = inputAbs
		log.Println("Translated sandbox path " + path + " to " + finalPath)
		return
	}

	openBlacklist := []string{
		"/sandbox",
		"/.flatpak-info",
		"/run",
		"/media",
		"/mnt",
		"/proc",
		"/root",
		"/srv",
		"/tmp",
		"top.kimiblock.portable",
		"/var",
		filepath.Join(sandboxHome, "options"),
		filepath.Join(sandboxHome, ".flatpak-info"),
		filepath.Join(sandboxHome, ".var"),
	}

	sharedPath := filepath.Join(
		sandboxHome,
		"Shared",
		filepath.Base(inputAbs),
	)

	for _, val := range openBlacklist {
		if strings.HasPrefix(inputAbs, val) {
			modified = true
			log.Println("Rewriting path")
			os.RemoveAll(sharedPath)
			err := reflink.Auto(inputAbs, sharedPath)
			if err != nil {
				log.Fatalln("Could not copy shared file: " + err.Error())
				return
			}
			finalPath = sharedPath
			break
		}
	}

	if modified == false {
		log.Println("Linking unknown path")
		os.RemoveAll(sharedPath)
		err := os.Symlink(inputAbs, sharedPath)
		if err != nil {
			log.Fatalln("Could not link path: " + err.Error())
		}
		finalPath = filepath.Dir(sharedPath)
	}

	log.Println("Translated " + path + " to " + finalPath)
	return
}

func openPathPortal(path string, dir bool) (success bool) {
	opts := openuri.Options{
		Writable:	false,
		Ask:		true,
	}
	log.Println("Calling portal for path: " + path)
	if dir {
		dir, err := os.Open(path)
		if err != nil {
			log.Fatal("Could not open path: " + err.Error())
		}
		fd := dir.Fd()
		err = openuri.OpenDirectory("", fd, &opts)
		if err != nil {
			log.Println("Portal denied request" + err.Error())
			return false
		} else {
			success = true
		}
	} else {
		file, err := os.OpenFile(path, os.O_RDONLY, 0700)
		if err != nil {
			log.Fatal("Could not open path: " + err.Error())
		}
		fd := file.Fd()
		err = openuri.OpenFile("", fd, &opts)
		if err != nil {
			log.Println("Portal denied request" + err.Error())
			return false
		} else {
			success = true
		}
	}
	return
}

func openLink(link string) {
	opts := openuri.Options{
		Writable:	true,
		Ask:		true,
	}
	err := openuri.OpenURI("", link, &opts)
	if err != nil {
		log.Fatalln("Could not call portal for opening links: " + err.Error())
	}
}

func main () {
	rawCmdArgs := os.Args
	fmt.Println("Received command line open request: " + strings.Join(rawCmdArgs, ", "))
	if os.Args[1] == "--show-item" {
		fmt.Println("Activating legacy dde-file-manager mode")
		totalLength := len(os.Args)
		var loopCounter uint = 2
		for {
			if loopCounter > uint(totalLength) {
				fmt.Println("Could not resolve path")
				break
			}
			_, err := os.Stat(os.Args[loopCounter])
			if err != nil {
				fmt.Println("Invalid argument: " + os.Args[loopCounter])
			} else {
				openPath(os.Args[loopCounter], true)
				break
			}
			loopCounter++
		}
	} else if strings.Contains(os.Args[1], "file://") == false && linkRegexp.Match([]byte(os.Args[1])) {
		fmt.Println("Got a link")
		openLink(os.Args[1])
	} else {
		openPath(os.Args[1], false)
	}
}