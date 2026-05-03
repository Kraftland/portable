package main

import (
	"bufio"
	"context"
	"os"
	"strings"
	"sync"
	"time"

	dbus "github.com/godbus/dbus/v5"
	udev "github.com/jochenvg/go-udev"
)

func gpuBind(gpuChan chan []string, config Config) {
	var gameModeEnabledChan = make(chan bool, 1)
	var chanWg sync.WaitGroup
	var wg sync.WaitGroup
	wg.Go(func() {
		if ! config.System.GameMode {
			gameModeEnabledChan <- false
			return
		}
		conn, err := dbus.SessionBus()
		if err != nil {
			gameModeEnabledChan <- config.System.GameMode
			pecho("warn", "Could not retrieve Low Power Mode status:", err)
			return
		}
		busObj := conn.Object(
			"org.freedesktop.portal.Desktop",
			"/org/freedesktop/portal/desktop",
		)
		ctx := context.TODO()
		ctxNew, cancelFunc := context.WithTimeout(ctx, 10 * time.Millisecond)

		call := busObj.CallWithContext(
			ctxNew,
			"org.freedesktop.DBus.Properties.Get",
			dbus.FlagNoAutoStart,
			"org.freedesktop.portal.PowerProfileMonitor",
			"power-saver-enabled",
		)
		cancelFunc()
		if call.Err != nil {
			pecho("warn", "Could not retrieve Low Power Mode status:", call.Err)
			gameModeEnabledChan <- config.System.GameMode
		}
		var powerSave bool
		err = call.Store(&powerSave)
		if err != nil {
			gameModeEnabledChan <- config.System.GameMode
			pecho("warn", "Could not retrieve Low Power Mode status:", err)
			return
		}
		switch powerSave {
			case true:
				pecho("warn", "Rejecting gameMode with Low Power Mode")
				gameModeEnabledChan <- false
			default:
				gameModeEnabledChan <- config.System.GameMode
		}
	})
	var argChan = make(chan []string, 128)
	var gpuArg = []string{"--tmpfs", "/dev/dri", "--tmpfs", "/sys/class/drm"}
	chanWg.Go(func() {
		for arg := range argChan {
			gpuArg = append(gpuArg, arg...)
		}
		gpuChan <- gpuArg
	})
	defer func () {
		wg.Wait()
		close(argChan)
		chanWg.Wait()
		close(gpuChan)
	} ()
	u := udev.Udev{}
	e := u.NewEnumerate()
	e.AddMatchIsInitialized()
	e.AddMatchSubsystem("drm")
	devs, errUdev := e.Devices()
	if errUdev != nil {
		pecho("warn", "Failed to query udev for GPU info")
	}




	// SHOULD contain strings like card0, card1 etc
	var totalGpus []GPUInfo

	var activeGpus []GPUInfo

	wg.Go(func() {
		var workers sync.WaitGroup
		defer workers.Wait()
		for _, path := range nvKernelModulePath {
			pth := path
			workers.Go(func() {
				argChan <- maskDir(pth)
			})
		}
	})


	for _, card := range devs {
		cardName := card.Sysname()
		cardPath := card.Syspath()
		devType := card.Devtype()
		if len(cardName) == 0 || len(cardPath) == 0 {
			pecho("warn", "Udev returned an empty sysname!")
			continue
		} else if ! strings.Contains(cardName, "card") || devType == "drm_connector" {
			pecho("debug", "Udev returned " + cardName + ", which is not a GPU")
			continue
		}
		totalGpus = append(
			totalGpus,
			GPUInfo{
				cardName:	cardName,
				cardPath:	cardPath,
			},
		)
	}
	wg.Wait()

	switch len(totalGpus) {
		case 0:
			pecho("warn", "Found no GPU")
		default:
			if <- gameModeEnabledChan {
				wg.Go(func() {
					setOffloadEnvs()
				})
				for _, cardInfo := range totalGpus {
					card := cardInfo.cardName
					wg.Go(func() {
						bindCard(card, argChan, config)
					})
				}
			} else {
				var activeGpuChan = make(chan GPUInfo, 5)
				var appendWg sync.WaitGroup
				appendWg.Go(func() {
					for sig := range activeGpuChan {
						activeGpus = append(activeGpus, sig)
					}
				})
				for _, cardInfo := range totalGpus {
					card := cardInfo
					wg.Go(func() {
						if detectCardStatus(card.cardPath, card.cardName) {
							activeGpuChan <- card
						}
					})
				}
				wg.Wait()
				close(activeGpuChan)
				appendWg.Wait()

				for _, cardInfo := range activeGpus {
					card := cardInfo
					wg.Go(func() {
						bindCard(
							card.cardName,
							argChan,
							config,
						)
					})
				}
			}
	}

	// TODO: Drop the debug output below
	//pecho("debug", "Generated GPU bind parameters:", gpuArg)
	pecho(
	"debug",
	"Total GPU count", len(totalGpus), "with active count:", activeGpus)
}

// Detects a card's status, true means connected
func detectCardStatus(cardPath string, cardNamed string) bool {
	connectors, err := os.ReadDir(cardPath)
	if err != nil {
		pecho(
			"warn",
			"Failed to read GPU connector status: " + err.Error(),
		)
		return false
	}
	for _, connectorName := range connectors {
		if strings.HasPrefix(connectorName.Name(), "card") == false {
			continue
		}
		conStatFd, err := os.OpenFile(
			cardPath + "/" + connectorName.Name() + "/status",
			os.O_RDONLY,
			0700,
		)
		if err != nil {
			pecho(
				"warn",
				"Failed to open GPU status: " + err.Error(),
			)
		} else {
			defer conStatFd.Close()
		}
		scanner := bufio.NewScanner(conStatFd)
		for scanner.Scan() {
			line := scanner.Text()
			switch line {
				case "disconnected":
					continue
				case "connected":
					return true
				default:
					pecho("warn", "Could not determine status of GPU: " + cardNamed)
			}
		}
	}
	return false
}

