package main

import (
	"os"
	"path/filepath"
	"sync"
)

func testWaylandSocket(path string) error {
	_, err := os.Stat(path)
	return err
}

func waylandDisplay(wdChan chan []string) () {
	type wDisplay struct {
		Path		string
		Priority	int
	}
	sessionType := os.Getenv("XDG_SESSION_TYPE")
	switch sessionType {
		case "x11":
			pecho("warn", "Running on X11, this is insecure")
			return
		case "wayland":
		default:
			pecho("warn", "Unknown XDG_SESSION_TYPE, treating as wayland")
	}

	socketInfo := os.Getenv("WAYLAND_DISPLAY")
	var wg sync.WaitGroup
	resChan := make(chan wDisplay, 3)
	wg.Go(func() {
		if len(os.Getenv("WAYLAND_DISPLAY")) > 0 {
			pecho("debug", "Not using default socket: WAYLAND_DISPLAY set")
			return
		}
		pth := filepath.Join(xdgDir.runtimeDir, "wayland-0")
		err := testWaylandSocket(pth)
		if err != nil {
			pecho("debug", "Could not use socket:", err)
		} else {
			resChan <- wDisplay{
				Path:		pth,
				Priority:	1,
			}
		}
	})
	wg.Go(func() {
		pth := filepath.Join(xdgDir.runtimeDir, socketInfo)
		err := testWaylandSocket(pth)
		if err != nil {
			pecho("debug", "Could not use socket:", err)
		} else {
			resChan <- wDisplay{
				Path:		pth,
				Priority:	2,
			}
		}
	})
	wg.Go(func() {
		pth := socketInfo
		err := testWaylandSocket(pth)
		if err != nil {
			pecho("debug", "Could not use socket:", err)
		} else {
			resChan <- wDisplay{
				Path:		pth,
				Priority:	3,
			}
		}
	})
	go func () {
		wg.Wait()
		close(resChan)
	} ()

	var result wDisplay
	for res := range resChan {
		if res.Priority > result.Priority {
			result = res
		}
	}
	if result.Priority == 0 {
		pecho("crit", "Could not find a useable Wayland socket")
	}

	var waylandArgs = []string{
		"--ro-bind",
			result.Path,
			xdgDir.runtimeDir + "/wayland-0",
	}
	wdChan <- waylandArgs
	pecho("debug", "Found Wayland socket: " + result.Path)
}