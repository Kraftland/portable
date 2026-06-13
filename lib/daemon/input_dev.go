package main

import (
	"sync"

	udev "github.com/jochenvg/go-udev"
)

func enumerateDevices(subsystem string) ([]*udev.Device, error) {
	u := udev.Udev{}
	e := u.NewEnumerate()
	e.AddMatchSubsystem(subsystem)
	devs, err := e.Devices()
	if err != nil {
		return nil, err
	}
	return devs, nil
}

func collectDevices(devs []*udev.Device, inputBindChan chan []string) {
	var wg sync.WaitGroup
	defer wg.Wait()
	for _, dev := range devs {
		device := dev
		wg.Go(func() {
			if path := device.Syspath(); len(path) > 0 {
				inputBindChan <- []string{
					"--dev-bind",
					path,
					path,
				}
			}

			if devName := device.PropertyValue("DEVNAME"); len(devName) > 0 {
				inputBindChan <- []string{
					"--dev-bind",
					devName,
					devName,
				}
			}
			devlinks := device.Devlinks()
			for k := range devlinks {
				inputBindChan <- []string{
					"--dev-bind",
					k,
					k,
				}
			}
		})
	}
}

func inputBind(inputBindChan chan []string) {
	var wg sync.WaitGroup
	inputBindChan <- []string{
		"--dev-bind-try",	"/sys/class/leds", "/sys/class/leds",
		"--dev-bind-try",	"/sys/class/input", "/sys/class/input",
		"--dev-bind-try",	"/sys/class/hidraw", "/sys/class/hidraw",
		"--dev-bind-try",	"/dev/input", "/dev/input",
		"--dev-bind-try",	"/dev/uinput", "/dev/uinput",
	}

	wg.Go(func() {
		devs, err := enumerateDevices("input")
		if err != nil {
			pecho("warn", "Could not query udev for device info:", err)
			return
		}
		collectDevices(devs, inputBindChan)
	})
	wg.Go(func() {
		devs, err := enumerateDevices("hid")
		if err != nil {
			pecho("warn", "Could not query udev for device info:", err)
			return
		}
		collectDevices(devs, inputBindChan)
	})
	wg.Go(func() {
		devs, err := enumerateDevices("hidraw")
		if err != nil {
			pecho("warn", "Could not query udev for device info:", err)
			return
		}
		collectDevices(devs, inputBindChan)
	})

	wg.Wait()
	close(inputBindChan)
}