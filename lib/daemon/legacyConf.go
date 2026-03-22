package main

import (
	"os"
	"bufio"
	"strings"
	"path/filepath"
)

type portableLegacyConfigOpts struct {
	confPath		string
	networkDeny		string
	friendlyName		string
	appID			string
	stateDirectory		string
	launchTarget		string	// this one may be empty?
	busLaunchTarget		string	// also may be empty
	bindNetwork		bool
	terminateImmediately	bool
	allowClassicNotifs	bool
	useZink			bool
	qt5Compat		bool
	waylandOnly		bool
	gameMode		bool
	mprisName		string // may be empty
	bindCameras		bool
	bindPipewire		bool
	bindInputDevices	bool
	allowInhibit		bool
	allowGlobalShortcuts	bool
	allowKDEStatus		bool
	dbusWake		bool
	mountInfo		bool
}

type confTarget struct {
	str			*string
	b			*bool
}

var (
	legacyConf		portableLegacyConfigOpts
)

// Defaults should be set in readConf()
var targets = map[string]confTarget{
	"appID":		{str: &legacyConf.appID},
	"friendlyName":		{str: &legacyConf.friendlyName},
	"stateDirectory":	{str: &legacyConf.stateDirectory},
	"launchTarget":		{str: &legacyConf.launchTarget},
	"busLaunchTarget":	{str: &legacyConf.busLaunchTarget},
	"mprisName":		{str: &legacyConf.mprisName},
	"bindNetwork":		{b: &legacyConf.bindNetwork},
	"terminateImmediately":	{b: &legacyConf.terminateImmediately},
	"networkDeny":		{str: &legacyConf.networkDeny},
	"allowClassicNotifs":	{b: &legacyConf.allowClassicNotifs},
	"useZink":		{b: &legacyConf.useZink},
	"qt5Compat":		{b: &legacyConf.qt5Compat},
	"waylandOnly":		{b: &legacyConf.waylandOnly},
	"gameMode":		{b: &legacyConf.gameMode},
	"bindCameras":		{b: &legacyConf.bindCameras},
	"bindPipewire":		{b: &legacyConf.bindPipewire},
	"bindInputDevices":	{b: &legacyConf.bindInputDevices},
	"allowInhibit":		{b: &legacyConf.allowInhibit},
	"allowGlobalShortcuts":	{b: &legacyConf.allowGlobalShortcuts},
	"allowKDEStatus":	{b: &legacyConf.allowKDEStatus},
	"dbusWake":		{b: &legacyConf.dbusWake},
	"mountInfo":		{b: &legacyConf.mountInfo},
}

var confInfo = map[string]string{
	"appID":		"string",
	"friendlyName":		"string",
	"stateDirectory":	"string",
	"launchTarget":		"string",
	"busLaunchTarget":	"string",
	"bindNetwork":		"bool",
	"terminateImmediately":	"bool",
	"networkDeny":		"string",
	"allowClassicNotifs":	"bool",
	"useZink":		"bool",
	"qt5Compat":		"bool",
	"waylandOnly":		"bool",
	"gameMode":		"bool",
	"mprisName":		"string",
	"bindCameras":		"bool",
	"bindPipewire":		"bool",
	"bindInputDevices":	"bool",
	"allowInhibit":		"bool",
	"allowGlobalShortcuts":	"bool",
	"allowKDEStatus":	"bool",
	"dbusWake":		"bool",
	"mountInfo":		"bool",
}

func determineLegacyConfPath() string {
	var portableConfigLegacyRaw string
	var portableConfigRaw string
	currentWd, wdErr := os.Getwd()
	portableConfigLegacyRaw = os.Getenv("_portalConfig")
	portableConfigRaw = os.Getenv("_portableConfig")
	if len(portableConfigLegacyRaw) > 0 {
		pecho("warn", "Using legacy configuration variable!")
		portableConfigRaw = portableConfigLegacyRaw
	}
	if len(portableConfigRaw) == 0 {
		pecho("crit", "_portableConfig undefined")
	}
	if isPathSuitableForConf(portableConfigRaw) == true {
		return portableConfigRaw
	}
	if isPathSuitableForConf(filepath.Join(xdgDir.confDir, "/portable/info", portableConfigRaw, "config")) {
		return filepath.Join(xdgDir.confDir, "/portable/info", portableConfigRaw, "config")
	} else if isPathSuitableForConf("/usr/lib/portable/info/" + portableConfigRaw + "/config") == true {
		return filepath.Join("/usr/lib/portable/info/", portableConfigRaw, "/config")
	} else if wdErr == nil {
		if isPathSuitableForConf(currentWd + portableConfigRaw) == true {
			return filepath.Join(currentWd, portableConfigRaw)
		}
	} else if wdErr != nil {
		pecho("warn", "Unable to get working directory: " + wdErr.Error())
	}
	pecho("crit", "Unable to determine configuration location")
	select {}
}

