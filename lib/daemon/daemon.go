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
	"bufio"
	"github.com/KarpelesLab/reflink"
	"k8s.io/utils/inotify"
)

const (
	version		float32	=	0.1
	controlFile	string	=	"instanceId=inIdHold\nappID=idHold\nbusDir=busHold\nbusDirAy=busAyHold\nfriendlyName=friendlyHold"
)

type RUNTIME_OPT struct {
	action		bool
	fullCmdline	string
	quit		int8 // 1 for normal, 2 for external, 3 for forced?
}

type RUNTIME_PARAMS struct {
	flatpakInstanceID	string
	waylandDisplay		string
	bwCmd			[]string
	sdEnvParm		[]string
	startAbort		bool
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
	runtimeInfo		RUNTIME_PARAMS
	xdgDir			XDG_DIRS
	runtimeOpt		RUNTIME_OPT
	envsChan		= make(chan string, 100)
	envsFlushReady		= make(chan int8, 1)
	envRegex		= regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)
	startAct		string
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
			//stopApp("normal")
			panic("A critical error has happened")
		default:
			fmt.Println("[Undefined] ", message)
	}
}

func addEnv(envToAdd string) {
	envsChan <- envToAdd
}

func cmdlineDispatcher(cmdChan chan int) {
	runtimeOpt.fullCmdline = strings.Join(os.Args, ", ")
	cmdlineArray := os.Args
	for index, value := range cmdlineArray {
		if runtimeOpt.action == true {
			runtimeOpt.action = false
			continue
		}
		switch value {
			case "--actions" :
			runtimeOpt.action = true
			switch cmdlineArray[index + 1] {
				case "quit":
					runtimeOpt.quit = 2
					stopApp("normal")
					pecho("debug", "Received quit request from user")
				case "debug-shell":
					addEnv("_portableDebug=1")
				case "share-file":
					startAct = "abort"
					shareFile()
				case "share-files":
					startAct = "abort"
					shareFile()
			}
			case "--dbus-activation":
				addEnv("_portableBusActivate=1")
		}
	}
	pecho("debug", "Full command line: " + runtimeOpt.fullCmdline)
	cmdChan <- 1
}

func shareFile() {
	var paths []string
	zenityCmd := exec.Command("/usr/bin/zenity", "--file-selection", "--multiple")
	zenityCmd.Stderr = os.Stderr
	zenityOut, err := zenityCmd.StdoutPipe()
	zenityCmd.Start()
	if err != nil {
		pecho("crit", "Unable to pipe zenity's output" + err.Error())
	}
	scanner := bufio.NewScanner(zenityOut)
	for scanner.Scan() {
		text := scanner.Text()
		paths = append(
			paths,
			strings.Split(text, "|")...
		)
	}
	if len(paths) == 0 {
		pecho("warn", "Did not get any path from zenity")
		os.Exit(2)
	} else {
		pecho("debug", "Got paths from zenity: " + strings.Join(paths, ", "))
	}
	for _, path := range paths {
		// stdlib doesn't seem to do reflink
		pathSp := strings.Split(path, "/")
		pathslices := len(pathSp)
		basename := pathSp[pathslices - 1]
		reflinkErr := reflink.Auto(path, xdgDir.dataDir + "/" + confOpts.stateDirectory + "/Shared/" + basename)
		if reflinkErr != nil {
			pecho("crit", "I/O error copying shared file: " + reflinkErr.Error())
		}
	}
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
	pecho("debug", "Trying configuration: " + path)
	confInfo, confReadErr := os.Stat(path)
	if confReadErr != nil {
		pecho("debug", "Unable to pick configuration at " + path + " for reason: " + confReadErr.Error())
	} else {
		if confInfo.IsDir() == true {
			pecho("debug", "Unable to pick configuration at " + path + " for reason: " + "is a directory")
			result = false
			return
		}
		pecho("debug", "Using configuration from " + path)
		result = true
		return
	}
	result = false
	return
}

