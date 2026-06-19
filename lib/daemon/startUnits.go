package main

import (
	"context"
	"errors"
	"sync"
	"time"

	systemd "github.com/coreos/go-systemd/v22/dbus"
)

func startRequiredUnits(sdConn *systemd.Conn) []error {
	var errSlice []error
	var errLock sync.Mutex

	var requiredUnits = []string{
		"pipewire.service",
	}
	contextNew := context.TODO()
	contextDeadline, cancelFunc := context.WithTimeout(
		contextNew,
		100 * time.Millisecond,
	)
	var wg sync.WaitGroup
	for idx := range requiredUnits {
		unit := requiredUnits[idx]
		wg.Go(func() {
			var resChan = make(chan string, 1)
			_, err := sdConn.StartUnitContext(
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