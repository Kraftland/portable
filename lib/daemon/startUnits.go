package main

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"

	systemd "github.com/coreos/go-systemd/v22/dbus"
)

func startRequiredUnits(sdConn *systemd.Conn) []error {
	var errSlice []error
	var errLock sync.Mutex
	var requiredUnits = []string{
		"pipewire.service",
		"xdg-desktop-portal.service",
	}
	var wantedUnits = []string{
		"pipewire-pulse.service",
		"pulseaudio.service",
	}

	ctx := context.TODO()
	ctxNew, cancelFunc := context.WithTimeout(ctx, 100 * time.Millisecond)
	unitStats, err := sdConn.ListUnitsByNamesContext(
		ctxNew,
		wantedUnits,
	)
	if err != nil {
		return []error{err}
	}

	for idx := range unitStats {
		if unitStats[idx].LoadState == "loaded" {
			requiredUnits = append(
				requiredUnits,
				unitStats[idx].Name,
			)
		}
	}

	contextNew := context.TODO()
	contextDeadline, cancelFunc := context.WithTimeout(
		contextNew,
		500 * time.Millisecond,
	)
	var wg sync.WaitGroup
	for idx := range requiredUnits {
		unit := requiredUnits[idx]
		wg.Go(func() {
			unitStats, err := sdConn.ListUnitsByNamesContext(
				contextDeadline,
				[]string{unit},
			)
			if err != nil {
				errLock.Lock()
				errSlice = append(errSlice, err)
				errLock.Unlock()
				return
			}
			if len(unitStats) != 1 {
				errLock.Lock()
				errSlice = append(errSlice, errors.New(
					"Expected 1 unit status, got " + strconv.Itoa(
						len(unitStats),
					),
				))
				errLock.Unlock()
				return
			}
			if unitStats[0].ActiveState == "active" {
				return
			}

			var resChan = make(chan string, 1)
			_, err = sdConn.StartUnitContext(
				contextDeadline,
				unit,
				"replace",
				resChan,
			)
			if err != nil {
				errLock.Lock()
				errSlice = append(errSlice, err)
				errLock.Unlock()
				return
			}
			result := <- resChan
			switch result {
				case "done":
				default:
					errLock.Lock()
					errSlice = append(errSlice, errors.New(
						"Could not start " + unit + ": " + result,
					))
					errLock.Unlock()
			}
		})
	}
	wg.Wait()
	cancelFunc()
	return errSlice
}