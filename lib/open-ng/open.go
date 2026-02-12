package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"github.com/rymdport/portal/openuri"
	"path/filepath"
	"regexp"
	"github.com/KarpelesLab/reflink"
)

var (
	linkRegexp	=	regexp.MustCompile(`[a-zA-Z][a-zA-Z0-9+.-]*://[^\s/$.?#].[^\s]*`);
)

func openPath(path string) {
	modPath, mod := evalPath(path)
	if len(modPath) == 0 {
		log.Fatalln("Failed to resolve path")
		return
	}

	stat, err := os.Stat(path)
	if err != nil {
		log.Fatalln("Could not stat path: " + err.Error())
		return
	}

	isDir := stat.IsDir()

	if mod == false {
		succ := openPathPortal(modPath, isDir)
		if succ == true {
			return
		}
	}
}

func evalPath(path string) (finalPath string, modified bool) {
	inputAbs, err := filepath.Abs(path)
	if err != nil {
		log.Fatalln("Could not get absolute path: " + err.Error())
		return
	}


	sandboxHome, err := filepath.Abs(os.Getenv("HOME"))
	if err != nil {
		log.Fatalln("Could not get home path: " + err.Error())
		return
	}

	if strings.HasPrefix(inputAbs, sandboxHome) {
		modified = false
		finalPath = inputAbs
		log.Println("Translated " + path + " to " + finalPath)
		return
	}

	openBlacklist := []string{
		"/sandbox",
		"/.flatpak-info",
		"/run",
		"/home",
		"/media",
		"/mnt",
		"/proc",
		"/root",
		"/srv",
		"/tmp",
		"top.kimiblock.portable",
		"/var",
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
		err := os.Symlink(inputAbs, sharedPath)
		if err != nil {
			log.Fatalln("Could not link path: " + err.Error())
		}
		finalPath = sharedPath
	}

	log.Println("Translated " + path + " to " + finalPath)
	return
}

func openPathPortal(path string, dir bool) (success bool) {
	opts := openuri.Options{
		Writable:	true,
		Ask:		true,
	}
	if dir {
		dir, err := os.Open(path)
		if err != nil {
			log.Fatal("Could not open path: " + err.Error())
		}
		fd := dir.Fd()
		err = openuri.OpenDirectory("", fd, &opts)
		if err != nil {
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
		var loopCounter uint = 1
		for {
			if loopCounter > uint(totalLength) {
				fmt.Println("Could not resolve path")
				break
			}
			_, err := os.Stat(os.Args[loopCounter])
			if err != nil {
				fmt.Println("Invalid argument: " + os.Args[loopCounter])
			} else {
				openPath(os.Args[loopCounter])
				break
			}
		}
	} else if strings.Contains(os.Args[1], "file://") == false || linkRegexp.Match([]byte(os.Args[1])) {
		fmt.Println("Got a link")
		openLink(os.Args[1])
	} else {
		openPath(os.Args[1])
	}
}