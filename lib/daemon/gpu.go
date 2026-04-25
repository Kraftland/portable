package main

import (
	"sync"
	udev "github.com/jochenvg/go-udev"
	"strings"
	"os"
	"bufio"
)

func gpuBind(gpuChan chan []string, config Config) {
	var chanWg sync.WaitGroup
	var wg sync.WaitGroup
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
	var totalGpus = []string{}
	var activeGpus = []string{}
	var cardList = make(chan []string, 512)
	var cardPaths []string

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
		} else if strings.Contains(cardName, "card") == false || devType == "drm_connector" {
			pecho("debug", "Udev returned " + cardName + ", which is not a GPU")
			continue
		}
		totalGpus = append(
			totalGpus,
			cardName,
		)
		cardPaths = append(
			cardPaths,
			card.Syspath(),
		)
	}
	wg.Wait()

	switch len(totalGpus) {
		case 0:
			pecho("warn", "Found no GPU")
		default:
			if config.System.GameMode {
				wg.Go(func() {
					setOffloadEnvs()
				})
				for _, cardName := range totalGpus {
					card := cardName
					wg.Go(func() {
						bindCard(card, argChan, config)
					})
				}
			} else {
				for idx, cardName := range totalGpus {
					wg.Add(1)
					go func (idx int, card string) {
						defer wg.Done()
						cardPath := cardPaths[idx]
						detectCardStatus(cardList, cardPath, card)
					} (idx, cardName)
				}
				go func () {
					wg.Wait()
					close (cardList)
				} ()
				for card := range cardList {
					activeGpus = append(
						activeGpus,
						card...,
					)
				}

				for _, cardName := range activeGpus {
					wg.Add(1)
					go func (card string) {
						defer wg.Done()
						bindCard(card, argChan, config)
					} (cardName)
				}
			}
	}

	// TODO: Drop the debug output below
	//pecho("debug", "Generated GPU bind parameters:", gpuArg)
	pecho(
	"debug",
	"Total GPU count", len(totalGpus), "with active count:", activeGpus)
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