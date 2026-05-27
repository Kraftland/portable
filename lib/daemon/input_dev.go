package main

import (
	"path/filepath"
	"strings"
	"sync"

	udev "github.com/jochenvg/go-udev"
)

func inputBind(inputBindChan chan []string) {
	var wg sync.WaitGroup
	inputBindChan <- []string{
		"--dev-bind-try",	"/sys/class/leds", "/sys/class/leds",
		"--dev-bind-try",	"/sys/class/input", "/sys/class/input",
		"--dev-bind-try",	"/sys/class/hidraw", "/sys/class/hidraw",
		"--dev-bind-try",	"/dev/input", "/dev/input",
		"--dev-bind-try",	"/dev/uinput", "/dev/uinput",
	}

	u := udev.Udev{}
	e := u.NewEnumerate()

	e.AddMatchSubsystem("input") // Later hidraw
	e.AddMatchIsInitialized()
	devs, errUdev := e.Devices()
	if errUdev != nil {
		pecho("warn", "Could not query udev for device info: " + errUdev.Error())
	}
	for _, dev := range devs {
		wg.Add(1)
		go func (device *udev.Device) {
			defer wg.Done()
			path := device.Syspath()
			if len(path) == 0 {
				return
			}
			sysSl := strings.Split(path, "/")
			sliceLen := len(sysSl)
			if strings.HasPrefix(sysSl[sliceLen - 1], "event") {
				if strings.HasPrefix(sysSl[sliceLen - 2], "input") {
					path = strings.Join(sysSl[0:sliceLen - 3], "/")
				}
			}
			inputBindChan <- []string{
			"--dev-bind",
				path,
				path,
			}
		} (dev)
	}

	hidrawE := u.NewEnumerate()
	hidrawE.AddMatchSubsystem("hidraw")
	rawDevs, errRawd := hidrawE.Devices()
	if errRawd != nil {
		pecho("warn", "Could not query udev for hidraw devices: " + errRawd.Error())
	}

	for _, dev := range rawDevs {
		wg.Add(1)
		go func (device *udev.Device) {
			defer wg.Done()
			path := device.Syspath()
			devPath := strings.TrimSpace(dev.PropertyValue("DEVNAME"))
			if len(devPath) > 0 {
				inputBindChan <- []string{
					"--dev-bind",
					devPath,
					devPath,
				}
			}
			if len(path) > 0 {
				dirPth := filepath.Dir(path)
				if strings.HasPrefix(filepath.Base(dirPth), "hidraw") {
					inputBindChan <- []string {
						"--dev-bind",
						dirPth,
						dirPth,
					}
				}
			}

		} (dev)
	}

	wg.Wait()
	close(inputBindChan)
}