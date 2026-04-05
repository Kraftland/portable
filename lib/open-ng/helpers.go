package main

import (
	"path/filepath"
	"strings"
	"os"
	"os/user"
	"github.com/KarpelesLab/reflink"
)

func evalPath(path string) (finalPath string, modified bool) {
	inputAbs, err := filepath.Abs(path)
	if err != nil {
		warn.Fatalln("Could not get absolute path: " + err.Error())
		return
	}

	inputAbs, _ = strings.CutPrefix(path, "file://")

	logger.Println("Resolved absolute path", inputAbs)


	sandboxHome, err := filepath.Abs(os.Getenv("HOME"))
	if err != nil {
		logger.Fatalln("Could not get home path: " + err.Error())
		return
	}

	var userName string
	userInfo, err := user.Current()
	if err != nil {
		logger.Println("Could not get current user name")
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
		logger.Println("Translated sandbox path " + path + " to " + finalPath)
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
			logger.Println("Rewriting path")
			os.RemoveAll(sharedPath)
			err := reflink.Auto(inputAbs, sharedPath)
			if err != nil {
				warn.Fatalln("Could not copy shared file: " + err.Error())
				return
			}
			finalPath = sharedPath
			break
		}
	}

	if modified == false {
		logger.Println("Linking unknown path")
		os.RemoveAll(sharedPath)
		err := os.Symlink(inputAbs, sharedPath)
		if err != nil {
			logger.Fatalln("Could not link path: " + err.Error())
		}
		finalPath = filepath.Dir(sharedPath)
	}

	logger.Println("Translated " + path + " to " + finalPath)
	return
}