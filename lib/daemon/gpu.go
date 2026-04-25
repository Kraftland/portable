package main

import (
	"sync"
	udev "github.com/jochenvg/go-udev"
	"strings"
	"os"
	"io"
)

func bindCard(cardName string, argChan chan []string, config Config) {
	var wg sync.WaitGroup
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
	}

	var devProc bool = false
	for _, dev := range devs {
		if devProc == true {
			pecho("warn", "bindCard found more than one candidates")
			continue
		}
		devNode := dev.Devnode()
		sysPath := dev.Syspath()
		cardRoot = strings.TrimSuffix(sysPath, "/drm/" + cardName)
		argChan <- []string{
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
		cardID = dev.PropertyValue("ID_PATH")
		pecho("debug", "Got ID_PATH: " + cardID)
		devProc = true
	}

	// Detect NVIDIA now, because they do not expose ID_VENDOR properly
	wg.Add(1)
	go func (arg chan []string) {
		defer wg.Done()
		cardVendorFd, openErr := os.OpenFile(cardRoot + "/vendor", os.O_RDONLY, 0700)
		if openErr != nil {
			pecho("warn", "Failed to open GPU vendor info " + openErr.Error())
			return
		} else {
			defer cardVendorFd.Close()
		}
		cardVendor, err := io.ReadAll(cardVendorFd)
		if err != nil {
			pecho("warn", "Failed to parse GPU vendor: " + err.Error())
		}
		if strings.Contains(string(cardVendor), "0x10de") == true {
			pecho("debug", "Found NVIDIA device")
			if config.Advanced.Zink {
				addEnv("__GLX_VENDOR_LIBRARY_NAME=mesa")
				addEnv("MESA_LOADER_DRIVER_OVERRIDE=zink")
				addEnv("GALLIUM_DRIVER=zink")
				addEnv("LIBGL_KOPPER_DRI2=1")
				addEnv("__EGL_VENDOR_LIBRARY_FILENAMES=/usr/share/glvnd/egl_vendor.d/50_mesa.json")
			}
			arg <- tryBindNv()
			for _, path := range nvKernelModulePath {
				stat, err := os.Stat(path)
				if err == nil && stat.IsDir() {
					arg <- []string{
						"--ro-bind",
						path, path,
					}
				} else {
					pecho("debug", "Skipping non-existent path: " + path)
					continue
				}
			}
		}
	} (argChan)


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
	devProc = false
	var renderNodeName string
	var renderDevPath string
	for _, dev := range devs {
		if strings.Contains(dev.Sysname(), "card") {
			continue
		} else if devProc == true {
			pecho(
				"warn",
				"Mapping card to renderer: surplus device ID: " + dev.PropertyValue("ID_PATH") + ", sysname: " + dev.Sysname(),
				)
			continue
		} else if dev.PropertyValue("ID_PATH") != cardID {
			pecho("debug", "Udev returned unknown card to us! ID: " + dev.PropertyValue("ID_PATH"))
			continue
		}
		renderNodeName = dev.Sysname()
		pecho("debug", "Got sysname: " + renderNodeName + ", with ID: " + dev.PropertyValue("ID_PATH"))
		renderDevPath = dev.Devnode()
		devProc = true
	}

	argChan <- []string{
		"--dev-bind",
			renderDevPath,
			renderDevPath,
		"--dev-bind",
			"/sys/class/drm/" + renderNodeName,
			"/sys/class/drm/" + renderNodeName,
	}

	wg.Wait()
}