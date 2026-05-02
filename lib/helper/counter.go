package main

import (
	"os"
	"sync"
	"io"
	"github.com/coreos/go-systemd/v22/daemon"
)

func startCounterV2(chann chan StartNofifyMsg) {
	var mu sync.RWMutex
	var cntChan = make(chan bool, 1024)
	var cnt uint
	defer close(cntChan)
	go func () {
		for sig := range cntChan {
			if sig {
				mu.Lock()
				cnt++
				mu.Unlock()
			} else {
				mu.Lock()
				cnt--
				mu.Unlock()
			}
			mu.RLock()
			if cnt == 0 {
				daemon.SdNotify(false, daemon.SdNotifyStopping)
				debug.Println("All tracked processes have exited")
				terminateNotify <- 1
				return
			}
			go updateSd(int(cnt))
			mu.RUnlock()
		}
	} ()

	for sig := range chann {
		go startAux(sig, cntChan)
	}
}

func rmSockDir(path string) {
	if len(path) == 0 {
		return
	}
	err := os.RemoveAll(path)
	if err != nil {
		warn.Println("Unable to clean up streaming directory:", err)
	} else {
		debug.Println("Cleaned up", path)
	}
}

func startAux(msg StartNofifyMsg, ch chan bool) {
	defer rmSockDir(msg.sockDir)
	ch <- true
	var streamWg sync.WaitGroup
	if len(msg.UDS) != 3 {
		warn.Println("UDS count mismatch: got", len(msg.UDS))
		msg.cmd.Stdin = os.Stdin
		msg.cmd.Stdout = os.Stdout
		msg.cmd.Stderr = os.Stderr
	} else {
		streamWg.Add(3)
		go func () {
			if msg.UDS[0] == nil {
				warn.Println("Could not stream: nil socket")
				streamWg.Done()
				return
			}
			conn, err := msg.UDS[0].Accept()
			if err != nil {
				warn.Println("Could not accept connection:", err)
				streamWg.Done()
				return
			}
			defer conn.Close()
			inP, err := msg.cmd.StdinPipe()
			streamWg.Done()
			if err != nil {
				warn.Println("Could not accept connection:", err)
				return
			}
			defer inP.Close()
			n, err := io.Copy(inP, conn)
			debug.Println("Streamed", n, "bytes of stdin")
		} ()
		go func () {
			if msg.UDS[1] == nil {
				warn.Println("Could not stream: nil socket")
				streamWg.Done()
				return
			}
			conn, err := msg.UDS[1].Accept()
			if err != nil {
				warn.Println("Could not accept connection:", err)
				return
			}
			defer conn.Close()
			pipe, err := msg.cmd.StdoutPipe()
			streamWg.Done()
			if err != nil {
				warn.Println("Could not accept connection:", err)
				return
			}
			defer pipe.Close()
			n, err := io.Copy(conn, pipe)
			debug.Println("Streamed", n, "bytes of stdout")
		} ()
		go func () {
			if msg.UDS[2] == nil {
				warn.Println("Could not stream: nil socket")
				streamWg.Done()
				return
			}
			conn, err := msg.UDS[2].Accept()
			if err != nil {
				warn.Println("Could not accept connection:", err)
				streamWg.Done()
				return
			}
			defer conn.Close()
			pipe, err := msg.cmd.StderrPipe()
			streamWg.Done()
			if err != nil {
				warn.Println("Could not accept connection:", err)
				streamWg.Done()
				return
			}
			defer pipe.Close()
			n, err := io.Copy(conn, pipe)
			debug.Println("Streamed", n, "bytes of stderr")
		} ()
	}
	streamWg.Wait()
	err := msg.cmd.Run()
	if err != nil {
		warn.Println("Command", msg.cmd.Path, msg.cmd.Args, "failed:", err)
	}
	ch <- false
}