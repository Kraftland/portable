package main

import (
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/Kraftland/portable/portals"
	"github.com/godbus/dbus/v5"
)

func (m *busStartProcessor) RequestFSAccess (directory bool) (*dbus.Error) {
	errChan := make(chan error, 1)
	go func () {
		errChan <- requestFiles(directory)
	} ()
	return nil

}

func requestFiles(directory bool) error {
	uris, err := portals.FileChooser(portals.FileChooserOptions{
		Title:		"Import files into the sandbox",
		AcceptLabel:	"Share",
		Multiple:	true,
		Directory:	directory,
	})
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	err = os.MkdirAll(filepath.Join(home, "Shared"), 0700)
	if err != nil {
		panic(err)
	}
	var wg sync.WaitGroup
	for _, file := range uris {
		pth := file
		wg.Go(func() {
			destPath := filepath.Join(
				home,
				"Shared",
				filepath.Base(pth),
			)
			err := os.Symlink(
				pth,
				destPath,
			)
			if err != nil {
				panic(errors.New("Could not link " + pth + " to " + destPath + ": " + err.Error()))
			} else {
				filemapAdd(destPath, pth)
				debug.Println("Linked", pth, "to", destPath)
			}
		})
	}

	wg.Wait()
	return nil
}