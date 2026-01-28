package main

import (
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

const (
	version		float32	=	0.1
	controlFile	string	=	"instanceId=inIdHold\nappID=idHold\nbusDir=busHold\nbusDirAy=busAyHold\nfriendlyName=friendlyHold"
)

type runtimeOpts struct {
	action		bool
	fullCmdline	string
	quit		int8 // 1 for normal, 2 for external, 3 for forced?
}

type runtimeParms struct {
	flatpakInstanceID	string
}

type XDG_DIRS struct {
	runtimeDir		string
	confDir			string
	cacheDir		string
	dataDir			string
	home			string
}

type portableConfigOpts struct {
	confPath		string
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
	dbusWake		bool
	mountInfo		bool
}

var (
	internalLoggingLevel	int
	confOpts		portableConfigOpts
	runtimeInfo		runtimeParms
	xdgDir			XDG_DIRS
	runtimeOpt		runtimeOpts
)

func pecho(level string, message string) {
	switch level {
		case "debug":
			if internalLoggingLevel <= 1 {
				fmt.Println("[Debug] ", message)
			}
		case "info":
			if internalLoggingLevel <= 2 {
				fmt.Println("[Info] ", message)
			}
		case "warn":
			fmt.Println("[Warn] ", message)
		case "crit":
			fmt.Println("[Critical] ", message)
			panic("A critical error has happened")
		default:
			fmt.Println("[Undefined] ", message)
	}
}

func cmdlineDispatcher(cmdChan chan int) {
	runtimeOpt.fullCmdline = strings.Join(os.Args, ", ")
	cmdlineArray := os.Args
	for index, value := range cmdlineArray {
		if runtimeOpt.action == true {
			runtimeOpt.action = false
			continue
		}
		if value == "--actions" {
			runtimeOpt.action = true
			if cmdlineArray[index + 1] == "quit" {
				runtimeOpt.quit = 2
				pecho("debug", "Received quit request from user")
			}
		}
	}
	pecho("debug", "Full command line: " + runtimeOpt.fullCmdline)
	cmdChan <- 1
}

func getVariables(varChan chan int) {
	var externalLoggingLevel = os.Getenv("PORTABLE_LOGGING")
	switch externalLoggingLevel {
		case "debug":
			internalLoggingLevel = 1
		case "info":
			internalLoggingLevel = 2
		case "warn":
			internalLoggingLevel = 3
		default:
			internalLoggingLevel = 3
	}
	varChan <- 1
}

func isPathSuitableForConf(path string) (result bool) {
	confInfo, confReadErr := os.Stat(path)
	if confReadErr != nil {
		pecho("debug", "Unable to pick configuration at " + path + " for reason: " + confReadErr.Error())
	} else {
		if confInfo.IsDir() == true {
			pecho("debug", "Unable to pick configuration at " + path + " for reason: " + "is a directory")
		}
		pecho("debug", "Using configuration from " + path)
		result = true
		return
	}
	result = false
	return
}

func determineConfPath() {
	currentWd, wdErr := os.Getwd()
	var portableConfigRaw string = os.Getenv("_portableConfig")
	var portableConfigLegacyRaw string = os.Getenv("_portalConfig")
	if len(portableConfigLegacyRaw) > 0 {
		pecho("warn", "Using legacy configuration variable!")
		portableConfigRaw = portableConfigLegacyRaw
	}
	if len(portableConfigRaw) == 0 {
		pecho("crit", "_portableConfig undefined")
	}
	if isPathSuitableForConf(portableConfigRaw) == true {
		confOpts.confPath = portableConfigRaw
		return
	} else if
	isPathSuitableForConf("/usr/lib/portable/info" + portableConfigRaw + "/config") == true {
		confOpts.confPath = "/usr/lib/portable/info" + portableConfigRaw + "/config"
		return
	} else if wdErr == nil {
		if isPathSuitableForConf(currentWd + portableConfigRaw) == true {
			confOpts.confPath = currentWd + portableConfigRaw
			return
		}
	} else if wdErr != nil {
		pecho("warn", "Unable to get working directory: " + wdErr.Error())
	}
	pecho("crit", "Unable to determine configuration location")
}

func tryUnquote(input string) (output string) {
	if len(input) == 0 {
		return
	}
	outputU, err := strconv.Unquote(input)
	if err != nil {
		pecho("debug", "Unable to unquote string: " + input + " : " + err.Error())
		output = input
		return
	}
	output = outputU
	return
}

func tryProcessConf(input string, trimObj string) (output string) {
	var outputTrimmed string = strings.TrimPrefix(input, trimObj + "=")
	output = tryUnquote(outputTrimmed)
	return
}

func readConf(readConfChan chan int) {
	determineConfPath()

	confReader, readErr := os.ReadFile(confOpts.confPath)
	if readErr != nil {
		pecho("crit", "Could not read configuration file: " + readErr.Error())
	}

	appID, appIDReadErr := regexp.Compile("appID=.*")
	if appIDReadErr == nil {
		confOpts.appID = tryProcessConf(string(appID.Find(confReader)), "appID")
		pecho("debug", "Determined appID: " + confOpts.appID)
	} else {
		pecho("crit", "Unable to parse appID: " + appIDReadErr.Error())
	}

	friendlyName, friendlyNameReadErr := regexp.Compile("friendlyName=.*")
	if friendlyNameReadErr == nil {
		confOpts.friendlyName = tryProcessConf(string(friendlyName.Find(confReader)), "friendlyName")
		pecho("debug", "Determined friendlyName: " + confOpts.friendlyName)
	} else {
		pecho("crit", "Unable to parse friendlyName: " + friendlyNameReadErr.Error())
	}

	stateDirectory, stateDirectoryReadErr := regexp.Compile("stateDirectory=.*")
	if stateDirectoryReadErr == nil {
		confOpts.stateDirectory = tryProcessConf(string(stateDirectory.Find(confReader)), "stateDirectory")
		pecho("debug", "Determined stateDirectory: " + confOpts.stateDirectory)
	} else {
		pecho("crit", "Unable to parse stateDirectory: " + stateDirectoryReadErr.Error())
	}

	mprisName, mprisNameReadErr := regexp.Compile("mprisName=.*")
	if mprisNameReadErr == nil {
		confOpts.mprisName = tryProcessConf(string(mprisName.Find(confReader)), "mprisName")
		pecho("debug", "Determined mprisName: " + confOpts.mprisName)
	} else {
		pecho("crit", "Unable to parse mprisName: " + mprisNameReadErr.Error())
	}

	launchTarget, launchTargetReadErr := regexp.Compile("launchTarget=.*")
	if launchTargetReadErr == nil {
		confOpts.launchTarget = tryProcessConf(string(launchTarget.Find(confReader)), "launchTarget")
		if len(confOpts.launchTarget) == 0 {
			if len(os.Getenv("launchTarget")) > 0 {
				pecho("warn", "Assigning launchTarget using environment variable, this is not recommended")
			} else {
				pecho("crit", "Unable to determine launchTarget")
			}
		}
		pecho("debug", "Determined launchTarget: " + strconv.Quote(confOpts.launchTarget))
	} else {
		pecho("crit", "Unable to parse launchTarget: " + launchTargetReadErr.Error())
	}

	busLaunchTarget, busLaunchTargetReadErr := regexp.Compile("busLaunchTarget=.*")
	if busLaunchTargetReadErr == nil {
		confOpts.busLaunchTarget = tryProcessConf(string(busLaunchTarget.Find(confReader)), "busLaunchTarget")
		if len(confOpts.busLaunchTarget) == 0 {
			if len(os.Getenv("busLaunchTarget")) > 0 {
				pecho("warn", "Assigning busLaunchTarget using environment variable, this is not recommended")
			} else {
				pecho("info", "busLaunchTarget not set")
			}
		}
		pecho("debug", "Determined busLaunchTarget: " + strconv.Quote(confOpts.launchTarget))
	} else {
		pecho("crit", "Unable to parse busLaunchTarget: " + launchTargetReadErr.Error())
	}

	waylandOnly, waylandOnlyReadErr := regexp.Compile("waylandOnly=.*")
	if waylandOnlyReadErr != nil {
		pecho("crit", "Unable to parse waylandOnly: " + waylandOnlyReadErr.Error())
	}
	var waylandOnlyRaw string = tryProcessConf(string(waylandOnly.Find(confReader)), "waylandOnly")
	switch waylandOnlyRaw {
		case "true":
			confOpts.waylandOnly = true
		case "false":
			confOpts.waylandOnly = false
		case "adaptive":
			if os.Getenv("XDG_SESSION_TYPE") == "wayland" {
				confOpts.waylandOnly = true
			}
		default:
			if os.Getenv("XDG_SESSION_TYPE") == "wayland" {
				confOpts.waylandOnly = true
			}
	}
	pecho("debug", "Determined waylandOnly: " + strconv.FormatBool(confOpts.waylandOnly))

	bindNetwork, bindNetworkReadErr := regexp.Compile("bindNetwork=.*")
	if bindNetworkReadErr != nil {
		pecho("crit", "Unable to parse bindNetwork: " + bindNetworkReadErr.Error())
	}
	var bindNetworkRaw string = tryProcessConf(string(bindNetwork.Find(confReader)), "bindNetwork")
	switch bindNetworkRaw {
		case "true":
			confOpts.bindNetwork = true
		case "false":
			confOpts.bindNetwork = false
		default:
			confOpts.bindNetwork = true
	}
	pecho("debug", "Determined bindNetwork: " + strconv.FormatBool(confOpts.bindNetwork))

	terminateImmediately, terminateImmediatelyReadErr := regexp.Compile("terminateImmediately=.*")
	if terminateImmediatelyReadErr != nil {
		pecho("crit", "Unable to parse terminateImmediately: " + terminateImmediatelyReadErr.Error())
	}
	var terminateImmediatelyRaw string = tryProcessConf(string(terminateImmediately.Find(confReader)), "terminateImmediately")
	switch terminateImmediatelyRaw {
		case "true":
			confOpts.terminateImmediately = true
		case "false":
			confOpts.terminateImmediately = false
		default:
			confOpts.terminateImmediately = false
	}
	pecho("debug", "Determined terminateImmediately: " + strconv.FormatBool(confOpts.terminateImmediately))

	useZink, useZinkReadErr := regexp.Compile("useZink=.*")
	if useZinkReadErr != nil {
		pecho("crit", "Unable to parse useZink: " + useZinkReadErr.Error())
	}
	var useZinkRaw string = tryProcessConf(string(useZink.Find(confReader)), "useZink")
	switch useZinkRaw {
		case "true":
			confOpts.useZink = true
		case "false":
			confOpts.useZink = false
		default:
			confOpts.useZink = false
	}
	pecho("debug", "Determined useZink: " + strconv.FormatBool(confOpts.useZink))

	qt5Compat, qt5CompatReadErr := regexp.Compile("qt5Compat=.*")
	if qt5CompatReadErr != nil {
		pecho("crit", "Unable to parse qt5Compat: " + qt5CompatReadErr.Error())
	}
	var qt5CompatRaw string = tryProcessConf(string(qt5Compat.Find(confReader)), "qt5Compat")
	switch qt5CompatRaw {
		case "true":
			confOpts.qt5Compat = true
		case "false":
			confOpts.qt5Compat = false
		default:
			confOpts.qt5Compat = true
	}
	pecho("debug", "Determined qt5Compat: " + strconv.FormatBool(confOpts.qt5Compat))

	allowClassicNotifs := regexp.MustCompile("allowClassicNotifs=.*")
	var allowClassicNotifsRaw string = tryProcessConf(string(allowClassicNotifs.Find(confReader)), "allowClassicNotifs")
	switch allowClassicNotifsRaw {
		case "true":
			confOpts.allowClassicNotifs = true
		case "false":
			confOpts.allowClassicNotifs = false
		default:
			confOpts.allowClassicNotifs = true
	}
	pecho("debug", "Determined allowClassicNotifs: " + strconv.FormatBool(confOpts.allowClassicNotifs))

	gameMode, gameModeReadErr := regexp.Compile("gameMode=.*")
	if gameModeReadErr != nil {
		pecho("crit", "Unable to parse gameMode: " + gameModeReadErr.Error())
	}
	var gameModeRaw string = tryProcessConf(string(gameMode.Find(confReader)), "gameMode")
	switch gameModeRaw {
		case "true":
			confOpts.gameMode = true
		case "false":
			confOpts.gameMode = false
		default:
			confOpts.gameMode = false
	}
	pecho("debug", "Determined gameMode: " + strconv.FormatBool(confOpts.gameMode))

	bindCameras, bindCamerasReadErr := regexp.Compile("bindCameras=.*")
	if bindCamerasReadErr != nil {
		pecho("crit", "Unable to parse bindCameras: " + bindCamerasReadErr.Error())
	}
	var bindCamerasRaw string = tryProcessConf(string(bindCameras.Find(confReader)), "bindCameras")
	switch bindCamerasRaw {
		case "true":
			confOpts.bindCameras = true
		case "false":
			confOpts.bindCameras = false
		default:
			confOpts.bindCameras = false
	}
	pecho("debug", "Determined bindCameras: " + strconv.FormatBool(confOpts.bindCameras))

	bindPipewire, bindPipewireReadErr := regexp.Compile("bindPipewire=.*")
	if bindPipewireReadErr != nil {
		pecho("crit", "Unable to parse bindPipewire: " + bindPipewireReadErr.Error())
	}
	var bindPipewireRaw string = tryProcessConf(string(bindPipewire.Find(confReader)), "bindPipewire")
	switch bindPipewireRaw {
		case "true":
			confOpts.bindPipewire = true
		case "false":
			confOpts.bindPipewire = false
		default:
			confOpts.bindPipewire = false
	}
	pecho("debug", "Determined bindPipewire: " + strconv.FormatBool(confOpts.bindPipewire))

	bindInputDevices, bindInputDevicesReadErr := regexp.Compile("bindInputDevices=.*")
	if bindInputDevicesReadErr != nil {
		pecho("crit", "Unable to parse bindInputDevices: " + bindInputDevicesReadErr.Error())
	}
	var bindInputDevicesRaw string = tryProcessConf(string(bindInputDevices.Find(confReader)), "bindInputDevices")
	switch bindInputDevicesRaw {
		case "true":
			confOpts.bindInputDevices = true
		case "false":
			confOpts.bindInputDevices = false
		default:
			confOpts.bindInputDevices = false
	}
	pecho("debug", "Determined bindInputDevices: " + strconv.FormatBool(confOpts.bindInputDevices))

	allowInhibit, allowInhibitReadErr := regexp.Compile("allowInhibit=.*")
	if allowInhibitReadErr != nil {
		pecho("crit", "Unable to parse allowInhibit: " + allowInhibitReadErr.Error())
	}
	var allowInhibitRaw string = tryProcessConf(string(allowInhibit.Find(confReader)), "allowInhibit")
	switch allowInhibitRaw {
		case "true":
			confOpts.allowInhibit = true
		case "false":
			confOpts.allowInhibit = false
		default:
			confOpts.allowInhibit = false
	}
	pecho("debug", "Determined allowInhibit: " + strconv.FormatBool(confOpts.allowInhibit))

	allowGlobalShortcuts, allowGlobalShortcutsReadErr := regexp.Compile("allowGlobalShortcuts=.*")
	if allowGlobalShortcutsReadErr != nil {
		pecho("crit", "Unable to parse allowGlobalShortcuts: " + allowGlobalShortcutsReadErr.Error())
	}
	var allowGlobalShortcutsRaw string = tryProcessConf(string(allowGlobalShortcuts.Find(confReader)), "allowGlobalShortcuts")
	switch allowGlobalShortcutsRaw {
		case "true":
			confOpts.allowGlobalShortcuts = true
		case "false":
			confOpts.allowGlobalShortcuts = false
		default:
			confOpts.allowGlobalShortcuts = false
	}
	pecho("debug", "Determined allowGlobalShortcuts: " + strconv.FormatBool(confOpts.allowGlobalShortcuts))

	dbusWake, dbusWakeReadErr := regexp.Compile("dbusWake=.*")
	if dbusWakeReadErr != nil {
		pecho("crit", "Unable to parse dbusWake: " + dbusWakeReadErr.Error())
	}
	var dbusWakeRaw string = tryProcessConf(string(dbusWake.Find(confReader)), "dbusWake")
	switch dbusWakeRaw {
		case "true":
			confOpts.dbusWake = true
		case "false":
			confOpts.dbusWake = false
		default:
			confOpts.dbusWake = false
	}
	pecho("debug", "Determined dbusWake: " + strconv.FormatBool(confOpts.dbusWake))

	mountInfo, mountInfoReadErr := regexp.Compile("mountInfo=.*")
	if mountInfoReadErr != nil {
		pecho("crit", "Unable to parse mountInfo: " + mountInfoReadErr.Error())
	}
	var mountInfoRaw string = tryProcessConf(string(mountInfo.Find(confReader)), "mountInfo")
	switch mountInfoRaw {
		case "true":
			confOpts.mountInfo = true
		case "false":
			confOpts.mountInfo = false
		default:
			confOpts.mountInfo = true
	}
	pecho("debug", "Determined mountInfo: " + strconv.FormatBool(confOpts.mountInfo))

	readConfChan <- 1
}

func stopMainAppCompat() {
	stopMainExec := exec.Command("systemctl", "--user", "stop", "app-portable-" + confOpts.friendlyName + ".slice")
	stopMainExec.Stderr = os.Stdout
	stopMainExecErr := stopMainExec.Run()
	if stopMainExecErr != nil {
		pecho("debug", "Stop " + "app-portable-" + confOpts.friendlyName + ".slice" + " failed: " + stopMainExecErr.Error())
	}
}

func stopMainApp() {
	stopMainExec := exec.Command("systemctl", "--user", "stop", "app-portable-" + confOpts.appID + ".service")
	stopMainExec.Stderr = os.Stdout
	stopMainExecErr := stopMainExec.Run()
	if stopMainExecErr != nil {
		pecho("debug", "Stop " + "app-portable-" + confOpts.appID + ".service" + " failed: " + stopMainExecErr.Error())
	}
}

func stopSlice() {
	stopMainExec := exec.Command("systemctl", "--user", "stop", "portable-" + confOpts.friendlyName + ".slice")
	stopMainExec.Stderr = os.Stdout
	stopMainExecErr := stopMainExec.Run()
	if stopMainExecErr != nil {
		pecho("debug", "Stop " + "portable-" + confOpts.friendlyName + ".slice" + " failed: " + stopMainExecErr.Error())
	}
}

func genFlatpakInstanceID(genInfo chan int8) {
	flatpakInfo, err := os.OpenFile("/usr/lib/portable/flatpak-info", os.O_RDONLY, 0600)
	if err != nil {
		pecho("crit", "Failed to read preset Flatpak info")
	}
	var i int
	var instanceIDCleared bool = false
	pecho("debug", "Generating instance ID")
	for i = 0; instanceIDCleared == false; i++ {
		genId, _ := rand.Int(rand.Reader, big.NewInt(9999999999))
		pecho("debug", "Trying instance ID: " + genId.String())
		err := os.Mkdir(xdgDir.runtimeDir + "/.flatpak/" + genId.String(), 0700)
		if err != nil {
			pecho("warn", "Unable to use instance ID " + genId.String())
		} else {
			instanceIDCleared = true
			runtimeInfo.flatpakInstanceID = genId.String()
			break
		}
	}
	os.MkdirAll(xdgDir.runtimeDir + "/portable/" + confOpts.appID, 0700)
	infoObj, ioErr := io.ReadAll(flatpakInfo)
	if ioErr != nil {
		pecho("debug", "Failed to read template Flatpak info for I/O error: " + ioErr.Error())
	}
	stringObj := string(infoObj)
	stringObj = strings.ReplaceAll(stringObj, "placeHolderAppName", confOpts.appID)
	stringObj = strings.ReplaceAll(stringObj, "placeholderInstanceId", runtimeInfo.flatpakInstanceID)
	stringObj = strings.ReplaceAll(stringObj, "placeholderPath", xdgDir.dataDir + "/" + confOpts.stateDirectory)

	os.WriteFile(xdgDir.runtimeDir + "/portable/" + confOpts.appID + "/flatpak-info", []byte(stringObj), 0700)
	os.WriteFile(xdgDir.runtimeDir + "/.flatpak/" + runtimeInfo.flatpakInstanceID + "/info", []byte(stringObj), 0700)

	os.MkdirAll(xdgDir.runtimeDir + "/.flatpak/" + confOpts.appID + "/xdg-run", 0700)
	os.MkdirAll(xdgDir.runtimeDir + "/.flatpak/" + confOpts.appID + "/tmp", 0700)

	var flatpakRef string = ""
	os.WriteFile(xdgDir.runtimeDir + "/.flatpak/" + confOpts.appID + "/.ref", []byte(flatpakRef), 0700)

	var controlContent = controlFile
	controlContent = strings.ReplaceAll(controlContent, "inIdHold", runtimeInfo.flatpakInstanceID)
	controlContent = strings.ReplaceAll(controlContent, "idHold", confOpts.appID)
	controlContent = strings.ReplaceAll(controlContent, "busHold", xdgDir.runtimeDir + "/app/" + confOpts.appID)
	controlContent = strings.ReplaceAll(controlContent, "busAyHold", xdgDir.runtimeDir + "/app/" + confOpts.appID + "-a11y")
	controlContent = strings.ReplaceAll(controlContent, "friendlyHold", confOpts.friendlyName)
	os.WriteFile(xdgDir.runtimeDir + "/portable/" + confOpts.appID + "/control", []byte(controlContent), 0700)

	genInfo <- 1
	flatpakInfo.Close()
}

func getFlatpakInstanceID() {
	if len(runtimeInfo.flatpakInstanceID) > 0 {
		pecho("debug", "Flatpak instance ID already known")
		return
	} else if confOpts.mountInfo == false {
		pecho("debug", "Not getting instance ID because mountInfo is disabled")
		return
	}
	controlFile, readErr := os.ReadFile(xdgDir.runtimeDir + "/portable/" + confOpts.appID + "/control")
	instanceID := regexp.MustCompile("instanceId=.*")
	if readErr == nil {
		var rawInstanceID string = string(instanceID.Find(controlFile))
		runtimeInfo.flatpakInstanceID = tryUnquote(strings.TrimPrefix(rawInstanceID, "instanceId="))
	} else {
		pecho("warn", "Unable to read control file: " + readErr.Error())
	}
	pecho("debug", "Got Flatpak instance ID: " + runtimeInfo.flatpakInstanceID)
}

func cleanDirs() {
	pecho("info", "Cleaning leftovers")
	getFlatpakInstanceID()
	var removeErr error
	if len(runtimeInfo.flatpakInstanceID) > 0 && confOpts.mountInfo == true {
		removeErr = os.RemoveAll(xdgDir.runtimeDir + "/.flatpak/" + confOpts.appID)
		if removeErr != nil {
			pecho("warn", "Unable to remove directory " + xdgDir.runtimeDir + "/.flatpak/" + confOpts.appID + removeErr.Error())
		} else {
			pecho("debug", "Removed directory " + xdgDir.runtimeDir + "/.flatpak/" + confOpts.appID)
		}
		removeErr = os.RemoveAll(xdgDir.runtimeDir + "/.flatpak/" + runtimeInfo.flatpakInstanceID)
		if removeErr != nil {
			pecho("warn", "Unable to remove directory " + xdgDir.runtimeDir + "/.flatpak/" + runtimeInfo.flatpakInstanceID + removeErr.Error())
		} else {
			pecho("debug", "Removed directory " + xdgDir.runtimeDir + "/.flatpak/" + runtimeInfo.flatpakInstanceID)
		}
	} else {
		pecho("debug", "Skipped cleaning Flatpak entries")
	}
	removeErr = os.RemoveAll(xdgDir.runtimeDir + "/app/" + confOpts.appID)
	if removeErr != nil {
		pecho("warn", "Unable to remove directory " + xdgDir.runtimeDir + "/app/" + confOpts.appID + removeErr.Error())
	} else {
		pecho("debug", "Removed directory " + xdgDir.runtimeDir + "/app/" + confOpts.appID)
	}
	removeErr = os.RemoveAll(xdgDir.runtimeDir + "/app/" + confOpts.appID + "-a11y")
	if removeErr != nil {
		pecho("warn", "Unable to remove directory " + xdgDir.runtimeDir + "/app/" + confOpts.appID + "-a11y" + removeErr.Error())
	} else {
		pecho("debug", "Removed directory " + xdgDir.runtimeDir + "/app/" + confOpts.appID + "-a11y")
	}
	removeErr = os.RemoveAll(xdgDir.dataDir + "/applications/" + confOpts.appID + ".desktop")
	if removeErr != nil {
		pecho("warn", "Unable to remove directory " + xdgDir.dataDir + "/applications/" + confOpts.appID + ".desktop" + removeErr.Error())
	} else {
		pecho("debug", "Removed directory " + xdgDir.dataDir + "/applications/" + confOpts.appID + ".desktop")
	}
}

func stopApp(operation string) {
	go stopMainApp()
	go stopMainAppCompat()
	go stopSlice()
	cleanDirs()
	switch operation {
		case "normal":
			pecho("debug", "Selected stop mode: normal")
		default:
			pecho("crit", "Unknown operation for stopApp: " + operation)
	}
}

func lookUpXDG(xdgChan chan int) {
	xdgDir.runtimeDir = os.Getenv("XDG_RUNTIME_DIR")
	if len(xdgDir.runtimeDir) == 0 {
		pecho("warn", "XDG_RUNTIME_DIR not set")
	} else {
		var runtimeDebugMsg string = "XDG_RUNTIME_DIR set to: " + xdgDir.runtimeDir
		pecho("debug", runtimeDebugMsg)
		runtimeDirInfo, errRuntimeDir := os.Stat(xdgDir.runtimeDir)
		var errRuntimeDirPrinted string = "Could not determine the status of XDG Runtime Directory "
		if errRuntimeDir != nil {
			println(errRuntimeDir)
			pecho("crit", errRuntimeDirPrinted)
		}
		if runtimeDirInfo.IsDir() == false {
			pecho("crit", "XDG_RUNTIME_DIR is not a directory")
		}
	}

	var cacheErr error
	var homeErr error
	var confErr error
	xdgDir.home, homeErr = os.UserHomeDir()
	if homeErr != nil {
		pecho("crit", "Falling back to working directory: " + homeErr.Error())
		xdgDir.home, homeErr = os.Getwd()
		if homeErr != nil {
			pecho("crit", "Unable to use working directory as fallback: " + homeErr.Error())
		}
	} else {
		pecho("debug", "Determined home: " + xdgDir.home)
	}

	xdgDir.cacheDir, cacheErr = os.UserCacheDir()
	if cacheErr != nil {
		xdgDir.cacheDir = xdgDir.home + "/.cache"
		pecho("warn", "Unable to determine cache directory, falling back to " + xdgDir.cacheDir)
	}

	xdgDir.confDir, confErr = os.UserConfigDir()
	if confErr != nil {
		xdgDir.confDir = xdgDir.home + "/.config"
		pecho("warn", "Unable to determine config directory, falling back to " + xdgDir.confDir)
	}

	if len(os.Getenv("XDG_DATA_HOME")) > 0 {
		xdgDir.dataDir = os.Getenv("XDG_DATA_HOME")
		pecho("debug", "User specified data home: " + xdgDir.dataDir)
	} else {
		xdgDir.dataDir = xdgDir.home + "/.local/share"
		pecho("debug", "Using default data home: " + xdgDir.dataDir)
	}

	xdgChan <- 1
}

func pwSecContext(pwChan chan string) {
	if confOpts.bindPipewire == false {
		pwChan <- "noop"
		return
	}
	pwSecCmd := []string{
		"--user",
		"--quiet",
		//"--no-block",
		"-p", "Slice=portable-" + confOpts.friendlyName + ".slice",
		"-u", "app-portable-" + confOpts.appID + "-pipewire-container",
		"-p", "KillMode=control-group",
		"-p", "After=pipewire.service",
		"-p", "Requires=pipewire.service",
		"-p", "Wants=wireplumber.service",
		"-p", "StandardOutput=file:" + xdgDir.runtimeDir + "/portable/" + confOpts.appID + "/pipewire-socket",
		"-p", "SuccessExitStatus=SIGKILL",
		"--",
		"stdbuf",
		"-oL",
		"/usr/bin/pw-container",
		"-P",
		`{ "pipewire.sec.engine": "top.kimiblock.portable", "pipewire.access": "restricted" }`,
	}

	pwSecRun := exec.Command("/usr/bin/systemd-run", pwSecCmd...)
	pwSecRun.Stderr = os.Stderr

	if internalLoggingLevel <= 1 {
		pwSecRun.Stdout = os.Stdout
	}

	var err error
	pecho("debug", "Executing pw-container")
	err = pwSecRun.Run()
	if err != nil {
		pecho("warn", "Failed to start up PipeWire proxy. " + err.Error())
	}

	pwProxyInfo, openErr := os.OpenFile(xdgDir.runtimeDir + "/portable/" + confOpts.appID + "/pipewire-socket", os.O_RDONLY, 0700)
	if openErr != nil {
		pecho(
			"crit",
			"Failed to open PipeWire proxy status: " + openErr.Error(),
			)
	}
	var pwProxySocket string
	for {
		pwInfoObj, ioReadErr := io.ReadAll(pwProxyInfo)
		if ioReadErr != nil {
			pecho("crit", "Failed to read PipeWire proxy status: " + ioReadErr.Error())
		}
		stringObj := string(pwInfoObj)
		if strings.HasPrefix(stringObj, "new socket: ") {
			pwProxySocket = strings.TrimPrefix(stringObj, "new socket: ")
			break
		}
		pecho("debug", "PipeWire proxy has not yet started")
	}

	pwChan <- pwProxySocket
}

func calcDbusArg(argChan chan []string) {
	pecho("debug", "Calculating D-Bus arguments...")
	argList := []string{}
	argList = append(
		argList,
		"--no-block",
		"--user",
		"-p", "Slice=portable-" + confOpts.friendlyName + ".slice",
		"-u", confOpts.friendlyName + "-dbus",
		"-p", "KillMode=control-group",
		"-p", "Wants=xdg-document-portal.service xdg-desktop-portal.service",
		"-p", "After=xdg-document-portal.service xdg-desktop-portal.service",
		"-p", "SuccessExitStatus=SIGKILL",
		"-p", "StandardError=file:" + xdgDir.runtimeDir + "/.flatpak/" + runtimeInfo.flatpakInstanceID + "/bwrapinfo.json",
		"--",
		"bwrap",
		"--json-status-fd", "2",
		"--unshare-all",
		"--symlink", "/usr/lib64", "/lib64",
		"--ro-bind", "/usr/lib", "/usr/lib",
		"--ro-bind", "/usr/lib64", "/usr/lib64",
		"--ro-bind", "/usr/bin", "/usr/bin",
		"--ro-bind-try", "/usr/share", "/usr/share",
		"--bind", xdgDir.runtimeDir, xdgDir.runtimeDir,
		"--ro-bind",
			xdgDir.runtimeDir + "/portable/" + confOpts.appID + "/flatpak-info",
			xdgDir.runtimeDir + "/.flatpak-info",
		"--ro-bind",
			xdgDir.runtimeDir + "/portable/" + confOpts.appID + "/flatpak-info",
			"/.flatpak-info",
		"--",
		"/usr/bin/xdg-dbus-proxy",
		os.Getenv("DBUS_SESSION_BUS_ADDRESS"),
		xdgDir.runtimeDir + "/app/" + confOpts.appID + "/bus",
		"--filter",
		"--own=com.belmoussaoui.ashpd.demo",
		"--talk=org.unifiedpush.Distributor.*",
		"--own=" + confOpts.appID,
		"--own=" + confOpts.appID + ".*",
		"--talk=org.kde.StatusNotifierWatcher",
		"--talk=com.canonical.AppMenu.Registrar",
		"--see=org.a11y.Bus",
		"--call=org.a11y.Bus=org.a11y.Bus.GetAddress@/org/a11y/bus",
		"--call=org.a11y.Bus=org.freedesktop.DBus.Properties.Get@/org/a11y/bus",
		// Screenshot stuff
		"--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Screenshot",
		"--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Screenshot.Screenshot",

		"--see=org.freedesktop.portal.Request",
		"--call=org.freedesktop.portal.Desktop=org.freedesktop.DBus.Properties.GetAll",
		"--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Session.Close",
		"--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Settings.ReadAll",
		"--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Settings.Read",
		"--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Request",
		"--call=org.freedesktop.portal.Desktop=org.freedesktop.DBus.Properties.Get@/org/freedesktop/portal/desktop",
		"--call=org.freedesktop.portal.Request=*",
		"--broadcast=org.freedesktop.portal.*=@/org/freedesktop/portal/*",
	)

	pecho("debug", "Expanding built-in rules")

	allowedPortals := []string{
		"Screenshot",
		"Email",
		"Usb",
		"PowerProfileMonitor",
		"MemoryMonitor",
		"ProxyResolver.Lookup",
		"ScreenCast",
		"Account.GetUserInformation",
		"Camera",
		"RemoteDesktop",
		"Documents",
		"Device",
		"FileChooser",
		"FileTransfer",
		"Notification",
		"Print",
		"NetworkMonitor",
		"OpenURI",
		"Fcitx",
		"IBus",
		"Secret",
		"OpenURI",
	}

	for _, talkDest := range allowedPortals {
		argList = append(
			argList,
			"--call=org.freedesktop.portal.Desktop=org.freedesktop.portal." + talkDest,
			"--call=org.freedesktop.portal.Desktop=org.freedesktop.portal." + talkDest + ".*",
		)
	}

	allowedTalks := []string{
		"org.freedesktop.portal.Documents",
		"org.freedesktop.portal.FileTransfer",
		"org.freedesktop.portal.Notification",
		"org.freedesktop.portal.Print",
		"org.freedesktop.FileManager1",
		"org.freedesktop.portal.Fcitx",
		"org.freedesktop.portal.IBus",
	}

	for _, talkDest := range allowedTalks {
		argList = append(
			argList,
			"--talk=" + talkDest,
			"--call=" + talkDest + "=*",
		)
	}

	if internalLoggingLevel < 1 {
		argList = append(argList, "--log")
	}
	if os.Getenv("XDG_CURRENT_DESKTOP") == "gnome" {
		pecho("debug", "Enabling GNOME exclusive feature: Location")
		argList = append(
			argList,
			"--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Location",
			"--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Location.*",
			)
	}
	os.MkdirAll(xdgDir.runtimeDir + "/doc/by-app/" + confOpts.appID, 0700)

	// Shitty MPRIS calc code
	mprisOwnList := []string{}
	/* Take an app ID top.kimiblock.test for example
		appIDSplit would have 3 substrings
		appIDSepNum would be 3
		so appIDSplit[3 - 1] should be the last part
	*/
	appIDSplit := strings.Split(confOpts.appID, ".")
	appIDSegNum := len(appIDSplit)
	var appIDLastSeg string = appIDSplit[appIDSegNum - 1]
	mprisOwnList = append(
		mprisOwnList,
		"--own=org.mpris.MediaPlayer2." + confOpts.appID,
		"--own=org.mpris.MediaPlayer2." + confOpts.appID + ".*",
		"--own=org.mpris.MediaPlayer2." + appIDLastSeg,
		"--own=org.mpris.MediaPlayer2." + appIDLastSeg + ".*",
	)
	if len(confOpts.mprisName) == 0 {
		pecho("debug", "Using default MPRIS own name")
	} else {
		mprisOwnList = append(
			mprisOwnList,
			"--own=org.mpris.MediaPlayer2." + confOpts.mprisName,
			"--own=org.mpris.MediaPlayer2." + confOpts.mprisName + ".*",
		)
	}

	if confOpts.allowClassicNotifs == true {
		argList = append(
			argList,
			"--talk=org.freedesktop.Notifications",
			"--call=org.freedesktop.Notifications.*=*",
		)
	}

	if confOpts.allowInhibit == true {
		argList = append(
			argList,
			"--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Inhibit",
			"--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Inhibit.*",
		)
	}

	if confOpts.allowGlobalShortcuts == true {
		argList = append(
			argList,
			"--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.GlobalShortcuts",
			"--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.GlobalShortcuts.*",
		)
	}

	argList = append(
		argList,
		mprisOwnList...
	)

	for i := 2; i < 30; i++ {
		argList = append(
			argList,
			"--own=org.kde.StatusNotifierItem-" + strconv.Itoa(i) + "-1",
		)
	}

	pecho("debug", "Calculated D-Bus arguments: " + strings.Join(argList, ", "))
	argChan <- argList
}

func doCleanUnit(dbusChan chan int8) {
	cleanUnits := []string{
		confOpts.friendlyName + "*",
		"app-portable-" + confOpts.appID,
		"app-portable-" + confOpts.appID + "-pipewire-container",
	}
	resetCmd := []string{"--user", "reset-failed"}
	resetCmd = append(
		resetCmd,
		cleanUnits...
	)

	cleanCmd := []string{"--user", "clean"}
	cleanCmd = append(
		cleanCmd,
		cleanUnits...
	)

	killCmd := []string{"--user", "kill"}
	killCmd = append(
		killCmd,
		cleanUnits...
	)

	err := exec.Command("systemctl", killCmd...)
	err.Run()

	err = exec.Command("systemctl", resetCmd...)
	err.Run()

	err = exec.Command("systemctl", cleanCmd...)
	err.Start()
	pecho("debug", "Cleaning ready")

	dbusChan <- 1
}

func startProxy(dbusChan chan int8) {
	argChan := make(chan []string)
	go calcDbusArg(argChan)

	dbusArgs := <- argChan
	pecho("debug", "D-Bus argument ready")
	os.MkdirAll(xdgDir.runtimeDir + "/app/" + confOpts.appID, 0700)
	os.MkdirAll(xdgDir.runtimeDir + "/app/" + confOpts.appID + "-a11y", 0700)
	pecho("info", "Starting D-Bus proxy")

	busExec := exec.Command(
		"systemd-run",
		dbusArgs...
	)
	busExec.Stderr = os.Stderr
	if internalLoggingLevel <= 1 {
		busExec.Stdout = os.Stdout
	}
	busErr := busExec.Run()
	dbusChan <- 1
	if busErr != nil {
		pecho("crit", "D-Bus proxy has failed! " + busErr.Error())
	}
}

func startApp(pwArg string) {
	go forceBackgroundPerm()
	sdExec := exec.Command("xargs", "-0", "-a", xdgDir.runtimeDir + "/portable/" + confOpts.appID + "/bwrapArgs", "systemd-run")
	sdExec.Stderr = os.Stderr
	sdExec.Stdout = os.Stdout
	sdExec.Stdin = os.Stdin
	sdExecErr := sdExec.Run()
	if sdExecErr != nil {
		fmt.Println(sdExecErr)
		pecho("crit", "Unable to start systemd-run")
	}
}

func forceBackgroundPerm() {
	pecho("debug", "Unrestricting background limits")
	dbusSendExec := exec.Command("dbus-send", "--session", "--print-reply", "--dest=org.freedesktop.impl.portal.PermissionStore", "/org/freedesktop/impl/portal/PermissionStore", "org.freedesktop.impl.portal.PermissionStore.SetPermission", "string:background", "boolean:true", "string:background", "string:" + confOpts.appID, "array:string:yes")
	dbusSendExec.Stderr = os.Stderr
	if internalLoggingLevel <= 1 {
		dbusSendExec.Stdout = os.Stdout
	}
	err := dbusSendExec.Run()
	if err != nil {
		pecho("warn", "Failed to set background permission, you apps may be terminated by desktop unexpectly: " + err.Error())
	}
}

func main() {
	fmt.Println("Portable daemon", version, "starting")
	readConfChan := make(chan int)
	go readConf(readConfChan)
	xdgChan := make(chan int)
	go lookUpXDG(xdgChan)
	cmdChan := make(chan int)
	go cmdlineDispatcher(cmdChan)
	varChan := make(chan int)
	go getVariables(varChan)
	getVariablesReady := <- varChan
	readConfReady := <- readConfChan
	cmdReady := <- cmdChan
	xdgReady := <- xdgChan
	if getVariablesReady == 1 && readConfReady == 1 && xdgReady == 1 && cmdReady == 1 {
		pecho("debug", "getVariables, lookupXDG, cmdlineDispatcher and readConf are ready")
	}

	// Warn multi-instance here
	cleanUnitChan := make(chan int8)
	go doCleanUnit(cleanUnitChan)
	pwSecContextChan := make(chan string)
	go pwSecContext(pwSecContextChan)
	genChan := make(chan int8)
	go genFlatpakInstanceID(genChan)
	genReady := <- genChan
	genReady = <- cleanUnitChan
	if genReady == 1 {
		pecho("debug", "Flatpak info and cleaning ready")
	}
	proxyChan := make(chan int8)
	go startProxy(proxyChan)
	ready := <- proxyChan
	if ready == 1 {
		pecho("debug", "Proxy ready")
	}
	pwBwArg := <- pwSecContextChan
	startApp(pwBwArg)
	stopApp("normal")
}