package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
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
				case true:
					logger.Println("Enabled compatibility mode for --show-item")
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
			} else {
				return
			}
	}

	err = openPathFileManager1(modPath)
	if err != nil {
		warn.Println("Could not open path using the FileManager1 interface:", err)
	}
}