func determineConfPath() {
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
		confOpts.confPath = portableConfigRaw
		return
	}
	if isPathSuitableForConf("/usr/lib/portable/info/" + portableConfigRaw + "/config") == true {
		confOpts.confPath = "/usr/lib/portable/info/" + portableConfigRaw + "/config"
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
	cleanChan := make(chan int8, 1)
	go doCleanUnit(cleanChan)
	<- cleanChan
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
			genInfo <- 1
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
	removeErr = os.RemoveAll(xdgDir.runtimeDir + "/portable/" + confOpts.appID)
	if removeErr != nil {
		pecho("warn", "Unable to remove directory " + xdgDir.runtimeDir + "/portable/" + confOpts.appID + removeErr.Error())
	} else {
		pecho("debug", "Removed directory " + xdgDir.runtimeDir + "/portable/" + confOpts.appID)
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
	os.Exit(0)
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

func pwSecContext(pwChan chan []string) {
	var pwSecArg = []string{}
	var pwProxySocket string
	if confOpts.bindPipewire == false {
		pwChan <- pwSecArg
		return
	}
	pwSecCmd := []string{
		"--user",
		"--quiet",
		"--pipe",
		"-p", "Slice=portable-" + confOpts.friendlyName + ".slice",
		"-u", "app-portable-" + confOpts.appID + "-pipewire-container",
		"-p", "KillMode=control-group",
		"-p", "After=pipewire.service",
		"-p", "Requires=pipewire.service",
		"-p", "Wants=wireplumber.service",
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
	stdout, pipeErr := pwSecRun.StdoutPipe()

	scanner := bufio.NewScanner(stdout)
	err := pwSecRun.Start()
	if err != nil {
		pecho("warn", "Failed to start up PipeWire proxy: " + err.Error() + pipeErr.Error())
	}
	for scanner.Scan() {
		stringObj := scanner.Text()
		if strings.HasPrefix(stringObj, "new socket: ") {
			pwProxySocket = strings.TrimPrefix(stringObj, "new socket: ")
			pecho("debug", "Got PipeWire socket: " + pwProxySocket)
			break
		}
	}
	pwSecArg = append(
		pwSecArg,
		"--bind", pwProxySocket, pwProxySocket,
	)
	pwChan <- pwSecArg
	pecho("debug", "pw-container available at " + pwProxySocket)
}

func calcDbusArg(argChan chan []string) {
	pecho("debug", "Calculating D-Bus arguments...")
	argList := []string{}
	argList = append(
		argList,
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
	os.RemoveAll(xdgDir.runtimeDir + "/portable/" + confOpts.appID + "/pipewire-socket")
	cleanUnits := []string{
		confOpts.friendlyName + "*",
		"app-portable-" + confOpts.appID + "*",
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
	err.Stderr = os.Stderr
	err.Run()

	err = exec.Command("systemctl", resetCmd...)
	err.Stderr = os.Stderr
	err.Run()

	err = exec.Command("systemctl", cleanCmd...)
	err.Stderr = os.Stderr
	err.Start()
	pecho("debug", "Cleaning ready")

	dbusChan <- 1
}

func startProxy(dbusChan chan int8) {
	argChan := make(chan []string, 1)
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

func watchForTerminate() {
	openFd, err := os.OpenFile(
		xdgDir.runtimeDir + "/portable/" + confOpts.appID + "/startSignal",
		os.O_RDONLY,
		0700)
	if err != nil {
		pecho("crit", "Unable to open signal file: " + err.Error())
	}
	watcher, errW := inotify.NewWatcher()
	if errW != nil {
		pecho("crit", "Failed to watch signal: " + errW.Error())
	}
	for {
		select {
			case ev := <- watcher.Event:
				pecho("debug", "Got event: " + strconv.Itoa(int(ev.Mask)))
				sigF, sigErr := io.ReadAll(openFd)
				if sigErr != nil {
					pecho("crit", "Unable to read event: " + sigErr.Error())
				}
				if strings.TrimSuffix(string(sigF), "\n") == "terminate-now" {
					stopApp("normal")
					os.Exit(0)
				}
		}
	}
}

func startApp() {
	go forceBackgroundPerm()
	pecho("debug", "Calculated arguments for systemd-run: " + strings.Join(runtimeInfo.bwCmd, ", "))
	sdExec := exec.Command("systemd-run", runtimeInfo.bwCmd...)
	sdExec.Stderr = os.Stderr
	sdExec.Stdout = os.Stdout
	sdExec.Stdin = os.Stdin
	<- envsFlushReady
	go watchForTerminate()
	if startAct == "abort" {
		os.Exit(0)
	}
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

func waylandDisplay(wdChan chan []string) () {
	sessionType := os.Getenv("XDG_SESSION_TYPE")
	switch sessionType {
		case "x11":
			pecho("warn", "Running on X11, this is insecure")
			runtimeInfo.waylandDisplay = "/dev/null"
			return
		case "wayland":
			pecho("debug", "Running under Wayland")
		default:
			pecho("warn", "Unknown XDG_SESSION_TYPE, treating as wayland")
	}

	socketInfo := os.Getenv("WAYLAND_DISPLAY")
	if len(socketInfo) == 0 {
		pecho("debug", "WAYLAND_DISPLAY unset, trying default")
		_, err := os.Stat(xdgDir.runtimeDir + "/wayland-0")
		if err != nil {
			pecho("crit", "Unable to stat Wayland socket: " + err.Error())
		}
		runtimeInfo.waylandDisplay = xdgDir.runtimeDir + "/wayland-0"
		pecho("debug", "Found Wayland socket: " + runtimeInfo.waylandDisplay)
	} else {
		_, err := os.Stat(xdgDir.runtimeDir + "/" + socketInfo)
		if err != nil {
			pecho(
			"info",
			"Unable to find Wayland socket using relative path under XDG_RUNTIME_DIR: " + err.Error(),
			)
			_, absErr := os.Stat(socketInfo)
			if absErr != nil {
				pecho("crit", "Unable to find Wayland socket: " + absErr.Error())
			} else {
				runtimeInfo.waylandDisplay = socketInfo
				pecho("debug", "Found Wayland socket: " + runtimeInfo.waylandDisplay)
			}
		} else {
			runtimeInfo.waylandDisplay = xdgDir.runtimeDir + "/" + socketInfo
			pecho("debug", "Found Wayland socket: " + runtimeInfo.waylandDisplay)
		}
	}
	var waylandArgs = []string{
		"--ro-bind",
			runtimeInfo.waylandDisplay,
			xdgDir.runtimeDir + "/wayland-0",
	}
	wdChan <- waylandArgs
}

func instDesktopFile(instDesktopChan chan int8) {
	_, err := os.Stat("/usr/share/applications/" + confOpts.appID + ".desktop")
	if err == nil {
		pecho("debug", ".desktop file detected")
	} else {
		const templateDesktopFile string = "[Desktop Entry]\nName=placeholderName\nExec=env _portableConfig=placeholderConfig portable\nTerminal=false\nType=Application\nIcon=image-missing\nComment=Application info missing\n"
		var desktopFile string
		desktopFile = templateDesktopFile
		strings.ReplaceAll(desktopFile, "placeholderName", confOpts.appID)
		strings.ReplaceAll(desktopFile, "placeholderConfig", confOpts.confPath)
		os.WriteFile(
			xdgDir.dataDir + "/applications/" + confOpts.appID + ".desktop",
			[]byte(desktopFile),
			0700,
		)
		pecho("debug", "Done installing stub file")
		pecho("warn", "You should supply your own .desktop file")
	}

	instDesktopChan <- 1
}

func setXDGEnvs (xdgEnvReady chan int8) {
	addEnv("XDG_CONFIG_HOME=" + translatePath(xdgDir.confDir))
	addEnv("XDG_DOCUMENTS_DIR=" + xdgDir.dataDir + "/" + confOpts.stateDirectory + "/Documents")
	addEnv("XDG_DATA_HOME=" + xdgDir.dataDir + "/" + confOpts.stateDirectory + "/.local/share")
	addEnv("XDG_STATE_HOME=" + xdgDir.dataDir + "/" + confOpts.stateDirectory + "/.local/state")
	addEnv("XDG_CACHE_HOME=" + xdgDir.dataDir + "/" + confOpts.stateDirectory + "/cache")
	addEnv("XDG_DESKTOP_DIR=" + xdgDir.dataDir + "/" + confOpts.stateDirectory + "/Desktop")
	addEnv("XDG_DOWNLOAD_DIR=" + xdgDir.dataDir + "/" + confOpts.stateDirectory + "/Downloads")
	addEnv("XDG_TEMPLATES_DIR=" + xdgDir.dataDir + "/" + confOpts.stateDirectory + "/Templates")
	addEnv("XDG_PUBLICSHARE_DIR=" + xdgDir.dataDir + "/" + confOpts.stateDirectory + "/Public")
	addEnv("XDG_MUSIC_DIR=" + xdgDir.dataDir + "/" + confOpts.stateDirectory + "/Music")
	addEnv("XDG_PICTURES_DIR=" + xdgDir.dataDir + "/" + confOpts.stateDirectory + "/Pictures")
	addEnv("XDG_VIDEOS_DIR=" + xdgDir.dataDir + "/" + confOpts.stateDirectory + "/Videos")
	xdgEnvReady <- 1
}

func imEnvs (imReady chan int8) {
	addEnv("IBUS_USE_PORTAL=1")
	var imKind string
	if confOpts.waylandOnly == true {
		addEnv("QT_IM_MODULE=wayland")
		addEnv("GTK_IM_MODULE=wayland")
	} else {
		var envToCheck = []string{
			"XMODIFIERS",
			"QT_IM_MODULE",
			"GTK_IM_MODULE",
		}
		for _, env := range envToCheck {
			if strings.Contains(os.Getenv(env), "fcitx") == true {
				imKind = "Fcitx 5"
				break
			} else if strings.Contains(os.Getenv(env), "ibus") == true {
				imKind = "iBus"
				break
			} else if strings.Contains(os.Getenv(env), "gcin") == true {
				imKind = "gcin"
				break
			}
		}
		pecho("debug", "Determined input method type: " + imKind)
		switch imKind {
			case "Fcitx 5":
				addEnv("GTK_IM_MODULE=fcitx")
				addEnv("QT_IM_MODULE=fcitx")
			case "iBus":
				addEnv("QT_IM_MODULE=ibus")
				addEnv("GTK_IM_MODULE=ibus")
			case "gcin":
				addEnv("QT_IM_MODULE=ibus")
				addEnv("GTK_IM_MODULE=gcin")
			default:
				pecho("warn", "Could not determine IM via environment variables")
				procEntries, err := os.ReadDir("/proc")
				if err != nil {
					pecho(
					"warn",
					"Could not determine input method via process lookup: " + err.Error(),
					)
				} else {
					for _, pid := range procEntries {
						if _, err := strconv.Atoi(pid.Name()); err != nil {
							continue
						}
						commFd, fdErr := os.OpenFile(
							"/proc/" + pid.Name() + "/comm",
							os.O_RDONLY,
							0644,
						)
						if fdErr == nil {
							commContent, rdErr := io.ReadAll(commFd)
							if rdErr == nil {
								stringObj := string(commContent)
								if strings.Contains(stringObj, "fcitx") {
									pecho("debug", "Guessing IM: Fcitx")
									addEnv("GTK_IM_MODULE=fcitx")
									addEnv("QT_IM_MODULE=fcitx")
									break
								} else if strings.Contains(stringObj, "ibus") {
									pecho("debug", "Guessing IM: iBus")
									addEnv("QT_IM_MODULE=ibus")
									addEnv("GTK_IM_MODULE=ibus")
									break
								}
							}
						}
					}
				}
		}
	}
	imReady <- 1
}

func setupSharedDir (shareDir chan int8) {
	os.MkdirAll(xdgDir.dataDir + "/" + confOpts.stateDirectory + "/Shared", 0700)
	os.Link(
		xdgDir.dataDir + "/" + confOpts.stateDirectory + "/Shared",
		xdgDir.dataDir + "/" + confOpts.stateDirectory + "/共享文件")
	shareDir <- 1
}

func miscEnvs (mEnvRd chan int8) {
	if confOpts.useZink == true {
		addEnv("__GLX_VENDOR_LIBRARY_NAME=mesa")
		addEnv("MESA_LOADER_DRIVER_OVERRIDE=zink")
		addEnv("GALLIUM_DRIVER=zink")
		addEnv("LIBGL_KOPPER_DRI2=1")
		addEnv("__EGL_VENDOR_LIBRARY_FILENAMES=/usr/share/glvnd/egl_vendor.d/50_mesa.json")
	}
	if confOpts.qt5Compat == true {
		addEnv("QT_QPA_PLATFORMTHEME=xdgdesktopportal")
	}
	const file = "source /run/portable-generated.env"
	wrErr := os.WriteFile(
		xdgDir.runtimeDir + "/portable/" + confOpts.appID + "/bashrc",
		[]byte(file),
		0700)
	if wrErr != nil {
		pecho("warn", "Unable to write bashrc: " + wrErr.Error())
	}
	addEnv("GDK_DEBUG=portals")
	addEnv("GTK_USE_PORTAL=1")
	addEnv("QT_AUTO_SCREEN_SCALE_FACTOR=1")
	addEnv("QT_ENABLE_HIGHDPI_SCALING=1")
	addEnv(`PS1=╰─>Portable·` + confOpts.appID + `·🧐⤔ `)
	addEnv("QT_SCALE_FACTOR=" + os.Getenv("QT_SCALE_FACTOR"))
	addEnv("HOME=" + xdgDir.dataDir + "/" + confOpts.stateDirectory)
	addEnv("XDG_SESSION_TYPE=" + os.Getenv("${XDG_SESSION_TYPE}"))
	addEnv("WAYLAND_DISPLAY=" + xdgDir.runtimeDir + "/wayland-0")
	addEnv("DBUS_SESSION_BUS_ADDRESS=unix:path=/run/sessionBus")
	mEnvRd <- 1
}

func prepareEnvs(readyChan chan int8) {
	imChan := make(chan int8, 1)
	xdgEnvChan := make(chan int8, 1)
	shareDirChan := make(chan int8, 1)
	miscEnvChan := make(chan int8, 1)
	go imEnvs(imChan)
	go setXDGEnvs(xdgEnvChan)
	go setupSharedDir(shareDirChan)
	go miscEnvs(miscEnvChan)
	userEnvs, err := os.OpenFile(xdgDir.dataDir + "/" + confOpts.stateDirectory + "/portable.env", os.O_RDONLY, 0700)
	if err != nil {
		pecho("info", "Unable to read user defined environment variables: " + err.Error())
		if os.IsNotExist(err) {
			var template string = "# This file accepts simple KEY=VAL envs"
			os.WriteFile(
				xdgDir.dataDir + "/" + confOpts.stateDirectory + "/portable.env",
				[]byte(template),
				0700,
			)
		} else {
			pecho("warn", "Unable to open file for reading environment variables: " + err.Error())
		}
	} else {
		userEnvRead, errRead := io.ReadAll(userEnvs)
		if errRead != nil {
			pecho("warn", "I/O error reading environment variables: " + errRead.Error())
		} else {
			lines := strings.Split(strings.TrimRight(string(userEnvRead), "\n"), "\n")
			for _, line := range lines {
				addEnv(line)
			}
		}
	}
	packageEnvs, errPkg := os.OpenFile(confOpts.confPath, os.O_RDONLY, 0700)
	if errPkg != nil {
		pecho("crit", "Could not open package config " + confOpts.confPath + ": " + errPkg.Error())
	}
	pkgRead, errPkgR := io.ReadAll(packageEnvs)
	if errPkgR != nil {
		pecho("crit", "I/O error reading config: " + errPkgR.Error())
	}
	pkgEnv := strings.Split(strings.TrimRight(string(pkgRead), "\n"), "\n")
	for _, line := range pkgEnv {
		addEnv(line)
	}

	<- shareDirChan
	<- miscEnvChan
	<- xdgEnvChan
	<- imChan
	readyChan <- 1
}

func genBwArg(argChan chan int8, pwChan chan []string) {
	wayDisplayChan := make(chan[]string, 1)
	go waylandDisplay(wayDisplayChan)
	inputChan := make(chan []string, 1)
	go inputBind(inputChan)
	instChan := make(chan int8, 1)
	go instSignalFile(instChan)
	gpuChan := make(chan []string, 1)
	go gpuBind(gpuChan)
	camChan := make(chan []string, 1)
	go tryBindCam(camChan)
	miscChan := make(chan []string, 1)
	go miscBinds(miscChan, pwChan)
	xChan := make(chan []string, 1)
	go bindXAuth(xChan)

	if internalLoggingLevel > 1 {
		runtimeInfo.bwCmd = append(runtimeInfo.bwCmd, "--quiet")
	}

	// Define global systemd args first
	runtimeInfo.bwCmd = append(
		runtimeInfo.bwCmd,
		"--user",
		"--pty",
		"--service-type=notify-reload",
		"--wait",
		"--unit=app-portable-" + confOpts.appID,
		"--slice=app.slice",
		"-p", "Delegate=yes",
		"-p", "DelegateSubgroup=portable-cgroup",
		"-p", "BindsTo=" + confOpts.friendlyName + "-dbus.service",
		"-p", "Description=Portable Sandbox for " + confOpts.friendlyName + "(" + confOpts.appID + ")",
		"-p", "Documentation=https://github.com/Kraftland/portable",
		"-p", "ExitType=cgroup",
		"-p", "NotifyAccess=all",
		"-p", "TimeoutStartSec=infinity",
		"-p", "OOMPolicy=stop",
		"-p", "SecureBits=noroot-locked",
		"-p", "NoNewPrivileges=yes",
		"-p", "KillMode=control-group",
		"-p", "MemoryHigh=90%",
		"-p", "ManagedOOMSwap=kill",
		"-p", "ManagedOOMMemoryPressure=kill",
		"-p", "OOMScoreAdjust=100",
		"-p", "IPAccounting=yes",
		"-p", "MemoryPressureWatch=yes",
		"-p", "SyslogIdentifier=portable-" + confOpts.appID,
		"-p", "SystemCallLog=@privileged @debug @cpu-emulation @obsolete io_uring_enter io_uring_register io_uring_setup @resources",
		"-p", "SystemCallLog=~@sandbox",
		"-p", "PrivateIPC=yes",
		"-p", "ProtectClock=yes",
		"-p", "CapabilityBoundingSet=",
		"-p", "RestrictSUIDSGID=yes",
		"-p", "LockPersonality=yes",
		"-p", "RestrictRealtime=yes",
		"-p", "ProtectProc=invisible",
		"-p", "ProcSubset=pid",
		"-p", "PrivateUsers=yes",
		"-p", "ProtectControlGroups=private",
		"-p", "PrivateMounts=yes",
		"-p", "ProtectHome=no",
		"-p", "KeyringMode=private",
		"-p", "TimeoutStopSec=10s",
		"-p", "UMask=077",
		"-p",
		"EnvironmentFile=" + xdgDir.runtimeDir + "/portable/" + confOpts.appID + "/generated.env",
		"-p", "Environment=instanceId=" + runtimeInfo.flatpakInstanceID,
		"-p", "Environment=busDir=" + xdgDir.runtimeDir + "/app/" + confOpts.appID,
		"-p", "UnsetEnvironment=GNOME_SETUP_DISPLAY",
		"-p", "UnsetEnvironment=PIPEWIRE_REMOTE",
		"-p", "UnsetEnvironment=PAM_KWALLET5_LOGIN",
		"-p", "UnsetEnvironment=GTK2_RC_FILES",
		"-p", "UnsetEnvironment=ICEAUTHORITY",
		"-p", "UnsetEnvironment=MANAGERPID",
		"-p", "UnsetEnvironment=INVOCATION_ID",
		"-p", "UnsetEnvironment=MANAGERPIDFDID",
		"-p", "UnsetEnvironment=SSH_AUTH_SOCK",
		"-p", "UnsetEnvironment=MAIL",
		"-p", "UnsetEnvironment=SYSTEMD_EXEC_PID",
		"-p", "WorkingDirectory=" + xdgDir.dataDir + "/" + confOpts.stateDirectory,
		"-p", "ExecReload=bash -c 'kill --signal SIGALRM 2'",
		"-p", "ReloadSignal=SIGALRM",
		//"-p", "EnvironmentFile=" + xdgDir.runtimeDir + "/portable/" + confOpts.appID + "/portable-generated-new.env",
		"-p", "SystemCallFilter=~@clock",
		"-p", "SystemCallFilter=~@cpu-emulation",
		"-p", "SystemCallFilter=~@module",
		"-p", "SystemCallFilter=~@obsolete",
		"-p", "SystemCallFilter=~@raw-io",
		"-p", "SystemCallFilter=~@reboot",
		"-p", "SystemCallFilter=~@swap",
		"-p", "SystemCallErrorNumber=EAGAIN",
	)

	if confOpts.bindNetwork == false {
		pecho("info", "Network Access disabled")
		runtimeInfo.bwCmd = append(
			runtimeInfo.bwCmd,
			"-p", "PrivateNetwork=yes",
		)
	} else {
		pecho("info", "Network Access allowed")
		runtimeInfo.bwCmd = append(
			runtimeInfo.bwCmd,
			"-p", "PrivateNetwork=no",
		)
	}

	pecho("debug", "Built systemd-run arguments")

	runtimeInfo.bwCmd = append(
		runtimeInfo.bwCmd,
		"--",
		"bwrap",
		// Unshares
		"--new-session",
		"--unshare-cgroup-try",
		"--unshare-ipc",
		"--unshare-uts",
		"--unshare-pid",
		"--unshare-user",

		// Tmp binds
		"--tmpfs",		"/tmp",

		// Dev binds
		"--dev",		"/dev",
		"--tmpfs",		"/dev/shm",
		"--dev-bind-try",	"/dev/mali", "/dev/mali",
		"--dev-bind-try",	"/dev/mali0", "/dev/mali0",
		"--dev-bind-try",	"/dev/umplock", "/dev/umplock",
		"--mqueue",		"/dev/mqueue",
		"--dev-bind",		"/dev/dri", "/dev/dri",
		"--dev-bind-try",	"/dev/udmabuf", "/dev/udmabuf",
		"--dev-bind-try",	"/dev/ntsync", "/dev/ntsync",
		"--dir",		"/top.kimiblock.portable",

		// Sysfs entries
		"--tmpfs",		"/sys",
		"--ro-bind-try",	"/sys/module", "/sys/module",
		"--ro-bind-try",	"/sys/dev/char", "/sys/dev/char",
		"--tmpfs",		"/sys/devices",
		"--ro-bind-try",	"/sys/fs/cgroup", "/sys/fs/cgroup",
		"--dev-bind",		"/sys/class/drm", "/sys/class/drm",
		"--bind-try",		"/sys/devices/system", "/sys/devices/system",
		"--ro-bind",		"/sys/kernel", "/sys/kernel",

		// usr binds
		"--bind",		"/usr", "/usr",
		"--overlay-src",	"/usr/bin",
		"--overlay-src",	"/usr/lib/portable/overlay-usr",
		"--ro-overlay",		"/usr/bin",
		"--symlink",		"/usr/lib", "/lib",
		"--symlink",		"/usr/lib", "/lib64",
		"--symlink",		"/usr/bin", "/bin",
		"--symlink",		"/usr/bin", "/sbin",

		// Proc binds
		"--proc",		"/proc",
		"--dev-bind-try",	"/dev/null", "/dev/null",
		"--ro-bind-try",	"/dev/null", "/proc/uptime",
		"--ro-bind-try",	"/dev/null", "/proc/modules",
		"--ro-bind-try",	"/dev/null", "/proc/cmdline",
		"--ro-bind-try",	"/dev/null", "/proc/diskstats",
		"--ro-bind-try",	"/dev/null", "/proc/devices",
		"--ro-bind-try",	"/dev/null", "/proc/config.gz",
		"--ro-bind-try",	"/dev/null", "/proc/mounts",
		"--ro-bind-try",	"/dev/null", "/proc/loadavg",
		"--ro-bind-try",	"/dev/null", "/proc/filesystems",

		// FHS dir
		"--perms",		"0000",
		"--tmpfs",		"/boot",
		"--perms",		"0000",
		"--tmpfs",		"/srv",
		"--perms",		"0000",
		"--tmpfs",		"/root",
		"--perms",		"0000",
		"--tmpfs",		"/media",
		"--perms",		"0000",
		"--tmpfs",		"/mnt",
		"--tmpfs",		"/home",
		"--tmpfs",		"/var",
		"--symlink",		"/run", "/var/run",
		"--symlink",		"/run/lock", "/var/lock",
		"--tmpfs",		"/var/empty",
		"--tmpfs",		"/var/lib",
		"--tmpfs",		"/var/log",
		"--tmpfs",		"/var/opt",
		"--tmpfs",		"/var/spool",
		"--tmpfs",		"/var/tmp",
		"--ro-bind-try",	"/opt", "/opt",

		"--ro-bind-try",	"/var/cache/fontconfig", "/var/cache/fontconfig",

		// Run binds
		"--bind",
			xdgDir.runtimeDir + "/portable/" + confOpts.appID,
			"/run",
		"--bind",
			xdgDir.runtimeDir + "/portable/" + confOpts.appID,
			xdgDir.runtimeDir + "/portable/" + confOpts.appID,
		"--ro-bind-try",
			"/run/systemd/userdb/io.systemd.Home",
			"/run/systemd/userdb/io.systemd.Home",
		"--ro-bind",
			xdgDir.runtimeDir + "/app/" + confOpts.appID + "/bus",
			"/run/sessionBus",
		"--ro-bind-try",
			xdgDir.runtimeDir + "/app/" + confOpts.appID + "-a11y",
			xdgDir.runtimeDir + "/at-spi",
		"--dir",		"/run/host",
		"--bind",
			xdgDir.runtimeDir + "/doc/by-app/" + confOpts.appID,
			xdgDir.runtimeDir + "/doc",
		"--ro-bind-try",
			"/run/systemd/resolve/stub-resolv.conf",
			"/run/systemd/resolve/stub-resolv.conf",
		"--bind",
			xdgDir.runtimeDir + "/systemd/notify",
			xdgDir.runtimeDir + "/systemd/notify",
		"--ro-bind-try",
			xdgDir.runtimeDir + "/pulse",
			xdgDir.runtimeDir + "/pulse",

		// HOME binds
		"--bind",
			xdgDir.dataDir + "/" + confOpts.stateDirectory,
			xdgDir.home,
		"--bind",
			xdgDir.dataDir + "/" + confOpts.stateDirectory,
			xdgDir.dataDir + "/" + confOpts.stateDirectory,

		"--ro-bind",		"/etc", "/etc",

		// Privacy mounts
		"--tmpfs",		"/proc/1",
		"--tmpfs",		"/usr/share/applications",
		"--tmpfs",		xdgDir.home + "/options",
		"--tmpfs",		xdgDir.dataDir + "/" + confOpts.stateDirectory + "/options",

	)

	xArgs := <- xChan
	runtimeInfo.bwCmd = append(
		runtimeInfo.bwCmd,
		xArgs...
	)

	wayArgs := <- wayDisplayChan
	runtimeInfo.bwCmd = append(
		runtimeInfo.bwCmd,
		wayArgs...
	)

	miscArgs := <- miscChan
	runtimeInfo.bwCmd = append(
		runtimeInfo.bwCmd,
		miscArgs...
	)


	inputArgs := <- inputChan
	runtimeInfo.bwCmd = append(
		runtimeInfo.bwCmd,
		inputArgs...
	)

	camArgs := <- camChan
	runtimeInfo.bwCmd = append(
		runtimeInfo.bwCmd,
		camArgs...
	)

	gpuArgs := <- gpuChan
	runtimeInfo.bwCmd = append(
		runtimeInfo.bwCmd,
		gpuArgs...
	)

	// NO arg should be added below this point
	runtimeInfo.bwCmd = append(
		runtimeInfo.bwCmd,
		"--",
		"/usr/lib/portable/helper",
	)

	addEnv("stop")

	var chanReady int8 = <- instChan
	chanReady++
	argChan <- 1
}

func isEnvValid(env string) bool {
		return envRegex.MatchString(env)
}

func flushEnvs() {
	os.RemoveAll(xdgDir.runtimeDir + "/portable/" + confOpts.appID + "/generated.env")
	fd, err := os.Create(xdgDir.runtimeDir + "/portable/" + confOpts.appID + "/generated.env")
	if err != nil {
		pecho("crit", "Could not store generated environment variables: " + err.Error())
	}
	for {
		envPend := <- envsChan
		if envPend == "stop" {
			for _, env := range runtimeInfo.sdEnvParm {
				_, err = fmt.Fprintln(fd, env)
				if err != nil {
					pecho("crit", "I/O error writing envs: " + err.Error())
				}
			}
			envsFlushReady <- 1
			break
		}
		if isEnvValid(envPend) == false {
			pecho("debug", "Rejecting invalid environment variable: " + envPend)
		}
		runtimeInfo.sdEnvParm = append(
			runtimeInfo.sdEnvParm,
			envPend,
		)
	}
}

func translatePath(input string) (output string) {
	output = strings.ReplaceAll(input, xdgDir.home, xdgDir.dataDir + "/" + confOpts.stateDirectory)
	return
}

func miscBinds(miscChan chan []string, pwChan chan []string) {
	var miscArgs = []string{}
	pwArgs := <- pwChan
	miscArgs = append(
		miscArgs,
		pwArgs...
	)
	if confOpts.mountInfo == true {
		miscArgs = append(
			miscArgs,
			"--ro-bind",
				"/dev/null",
				xdgDir.runtimeDir + "/.flatpak/" + runtimeInfo.flatpakInstanceID + "-private/run-environ",
			"--ro-bind",
				xdgDir.runtimeDir + "/.flatpak/" + runtimeInfo.flatpakInstanceID,
				xdgDir.runtimeDir + "/.flatpak/" + runtimeInfo.flatpakInstanceID,
			"--ro-bind",
				xdgDir.runtimeDir + "/.flatpak/" + runtimeInfo.flatpakInstanceID,
				xdgDir.runtimeDir + "/flatpak-runtime-directory",
			"--ro-bind",
				xdgDir.runtimeDir + "/portable/" + confOpts.appID + "/flatpak-info",
				"/.flatpak-info",
			"--ro-bind",
				xdgDir.runtimeDir + "/portable/" + confOpts.appID + "/flatpak-info",
				xdgDir.runtimeDir + "/.flatpak-info",
			"--ro-bind",
				xdgDir.runtimeDir + "/portable/" + confOpts.appID + "/flatpak-info",
				xdgDir.dataDir + "/" + confOpts.stateDirectory + "/.flatpak-info",
			"--tmpfs",		xdgDir.home + "/.var",
			"--tmpfs",		xdgDir.dataDir + "/" + confOpts.stateDirectory + "/.var",
			"--bind",
				xdgDir.dataDir + "/" + confOpts.stateDirectory,
				xdgDir.dataDir + "/" + confOpts.stateDirectory + "/.var/app/" + confOpts.appID,
			"--tmpfs",
				xdgDir.dataDir + "/" + confOpts.stateDirectory + "/.var/app/" + confOpts.appID + "/options",
		)
	}

	_, err := os.Stat("/usr/lib/flatpak-xdg-utils/flatpak-spawn")
	if err == nil {
		miscArgs = append(
			miscArgs,
			"--ro-bind",
			"/usr/lib/portable/overlay-usr/flatpak-spawn",
			"/usr/lib/flatpak-xdg-utils/flatpak-spawn",
		)
	}

	dirFd, errRead := os.Stat("/etc/kernel")
	if errRead == nil && dirFd.IsDir() == true {
		miscArgs = append(
			miscArgs,
			"--tmpfs",
			"/etc/kernel",
		)
	}
	miscArgs = append(
		miscArgs,
		"--ro-bind-try",
			xdgDir.confDir + "/fontconfig",
			translatePath(xdgDir.confDir + "/fontconfig"),
		"--ro-bind-try",
			xdgDir.confDir + "/gtk-3.0/gtk.css",
			translatePath(xdgDir.confDir + "/gtk-3.0/gtk.css"),
		"--ro-bind-try",
			xdgDir.confDir + "/gtk-3.0/colors.css",
			translatePath(xdgDir.confDir + "/gtk-3.0/colors.css"),
		"--ro-bind-try",
			xdgDir.confDir + "/gtk-4.0/gtk.css",
			translatePath(xdgDir.confDir + "/gtk-4.0/gtk.css"),
		"--ro-bind-try",
			xdgDir.confDir + "/qt6ct",
			translatePath(xdgDir.confDir + "/qt6ct"),
		"--ro-bind-try",
			xdgDir.dataDir + "/fonts",
			translatePath(xdgDir.dataDir + "/fonts"),
		"--ro-bind-try",
			xdgDir.dataDir + "/icons",
			translatePath(xdgDir.dataDir + "/icons"),
	)

	miscChan <- miscArgs
}

func bindXAuth(xauthChan chan []string) {
	var xArg = []string{}
	if confOpts.waylandOnly == false {
		xArg = append(
			xArg,
			"--bind-try",		"/tmp/.X11-unix", "/tmp/.X11-unix",
			"--bind-try",		"/tmp/.XIM-unix", "/tmp/.XIM-unix",
		)
		osAuth := os.Getenv("XAUTHORITY")
		_, err := os.Stat(osAuth)
		if err == nil {
			pecho("debug", "XAUTHORITY specified as absolute path: " + osAuth)
			xArg = append(
				xArg,
				"--ro-bind",
					osAuth,
					"/run/.Xauthority",
			)
			addEnv("XAUTHORITY=/run/.Xauthority")
		} else {
			osAuth = xdgDir.home + "/.Xauthority"
			_, err = os.Stat(osAuth)
			if err == nil {
				pecho(
					"warn",
					"Implied XAUTHORITY " + osAuth + ", this is not recommended",
				)
				xArg = append(
					xArg,
					"--ro-bind",
						osAuth,
						"/run/.Xauthority",
				)
				addEnv("XAUTHORITY=/run/.Xauthority")
			} else {
				pecho("warn", "Could not locate XAUTHORITY file")
			}
		}
		addEnv("DISPLAY=" + os.Getenv("DISPLAY"))
	}
	xauthChan <- xArg
}

func gpuBind(gpuChan chan []string) {
	var gpuArg = []string{}
	// SHOULD contain strings like card0, card1 etc
	var totalGpus = []string{}
	var activeGpus = []string{}
	var cardSums int = 0

	gpuEntries, err := os.ReadDir("/sys/class/drm")
	if err != nil {
		pecho(
			"warn",
			"Unable to parse GPU information: failed reading /sys/class/drm: " + err.Error())
		return
	}
	for _, cardName := range gpuEntries {
		if strings.Contains(cardName.Name(), "-") {
			continue
		} else if strings.HasPrefix(cardName.Name(), "card") {
			cardSums++
			totalGpus = append(
				totalGpus,
				cardName.Name(),
			)
		}
	}

	var trailingS string

	if len(os.Getenv("PORTABLE_ASSUME_SINGLE_GPU")) != 0 {
		cardSums = 1
	}

	gpuArg = append(
		gpuArg,
		"--tmpfs", "/dev/dri",
		"--tmpfs", "/sys/class/drm",
	)

	switch cardSums {
		case 0:
			pecho("warn", "Found no GPU")
		case 1:
			nvChan := make(chan []string, 1)
			go tryBindNv(nvChan)
			nvArgs := <- nvChan
			gpuArg = append(
				gpuArg,
				nvArgs...,
			)
			for _, cardName := range totalGpus {
				gpuArg = append(
					gpuArg,
					bindCard(cardName)...
				)
			}
			activeGpus = totalGpus
		default:
			trailingS = "s"
			if confOpts.gameMode == true {
				envChan := make(chan int8, 1)
				setOffloadEnvs(envChan)
				nvChan := make(chan []string, 1)
				go tryBindNv(nvChan)
				nvArgs := <- nvChan
				gpuArg = append(
					gpuArg,
					nvArgs...,
				)
				for _, cardName := range totalGpus {
					gpuArg = append(
						gpuArg,
						bindCard(cardName)...
					)
				}
				envReady := <- envChan
				envReady++
			} else {
				for _, cardName := range totalGpus {
					connectors, err := os.ReadDir("/sys" + cardName)
					if err != nil {
						pecho(
							"warn",
							"Failed to read GPU connector status: " + err.Error(),
						)
						continue
					}
					for _, connectorName := range connectors {
						if strings.HasPrefix(connectorName.Name(), "card") == false {
							continue
						}
						conStatFd, err := os.OpenFile(
							"/sys/class/drm/" + cardName + "/" + connectorName.Name() + "/status",
							os.O_RDONLY,
							0700,
						)
						if err != nil {
							pecho(
								"warn",
								"Failed to open GPU status: " + err.Error(),
							)
						}
						conStat, ioErr := io.ReadAll(conStatFd)
						if ioErr != nil {
							pecho(
								"warn",
								"Failed to read GPU status: " + ioErr.Error(),
							)
						}
						if strings.Contains(string(conStat), "disconnected") {
							continue
						} else {
							activeGpus = append(
								activeGpus,
								cardName,
							)
							break
						}
					}
				}
				pecho("debug", "Active GPU slice: " + strings.Join(activeGpus, ", "))
				for _, cardName := range activeGpus {
					bindCard(cardName)
				}
			}
	}
	gpuChan <- gpuArg
	var activeGPUList string = strings.Join(activeGpus, ", ")
	pecho("debug", "Generated GPU bind parameters: " + strings.Join(gpuArg, ", "))
	pecho(
	"debug",
	"Found " + strconv.Itoa(cardSums) + " GPU" + trailingS + ", identified active: " + activeGPUList)
}

func setOffloadEnvs(envsReady chan int8) () {
	var nvExist bool = false
	addEnv("VK_LOADER_DRIVERS_DISABLE=none")
	_, err := os.Stat("/dev/nvidia0")
	if err == nil {
		nvExist = true
	}

	if nvExist == true {
		addEnv("__NV_PRIME_RENDER_OFFLOAD=1")
		addEnv("__VK_LAYER_NV_optimus=NVIDIA_only")
		addEnv("__GLX_VENDOR_LIBRARY_NAME=nvidia")
		addEnv("VK_LOADER_DRIVERS_SELECT=nvidia_icd.json")
	} else {
		addEnv("DRI_PRIME=1")
	}
	envsReady <- 1
}

func bindCard(cardName string) (cardBindArg []string) {
	resolveUdevArgs := []string{
		"info",
		"/sys/class/drm/" + cardName,
		"--query=path",
	}
	resolveUdevCmd := exec.Command("/usr/bin/udevadm", resolveUdevArgs...)
	resolveUdevCmd.Stderr = os.Stderr
	path, err := resolveUdevCmd.Output()
	if err != nil {
		pecho("warn", "Failed to resolve GPU " + cardName + ": " + err.Error())
		return
	}
	sysfsPath := "/sys" + strings.TrimSpace(string(path))
	cardBindArg = append(
		cardBindArg,
		"--dev-bind",
			sysfsPath, sysfsPath,
		"--dev-bind",
			"/sys/class/drm/" + cardName,
			"/sys/class/drm/" + cardName,
		"--dev-bind",
			"/dev/dri/" + cardName,
			"/dev/dri/" + cardName,
	)
	devDrmPath := strings.TrimSuffix(sysfsPath, "/" + cardName)
	drmEntries, readErr := os.ReadDir(devDrmPath)
	pecho("debug", "Got sysfs path from udev: " + devDrmPath)
	if readErr != nil {
		pecho("warn", "Failed to read "+ devDrmPath + ": " + readErr.Error())
		return
	} else {
		for _, candidate := range drmEntries {
			if strings.HasPrefix(candidate.Name(), "renderD") {
				cardBindArg = append(
					cardBindArg,
					"--dev-bind",
						"/dev/dri/" + candidate.Name(),
						"/dev/dri/" + candidate.Name(),
					"--dev-bind",
						"/sys/class/drm/" + candidate.Name(),
						"/sys/class/drm/" + candidate.Name(),
				)
			}
		}
	}

	return
}

func tryBindCam(camChan chan []string) {
	camArg := []string{}
	if confOpts.bindCameras == true {
		camEntries, err := os.ReadDir("/dev")
		if err != nil {
			pecho("warn", "Failed to parse camera entries")
			return
		}
		for _, file := range camEntries {
			if strings.HasPrefix(file.Name(), "video") && file.IsDir() == false {
				camArg = append(
					camArg,
					"--dev-bind",
						"/dev/" + file.Name(),
						"/dev/" + file.Name(),
				)
			}
		}
	}
	camChan <- camArg
}

func tryBindNv(nvChan chan []string) {
	nvDevsArg := []string{}
	devEntries, err := os.ReadDir("/dev")
	if err != nil {
		pecho("warn", "Failed to read /dev: " + err.Error())
	} else {
		for _, devFile := range devEntries {
			if strings.HasPrefix(devFile.Name(), "nvidia") {
				nvDevsArg = append(
					nvDevsArg,
					"--dev-bind",
						"/dev/" + devFile.Name(),
						"/dev/" + devFile.Name(),
				)
			}
		}
	}
	nvChan <- nvDevsArg
}

func inputBind(inputBindChan chan []string) {
	inputBindArg := []string{}
	if confOpts.bindInputDevices == false {
		inputBindChan <- inputBindArg
		return
	}
	inputBindArg = append(
		inputBindArg,
		"--dev-bind-try",	"/sys/class/leds", "/sys/class/leds",
		"--dev-bind-try",	"/sys/class/input", "/sys/class/input",
		"--dev-bind-try",	"/sys/class/hidraw", "/sys/class/hidraw",
		"--dev-bind-try",	"/dev/input", "/dev/input",
		"--dev-bind-try",	"/dev/uinput", "/dev/uinput",
	)

	devEntries, err := os.ReadDir("/dev")
	if err != nil {
		pecho("warn", "Could not parse /dev/ for hidraw devices: " + err.Error())
	} else {
		for _, entry := range devEntries {
			if strings.HasPrefix(entry.Name(), "hidraw") {
				pecho("debug", "Detected hidraw input device " + entry.Name())
				inputBindArg = append(
					inputBindArg,
					"--dev-bind",
						"/dev/" + entry.Name(),
						"/dev/" + entry.Name(),
				)
				udevArgs := []string{
					"info",
					"/dev/" + entry.Name(),
					"-qpath",
				}
				udevExec := exec.Command("udevadm", udevArgs...)
				sysDevice, sysErrout := udevExec.Output()
				if sysErrout != nil {
					pecho(
					"warn",
					"Unable to resolve device path using udev: " + sysErrout.Error(),
					)
				} else {
					inputBindArg = append(
						inputBindArg,
						"--dev-bind",
							"/sys" + strings.TrimSpace(string(sysDevice)),
							"/sys" + strings.TrimSpace(string(sysDevice)),
					)
				}
			}
		}
	}

	devEntries, err = os.ReadDir("/dev/input")
	if err != nil {
		pecho("warn", "Could not parse /dev/input for devices: " + err.Error())
	} else {
		for _, entry := range devEntries {
			if entry.IsDir() == true {
				continue
			}
			if strings.HasPrefix(entry.Name(), "event") || strings.HasPrefix(entry.Name(), "js") {
				pecho("debug", "Detected input device " + entry.Name())
				inputBindArg = append(
					inputBindArg,
					"--dev-bind",
						"/dev/input/" + entry.Name(),
						"/dev/input/" + entry.Name(),
				)
				udevArgs := []string{
					"info",
					"/dev/input/" + entry.Name(),
					"-qpath",
				}
				udevExec := exec.Command("udevadm", udevArgs...)
				sysDevice, sysErrout := udevExec.Output()
				if sysErrout != nil {
					pecho(
					"warn",
					"Unable to resolve device path using udev: " + sysErrout.Error(),
					)
				} else {
					inputBindArg = append(
						inputBindArg,
						"--dev-bind",
							"/sys" + strings.TrimSpace(string(sysDevice)),
							"/sys" + strings.TrimSpace(string(sysDevice)),
					)
				}
			}
		}
	}
	inputBindChan <- inputBindArg
	pecho("debug", "Finished calculating input arguments: " + strings.Join(inputBindArg, " "))
}

func instSignalFile(instChan chan int8) {
	const content string = "false"
	os.WriteFile(
		xdgDir.runtimeDir + "/portable/" + confOpts.appID + "/startSignal",
		[]byte(content),
		0700,
	)
	instChan <- 1
	pecho("debug", "Created signal file")
}

func main() {
	fmt.Println("Portable daemon", version, "starting")
	readConfChan := make(chan int, 1)
	go readConf(readConfChan)
	xdgChan := make(chan int, 1)
	go lookUpXDG(xdgChan)
	cmdChan := make(chan int, 1)
	<- xdgChan
	go cmdlineDispatcher(cmdChan)
	varChan := make(chan int, 1)
	go getVariables(varChan)
	<- varChan
	<- readConfChan
	<- cmdChan
	go flushEnvs()
	envPreChan := make(chan int8, 1)
	go prepareEnvs(envPreChan)
	pecho("debug", "getVariables, lookupXDG, cmdlineDispatcher and readConf are ready")
	if startAct == "abort" {
		os.Exit(0)
	}

	// Warn multi-instance here
	cleanUnitChan := make(chan int8, 1)
	go doCleanUnit(cleanUnitChan)
	genChan := make(chan int8, 2)
	go genFlatpakInstanceID(genChan)
	argChan := make(chan int8, 1)
	pwSecContextChan := make(chan []string, 1)
	<- genChan
	go genBwArg(argChan, pwSecContextChan)
	instDesktopChan := make(chan int8, 1)
	go instDesktopFile(instDesktopChan)
	<- genChan
	<- cleanUnitChan
	go pwSecContext(pwSecContextChan)
	pecho("debug", "Flatpak info and cleaning ready")

	proxyChan := make(chan int8, 1)
	go startProxy(proxyChan)
	<- proxyChan
	<- instDesktopChan
	<- argChan
	<- envPreChan
	pecho("debug", "Proxy, PipeWire, argument generation and desktop file ready")
	addEnv("stop")
	// TODO: add non-sandbox code here
	startApp()
	stopApp("normal")
}