func readLegacyConf() Config {
	path := determineLegacyConfPath()
	config := setDefaultConfOpts()


	confFd, fdErr := os.OpenFile(path, os.O_RDONLY, 0700)
	if fdErr != nil {
		pecho("crit", "Could not open configuration for reading: " + fdErr.Error())
	}
	defer confFd.Close()
	scanner := bufio.NewScanner(confFd)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}
		confSlice := strings.SplitN(line, "=", 2)
		var val string

		if len(confSlice) < 2 {
			pecho("debug", "Using default value for" + confSlice[0])
		} else {
			val = tryUnquote(confSlice[1])
		}
		key := confSlice[0]
		target, ok := targets[key]
		if ! ok {
			pecho("warn", "Unknown option " + confSlice[0])
			continue
		}
		switch confInfo[key] {
			case "string":
				if target.str == nil {
					pecho("warn", "Unknown option: " + key)
					continue
				}
				if len(val) == 0 {
					continue
				}
				*target.str = val
			case "bool":
				if target.b == nil {
					pecho("warn", "Unknown option: " + key)
					continue
				}
				if len(val) == 0 {
					continue
				}
				switch val {
					case "true":
						*target.b = true
					case "false":
						*target.b = false
					default:
						if key == "waylandOnly" {
							if val == "adaptive" {
								continue
							}
						}
						pecho("warn", "Invalid value for boolean option: " + key)
				}
		}
	}
	config.Metadata.AppID = legacyConf.appID
	config.Metadata.FriendlyName = legacyConf.friendlyName
	config.Metadata.StateDirectory = legacyConf.stateDirectory
	if len(legacyConf.launchTarget) > 0 {
		split := strings.Split(legacyConf.launchTarget, " ")
		if len(split) > 1 {
			config.Exec.Arguments = split[1:]
		}
		config.Exec.Target = split[0]
	}
	if len(legacyConf.busLaunchTarget) > 0 {
		split := strings.Split(legacyConf.busLaunchTarget, " ")
		if len(split) > 1 {
			config.BusActivation.Arguments = split[1:]
		}
		config.BusActivation.Target = split[0]
	}

	// Terminate immediately is not defined

	if legacyConf.gameMode {
		config.System.GameMode = true
	}
	if legacyConf.allowGlobalShortcuts {
		config.System.GlobalShortcuts = true
	}
	if legacyConf.allowInhibit {
		config.System.InhibitSuspend = true
	}
	if legacyConf.bindNetwork {
		config.Network.Enable = true
		if len(legacyConf.networkDeny) > 0 {
			sp := strings.Split(legacyConf.networkDeny, " ")
			config.Network.FilterDest = sp
			config.Network.Filter = true
		}
	}
	if legacyConf.allowClassicNotifs == false {
		config.Privacy.ClassicNotifications = false
	}
	if legacyConf.bindCameras {
		config.Privacy.Cameras = true
	}
	if legacyConf.bindPipewire {
		config.Privacy.PipeWire = true
	}
	if legacyConf.bindInputDevices {
		config.Privacy.Input = true
	}
	if legacyConf.useZink {
		config.Advanced.Zink = true
	}
	if legacyConf.qt5Compat == false {
		config.Advanced.Qt5Compat = false
	}
	if len(legacyConf.mprisName) > 0 {
		config.Advanced.MprisName = append(config.Advanced.MprisName, legacyConf.mprisName)
	}
	if legacyConf.dbusWake {
		config.Advanced.TrayWake = true
	}
	if legacyConf.allowKDEStatus == false {
		config.Advanced.KDEStatus = false
	}
	if legacyConf.mountInfo == false {
		config.Advanced.FlatpakInfo = false
	}

	return config
}