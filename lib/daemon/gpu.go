package main

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/Kraftland/portable/lib/gpu"
	"github.com/Kraftland/portable/lib/portals"
	udev "github.com/jochenvg/go-udev"
)

func detectCardBrand(dev *udev.Device) (gpuBrand, error) {
	var device *udev.Device
	driver := dev.Driver()
	if len(driver) == 0 {
		// Lookup for parent first
		device = dev.Parent()
	} else {
		device = dev
	}

	// TODO: what about AMD?
	switch vendor := device.SysattrValue("vendor"); vendor {
		case "0x8086":
			return "intel", nil
		case "0x10de":
			return "nvidia", nil
		case "0x1002":
			return "amd", nil
		default:
			return "unknown", errors.New("Unrecognised vendor " + strconv.Quote(vendor))
	}
}

func gpuBind(gpuChan chan []string, config Config) {
	var gameModeEnabledChan = make(chan bool, 1)
	var chanWg sync.WaitGroup
	var wg sync.WaitGroup

	go func () {
		if ! config.System.GameMode {
			gameModeEnabledChan <- false
			return
		}
		object := portals.PowerProfileMonitor{}
		lowPower, err := object.PowerSaverEnabled()
		if err != nil {
			pecho("warn", "Could not detect power profiles status:", err)
			gameModeEnabledChan <- true
		}
		switch lowPower {
			case true:
				pecho("warn", "Rejecting gameMode with Low Power Mode")
				gameModeEnabledChan <- false
			default:
				gameModeEnabledChan <- config.System.GameMode
		}
	} ()

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

	var cardsToBind []*udev.Device

	switch gameMode := <- gameModeEnabledChan; gameMode {
		case true:
			var err error
			cardsToBind, err = gpu.ListGraphicsCard()
			if err != nil {
				pecho("warn", "Could not list GPUs:", err)
				return
			}
			wg.Go(func() {
				for idx := range cardsToBind {
					nvDev := cardsToBind[idx].Driver() == "nvidia"
					nvParent := cardsToBind[idx].Parent().Driver() == "nvidia"
					if nvDev || nvParent {
						setOffloadEnvs(true)
						return
					}
				}
				setOffloadEnvs(false)
			})
		case false:
			var err error
			cardsToBind, err = gpu.ListGraphicsCard()
			if err != nil {
				pecho("warn", "Could not list active GPUs:", err)
				return
			}
	}


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
	wg.Wait()

	switch len(cardsToBind) {
		case 0:
			pecho("warn", "Found no GPU to expose, possible headless configuration")
		default:
			for idx := range cardsToBind {
				card := cardsToBind[idx]
				wg.Go(func() {
					bindCard(
						card,
						argChan,
						config,
					)
				})
			}
	}
}

// Set required envs for PRIME render offloading
func setOffloadEnvs(hasNvidia bool) {
	//addEnv("VK_LOADER_DRIVERS_DISABLE=none")
	if hasNvidia {
		addEnv("__NV_PRIME_RENDER_OFFLOAD=1")
		addEnv("__VK_LAYER_NV_optimus=NVIDIA_only")
		//addEnv("__GLX_VENDOR_LIBRARY_NAME=nvidia")
		//addEnv("VK_LOADER_DRIVERS_SELECT=nvidia_icd.json")
	} else {
		addEnv("DRI_PRIME=1")
	}
}

func bindCard(cardDevice *udev.Device, argChanFin chan []string, config Config) {
	var sendWg sync.WaitGroup
	var argComb = make(chan []string, 5)
	sendWg.Go(func() {
		var args []string
		for arg := range argComb {
			args = append(args, arg...)
		}
		pecho("debug", "Generated bwrap argument for", cardDevice.Sysname(), args)
		argChanFin <- args
	})

	var wg sync.WaitGroup

	defer func () {
		wg.Wait()
		close(argComb)
		sendWg.Wait()
	} ()

	// Bind parent device so lutris is happy
	wg.Go(func() {
		if len(cardDevice.Driver()) == 0 {
			parent := cardDevice.Parent()

			// Ensure that the parent device has a driver, and is a PCI device
			if parent.Subsystem() != "pci" {
				return
			}
			if len(parent.Driver()) == 0 {
				return
			}
			pecho("debug", "Binding parent device for GPU:", parent.Syspath())
			argComb <- []string{
				"--dev-bind",
					parent.Syspath(),
					parent.Syspath(),
			}
		}
	})

	wg.Go(func() {
		vendor, err := detectCardBrand(cardDevice)
		if err != nil {
			pecho("warn",
				"Could not detect GPU vendor for device",
				cardDevice.Devpath(),
				":",
				err,
			)
			return
		}
		switch vendor {
			case "amd":
				if _, err := os.Stat("/dev/kfd"); err == nil {
					argComb <- []string{
						"--dev-bind",
						"/dev/kfd",
						"/dev/kfd",
					}
				}
			case "nvidia":
				wg.Go(func() {
					argComb <- tryBindNv()
				})
				wg.Go(func() {
					pecho("debug",
						"Detected NVIDIA device:",
						cardDevice.Devpath(),
					)
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
					wg.Go(func() {
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
			default:
		}
	})

	devNode := cardDevice.Devnode()
	//nodeName := filepath.Base(devNode)
	sysPath := cardDevice.Syspath()
	for k := range cardDevice.Devlinks() {
		argComb <- []string{
			"--symlink",
				sysPath,
				k,
		}
	}
	//cardRoot := strings.TrimSuffix(sysPath, "/drm/" + cardName)
	argComb <- []string{
		"--symlink",
			sysPath,
			"/sys/class/drm/" + cardDevice.Sysname(),
		"--dev-bind",
			devNode,
			devNode,
		"--dev-bind",
			sysPath,
			sysPath,
	}
	cardID := cardDevice.PropertyValue("ID_PATH")
	u := udev.Udev{}
	// Map card* to renderD*
	eR := u.NewEnumerate()
	eR.AddMatchIsInitialized()
	eR.AddMatchSubsystem("drm")
	eR.AddMatchProperty("DEVTYPE", "drm_minor")
	//eR.AddMatchProperty("ID_PATH", cardID)
	devs, err := eR.Devices()
	if err != nil {
		pecho("warn", "Could not query udev for render node:", err)
	}
	var rendererSlice []*udev.Device
	for _, device := range devs {
		nodeName := filepath.Base(device.Devnode())
		if strings.HasPrefix(nodeName, "renderD") && device.PropertyValue("ID_PATH") == cardID {
			rendererSlice = append(rendererSlice, device)
		}
	}
	switch devsCnt := len(rendererSlice); devsCnt {
		case 0:
			pecho("warn", "Could not translate GPU to render node: did not receive any result from udev")
			return
		case 1:
		default:
			pecho("warn", "Udev returned more devices than required:", devsCnt)
	}

	// var renderNodeName string = rendererSlice[0].Sysname()
	var renderDevPath string = rendererSlice[0].Devnode()
	sysPath = rendererSlice[0].Syspath()

	for k := range rendererSlice[0].Devlinks() {
		argComb <- []string{
			"--symlink",
				sysPath,
				k,
		}
	}

	argComb <- []string{
		"--dev-bind",
			renderDevPath,
			renderDevPath,
		"--symlink",
			sysPath,
			"/sys/class/drm/" + rendererSlice[0].Sysname(),
	}
}