// Set required envs for PRIME render offloading
func setOffloadEnvs() {
	addEnv("VK_LOADER_DRIVERS_DISABLE=none")
	_, err := os.Stat("/dev/nvidia0")
	if err == nil {
		addEnv("__NV_PRIME_RENDER_OFFLOAD=1")
		addEnv("__VK_LAYER_NV_optimus=NVIDIA_only")
		addEnv("__GLX_VENDOR_LIBRARY_NAME=nvidia")
		addEnv("VK_LOADER_DRIVERS_SELECT=nvidia_icd.json")
	} else {
		addEnv("DRI_PRIME=1")
	}
}

func bindCard(cardName string, argChanFin chan []string, config Config) {
	var sendWg sync.WaitGroup
	var argComb = make(chan []string, 5)
	sendWg.Go(func() {
		var args []string
		for arg := range argComb {
			args = append(args, arg...)
		}
		argChanFin <- args
	})

	var wg sync.WaitGroup

	defer func () {
		wg.Wait()
		close(argComb)
		sendWg.Wait()
	} ()

	u := udev.Udev{}
	var cardID string
	var cardRoot string
	e := u.NewEnumerate()
	e.AddMatchSysname(cardName)
	e.AddMatchIsInitialized()
	e.AddMatchSubsystem("drm")

	devs, errUdev := e.Devices()
	if errUdev != nil {
		pecho("warn", "Failed to query udev for GPU info" + errUdev.Error())
		return
	}

	switch devsCnt := len(devs); devsCnt {
		case 0:
			pecho("warn", "Udev did not return any matching device for", cardName, "oh no")
			return
		case 1:
		default:
			pecho("warn", "Udev returned", devsCnt, "devices, of which should only be one")
	}

	var devNode string
	var sysPath string

	devNode = devs[0].Devnode()
	sysPath = devs[0].Syspath()
	cardRoot = strings.TrimSuffix(sysPath, "/drm/" + cardName)
	argComb <- []string{
		"--dev-bind",
		"/sys/class/drm/" + cardName,
		"/sys/class/drm/" + cardName,
		"--dev-bind",
		devNode,
		devNode,
		"--dev-bind",
		cardRoot,
		cardRoot,
	}
	cardID = devs[0].PropertyValue("ID_PATH")
	pecho("debug", "Got ID_PATH: " + cardID, "for card", cardName)

	// Detect NVIDIA now, because they do not expose ID_VENDOR properly
	wg.Go(func() {
		var bindWg sync.WaitGroup
		defer bindWg.Wait()
		vendorFile, openErr := os.OpenFile(cardRoot + "/vendor", os.O_RDONLY, 0700)
		if openErr != nil {
			pecho("warn", "Failed to open GPU vendor info " + openErr.Error())
			return
		}
		defer vendorFile.Close()
		scanner := bufio.NewScanner(vendorFile)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "0x10de" {
				bindWg.Go(func() {
					argComb <- tryBindNv()
				})
				bindWg.Go(func() {
					pecho("debug", "Detected NVIDIA device:", cardName)
					if config.Advanced.Zink {
						addEnv("__GLX_VENDOR_LIBRARY_NAME=mesa")
						addEnv("MESA_LOADER_DRIVER_OVERRIDE=zink")
						addEnv("GALLIUM_DRIVER=zink")
						addEnv("LIBGL_KOPPER_DRI2=1")
						addEnv("__EGL_VENDOR_LIBRARY_FILENAMES=/usr/share/glvnd/egl_vendor.d/50_mesa.json")
					}
				})
				for _, pth := range nvKernelModulePath {
					path := pth
					bindWg.Go(func() {
						stat, err := os.Stat(path)
						if err == nil && stat.IsDir() {
							argComb <- []string{
								"--ro-bind",
								path, path,
							}
						} else {
							pecho("debug", "Skipping non-existent path:", path)
							return
						}
					})
				}
			}
		}
	})


	// Map card* to renderD*
	eR := u.NewEnumerate()
	eR.AddMatchIsInitialized()
	eR.AddMatchSubsystem("drm")
	eR.AddMatchProperty("DEVTYPE", "drm_minor")
	//eR.AddMatchProperty("ID_PATH", cardID)
	devs, errUdev = eR.Devices()
	if errUdev != nil {
		pecho("warn", "Could not query udev for render node" + errUdev.Error())
	}
	switch devsCnt := len(devs); devsCnt {
		case 0:
			pecho("warn", "Could not translate", cardName, "to render node: did not receive any result from udev")
			return
		case 1:
		default:
			pecho("warn", "Udev returned more devices than required:", devsCnt)
	}

	var renderNodeName string
	var renderDevPath string
	for _, dev := range devs {
		if strings.Contains(dev.Sysname(), "card") {
			continue
		} else if dev.PropertyValue("ID_PATH") != cardID {
			pecho("debug", "Udev returned unknown card to us! ID: " + dev.PropertyValue("ID_PATH"))
			continue
		}
		renderNodeName = dev.Sysname()
		pecho("debug", "Got sysname: " + renderNodeName + ", with ID: " + dev.PropertyValue("ID_PATH"))
		renderDevPath = dev.Devnode()
		break
	}
	argComb <- []string{
		"--dev-bind",
			renderDevPath,
			renderDevPath,
		"--dev-bind",
			"/sys/class/drm/" + renderNodeName,
			"/sys/class/drm/" + renderNodeName,
	}

}