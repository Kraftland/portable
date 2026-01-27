package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

const (
	version float32 = 0.1
)

type portableConfigOpts struct {
	confPath		string
	friendlyName		string
	appID			string
	stateDirectory		string
	launchTarget		string	// this one may be empty?
	busLaunchTarget		string	// also may be empty
	bindNetwork		bool
	terminateImmediately	bool
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
	runtimeDir		string
	confOpts		portableConfigOpts
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
	runtimeDir = os.Getenv("XDG_RUNTIME_DIR")
	if len(runtimeDir) == 0 {
		pecho("warn", "XDG_RUNTIME_DIR not set")
	} else {
		var runtimeDebugMsg string = "XDG_RUNTIME_DIR set to: " + runtimeDir
		pecho("debug", runtimeDebugMsg)
		runtimeDirInfo, errRuntimeDir := os.Stat(runtimeDir)
		var errRuntimeDirPrinted string = "Could not determine the status of XDG Runtime Directory "
		if errRuntimeDir != nil {
			println(errRuntimeDir)
			pecho("crit", errRuntimeDirPrinted)
		}
		if runtimeDirInfo.IsDir() == false {
			pecho("crit", "XDG_RUNTIME_DIR is not a directory")
		}
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

func stopApp(operation string) {
	go stopMainApp()
	go stopMainAppCompat()
	go stopSlice()
	switch operation {
		case "normal":
			pecho("debug", "Cleaning leftovers...")
		default:
			pecho("crit", "Unknown operation for stopApp: " + operation)
	}
}

func startApp() {
	sdExec := exec.Command("xargs", "-0", "-a", runtimeDir + "/portable/" + confOpts.appID + "/bwrapArgs", "systemd-run")
	sdExec.Stderr = os.Stderr
	sdExec.Stdout = os.Stdout
	sdExec.Stdin = os.Stdin
	sdExecErr := sdExec.Run()
	if sdExecErr != nil {
		fmt.Println(sdExecErr)
		pecho("crit", "Unable to start systemd-run")
	}
}

func main() {
	fmt.Println("Portable daemon", version, "starting")
	readConfChan := make(chan int)
	go readConf(readConfChan)
	varChan := make(chan int)
	go getVariables(varChan)
	getVariablesReady := <- varChan
	readConfReady := <- readConfChan
	if getVariablesReady == 1 && readConfReady == 1 {
		pecho("debug", "getVariables and readConf are ready")
	}
	startApp()
	stopApp("normal")
}