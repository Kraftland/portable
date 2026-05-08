package main

import (
	"os"
	"sync"
	"strings"
)

func main() {
	logger.Println("Received open request:", os.Args)
	if len(os.Args) <= 1 {
		warn.Println("open requires at least one destination")
	}
	var showItem bool
	var showLock sync.Mutex
	var wg sync.WaitGroup

	for _, arg := range os.Args[1:] {
		argv := arg
		if argv == "--show-item" {
			showLock.Lock()
			showItem = true
			showLock.Unlock()
		} else if ! strings.Contains(argv, "file://") && linkRegexp.Match([]byte(argv)) {
			wg.Go(func() {
				err := OpenURI(argv)
				if err != nil {
					warn.Println("Could not open link via Portal:", err)
				}
			})
		} else {
			showLock.Lock()
			wg.Go(func() {
				openPath(arg, showItem)
			})
			showLock.Unlock()
		}
	}

	wg.Wait()
}