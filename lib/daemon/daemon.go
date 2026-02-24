package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
	"net"
	"net/http"
	"sync"
	"github.com/KarpelesLab/reflink"
	"github.com/coreos/go-systemd/v22/dbus"
	sdutil "github.com/coreos/go-systemd/v22/util"
	godbus "github.com/godbus/dbus/v5"
	udev "github.com/jochenvg/go-udev"
)

const (
	version		float32	=	13.99
	controlFile	string	=	"instanceId=inIdHold\nappID=idHold\nbusDir=busHold\nfriendlyName=friendlyHold"
)

type RUNTIME_OPT struct {
	action		bool
	argStop		bool
	applicationArgs	[]string
	userExpose	chan map[string]string
	userLang	string
	miTerminate	bool
}

type RUNTIME_PARAMS struct {
	instanceID		string
	waylandDisplay		string
	bwCmd			[]string
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
	dbusWake		bool
	mountInfo		bool
}

type confTarget struct {
	str			*string
	b			*bool
}

// Defaults should be set in readConf()
var targets = map[string]confTarget{
	"appID":		{str: &confOpts.appID},
	"friendlyName":		{str: &confOpts.friendlyName},
	"stateDirectory":	{str: &confOpts.stateDirectory},
	"launchTarget":		{str: &confOpts.launchTarget},
	"busLaunchTarget":	{str: &confOpts.busLaunchTarget},
	"mprisName":		{str: &confOpts.mprisName},
	"bindNetwork":		{b: &confOpts.bindNetwork},
	"terminateImmediately":	{b: &confOpts.terminateImmediately},
	"networkDeny":		{str: &confOpts.networkDeny},
	"allowClassicNotifs":	{b: &confOpts.allowClassicNotifs},
	"useZink":		{b: &confOpts.useZink},
	"qt5Compat":		{b: &confOpts.qt5Compat},
	"waylandOnly":		{b: &confOpts.waylandOnly},
	"gameMode":		{b: &confOpts.gameMode},
	"bindCameras":		{b: &confOpts.bindCameras},
	"bindPipewire":		{b: &confOpts.bindPipewire},
	"bindInputDevices":	{b: &confOpts.bindInputDevices},
	"allowInhibit":		{b: &confOpts.allowInhibit},
	"allowGlobalShortcuts":	{b: &confOpts.allowGlobalShortcuts},
	"dbusWake":		{b: &confOpts.dbusWake},
	"mountInfo":		{b: &confOpts.mountInfo},
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
	"dbusWake":		"bool",
	"mountInfo":		"bool",
}

var (
	internalLoggingLevel	int
	confOpts		portableConfigOpts
	runtimeInfo		RUNTIME_PARAMS
	xdgDir			XDG_DIRS
	runtimeOpt		RUNTIME_OPT
	envsChan		= make(chan string, 512)
	envsFlushReady		= make(chan int8, 1)
	startAct		string
	signalWatcherReady	= make(chan int8, 1)
	gpuChan 		= make(chan []string, 1)
	busArgChan		= make(chan []string, 1)
	socketStop		= make(chan int8, 10)
	stopAppChan		= make(chan int8, 512)
	stopAppDone		= make(chan int8)
	nvKernelModulePath 	= []string{
					"/sys/module/nvidia",
					"/sys/module/nvidia_drm",
					"/sys/module/nvidia_modeset",
					"/sys/module/nvidia_uvm",
					"/sys/module/nvidia_wmi_ec_backlight",
				}
	pechoChan		= make(chan []string, 128)
)

func pechoWorker() {
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
	pechoChan <- []string{
		"debug",
		"Initialized logging daemon",
	}
	for {
		chanRes := <- pechoChan
		switch chanRes[0] {
			case "debug":
				if internalLoggingLevel <= 1 {
					fmt.Println("[Debug] ", chanRes[1])
				}
			case "info":
				if internalLoggingLevel <= 2 {
					fmt.Println("[Info] ", chanRes[1])
				}
			case "warn":
				fmt.Println("[Warn] ", chanRes[1])
			case "crit":
				fmt.Println("[Critical] ", chanRes[1])
				stopApp()
				panic("A critical error has happened")
			default:
				fmt.Println("[Undefined] ", chanRes[1])
		}
	}
}

func pecho(level string, message string) {
	var msgSlice = []string{
		level,
		message,
	}
	pechoChan <- msgSlice
}


func sanityChecks() {
	if sdutil.IsRunningSystemd() == false {
		pecho("crit", "Portable requires the systemd service manager")
	}
	var appIDValid bool = true
	if strings.Contains(confOpts.appID, "org.freedesktop.impl") == true {
		appIDValid = false
	} else if strings.Contains(confOpts.appID, "org.gtk.vfs") == true {
		appIDValid = false
	} else if confOpts.appID == "org.mpris.MediaPlayer2" {
		appIDValid = false
	} else if len(confOpts.appID) == 0 {
		appIDValid = false
	} else if len(strings.Split(confOpts.appID, ".")) < 2 {
		appIDValid = false
	}
	if appIDValid == false {
		startAct = "abort"
		pecho("crit", "Invalid appID: " + confOpts.appID)
	}
	if len(confOpts.friendlyName) == 0 {
		pecho("crit", "Could not parse friendlyName")
	}
	if len(confOpts.stateDirectory) == 0 {
		pecho("crit", "Could not parse stateDirectory")
	}

	mountCheckArgs := []string{
		"--quiet",
		"--user",
		"--tty",
		"--wait",
		"--",
		"findmnt",
		"-R",
	}
	mountCheckCmd := exec.Command("systemd-run", mountCheckArgs...)
	mountCheckCmd.Stderr = os.Stderr
	stdout, errP := mountCheckCmd.StdoutPipe()
	if errP != nil {
		pecho("crit", "Failed to pipe findmnt output: " + errP.Error())
	}
	scanner := bufio.NewScanner(stdout)
	err := mountCheckCmd.Run()
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "/usr/bin/") {
			startAct = "abort"
			pecho("crit", "Found mount points inside /usr/bin")
		}
	}
	if err != nil {
		pecho("crit", "Could not check mountpoints: " + err.Error())
	}
}

func addEnv(envToAdd string) {
	envsChan <- envToAdd
}

func openHome () {
	cmdArgs := []string{
		xdgDir.dataDir + "/" + confOpts.stateDirectory,
	}
	openCmd := exec.Command("/usr/bin/xdg-open", cmdArgs...)
	openCmd.Stderr = os.Stderr
	openCmd.Run()
	os.Exit(0)
}

func resetDocs () {
	cmdArgs := []string{
		"permission-reset",
		confOpts.appID,
	}
	openCmd := exec.Command("flatpak", cmdArgs...)
	openCmd.Stderr = os.Stderr
	openCmd.Run()
	os.Exit(0)
}

func showStats() {
	getFlatpakInstanceID()
	cmdArgs := []string{
		"--user",
		"status",
		"app-portable-" + confOpts.appID + "-" + runtimeInfo.instanceID,
	}
	openCmd := exec.Command("systemctl", cmdArgs...)
	openCmd.Stderr = os.Stderr
	openCmd.Stdout = os.Stdout
	openCmd.Run()
	os.Exit(0)
}

func cmdlineDispatcher(cmdChan chan int8) {
	cmdlineArray := os.Args
	for index, value := range cmdlineArray {
		if index == 0 {
			continue
		}
		if runtimeOpt.action == true {
			runtimeOpt.action = false
			continue
		} else if runtimeOpt.argStop == true {
			runtimeOpt.applicationArgs = append(
				runtimeOpt.applicationArgs,
				value,
			)
			continue
		}
		switch value {
			case "--actions" :
			runtimeOpt.action = true
			if len(cmdlineArray) <= index + 1 {
				pecho("warn", "--actions require an argument")
				break
			}
			switch cmdlineArray[index + 1] {
				case "quit":
					runtimeOpt.miTerminate = true
					pecho("debug", "Received quit request from user")
				case "debug-shell":
					addEnv("_portableDebug=1")
				case "share-file":
					startAct = "abort"
					shareFile()
				case "share-files":
					startAct = "abort"
					shareFile()
				case "openhome":
					startAct = "abort"
					openHome()
				case "opendir":
					startAct = "abort"
					openHome()
				case "home":
					startAct = "abort"
					openHome()
				case "reset-document":
					startAct = "abort"
					resetDocs()
				case "reset-documents":
					startAct = "abort"
					resetDocs()
				case "stat":
					startAct = "abort"
					showStats()
				case "stats":
					startAct = "abort"
					showStats()
				default:
					pecho("warn", "Unrecognised action: " + cmdlineArray[index + 1])
			}
			case "--dbus-activation":
				addEnv("_portableBusActivate=1")
			case "--":
				runtimeOpt.argStop = true
			default:
				pecho("warn", "Unrecognised option: " + value)
		}
	}
	for index, _ := range runtimeOpt.applicationArgs {
		runtimeOpt.applicationArgs[index] = strings.TrimSuffix(
			runtimeOpt.applicationArgs[index],
			"\n")
	}
	encodedArg, errEncode := json.Marshal(runtimeOpt.applicationArgs)
	if errEncode != nil {
		pecho("warn", "Could not encode arguments as json")
	}
	addEnv("targetArgs=" + string(encodedArg))
	cmdChan <- 1
	fullCmdline := strings.Join(cmdlineArray, ", ")
	pecho("debug", "Full command line: " + fullCmdline)
	pecho("info", "Application arguments: " + strings.Join(runtimeOpt.applicationArgs, ", "))
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

func getVariables() {
	if len(os.Getenv("bwBindPar")) > 0 {
		pecho("warn",
		"The legacy bwBindPar has been deprecated! Please read documents about --expose flags",
		)
		bwBindParMap := map[string]string{
			os.Getenv("bwBindPar"):		os.Getenv("bwBindPar"),
		}
		runtimeOpt.userExpose <- bwBindParMap
	}

	runtimeOpt.userLang = os.Getenv("LANG")
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

func readConf() {
	determineConfPath()
	sessionType := os.Getenv("XDG_SESSION_TYPE")

	switch sessionType {
		case "wayland":
			confOpts.waylandOnly = true
		default:
			confOpts.waylandOnly = false
	}
	confOpts.launchTarget = os.Getenv("launchTarget")
	confOpts.bindNetwork = true
	confOpts.terminateImmediately = true
	confOpts.allowClassicNotifs = true
	confOpts.qt5Compat = true
	confOpts.mountInfo = true

	if len(os.Getenv("launchTarget")) > 0 {
		confOpts.launchTarget = os.Getenv("launchTarget")
	}
	if len(os.Getenv("busLaunchTarget")) > 0 {
		confOpts.busLaunchTarget = os.Getenv("busLaunchTarget")
	}
	confFd, fdErr := os.OpenFile(confOpts.confPath, os.O_RDONLY, 0700)
	if fdErr != nil {
		pecho("crit", "Could not open configuration for reading: " + fdErr.Error())
	}
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
}

func setFirewall() {
	rawDenyList := strings.TrimSpace(confOpts.networkDeny)
	denyList := strings.Split(rawDenyList, " ")

	pecho("debug", "Decoded raw deny list: " + strings.Join(denyList, ", "))

	var sig IncomingSig
	sig.RawDenyList = denyList
	sig.AppID = confOpts.appID
	sig.SandboxEng = "top.kimiblock.portable"
	sig.CgroupNested = "app.slice/app-portable-" + confOpts.appID + "-" + runtimeInfo.instanceID + ".service/portable-cgroup"

	rules := make(chan string, 1)
	ready := make(chan int, 1)

	go dialNetsock(rules, ready)

	if len(rawDenyList) == 0 {
		pecho("debug", "Did not find any network restriction")
		return
	}

	jsonObj, encodeErr := json.Marshal(sig)
	if encodeErr != nil {
		pecho("crit", "Could not decode network restriction list: " + encodeErr.Error())
		return
	}

	rules <- string(jsonObj)
	close(rules)


}

// Copied from netsock
type ResponseSignal struct {
	Success			bool
	Log			string
}

type IncomingSig struct {
	CgroupNested		string
	RawDenyList		[]string
	SandboxEng		string
	AppID			string
}

func dialNetsock(rules chan string, ready chan int) {
	// dialer, errorDial := net.Dial("unix", "/run/netsock/control.sock")
	// if errorDial != nil {
	// 	pecho("warn", "Could not dial netsock socket, firewall disabled")
	// 	ready <- 1
	// 	return
	// }
	transport := http.Transport {
		Proxy:		nil,
		Dial:		func(network, addr string) (net.Conn, error) {return net.Dial("unix", "/run/netsock/control.sock")},
	}

	client := http.Client{
		Transport:	&transport,
		Timeout:	10 * time.Second,
	}

	buf := strings.NewReader(<-rules)
	var resp ResponseSignal

	respPtr, postErr := client.Post("http://127.0.0.114/add", "application/json", buf)
	if postErr != nil {
		pecho("warn", "Could not post data to netsock: " + postErr.Error())
		addEnv("netsockFail=" + "Could not post data to netsock, check netsock.socket enabled and running")
		return
	}
	defer respPtr.Body.Close()

	decoder := json.NewDecoder(respPtr.Body)

	err := decoder.Decode(&resp)
	if err != nil {
		pecho("warn", "Could not decode response from netsock: " + err.Error())
		addEnv("netsockFail=" + "Could not decode response from netsock: " + err.Error())
	}
	if resp.Success == true {
		pecho("debug", "Firewall active")
	} else {
		pecho("warn", "netsock respond with: " + resp.Log)
		addEnv("netsockFail=1" + "netsock respond with: " + resp.Log)
	}
	ready <- 1
}

func mkdirPool(dirs []string) {
	var wg sync.WaitGroup
	for _, dir := range dirs {
		wg.Go(
			func() {
				os.MkdirAll(dir, 0700)
			},
		)
	}
	wg.Wait()
}

func genInstanceID(genInfo chan int8, proceed chan int8) {
	var wg sync.WaitGroup
	pecho("debug", "Generating instance ID")
	for {
		idCandidate := rand.Intn(2147483647)
		pecho("debug", "Trying instance ID: " + strconv.Itoa(idCandidate))
		_, err := os.Stat(xdgDir.runtimeDir + "/.flatpak/" + strconv.Itoa(idCandidate))
		if err == nil {
			pecho("warn", "Unable to use instance ID " + strconv.Itoa(idCandidate))
		} else if idCandidate < 1024 {
			pecho("debug", "Rejecting low ID")
		} else {
			runtimeInfo.instanceID = strconv.Itoa(idCandidate)
			genInfo <- 1
			break
		}
	}
	dirs := []string {
		xdgDir.dataDir + "/" + confOpts.stateDirectory,
		xdgDir.runtimeDir + "/.flatpak/" + confOpts.appID + "/xdg-run",
		xdgDir.runtimeDir + "/.flatpak/" + confOpts.appID + "/tmp",
	}

	wg.Add(1)
	go func () {
		defer wg.Done()
		mkdirPool(dirs)
	} ()

	<- proceed

	wg.Add(3)
	go func () {
		defer wg.Done()
		writeControlFile()
	} ()
	go func () {
		defer wg.Done()
		writeInfoFile(genInfo)
	} ()
	go func () {
		defer wg.Done()
		writeFlatpakRef()
	} ()
	wg.Wait()
}

func writeControlFile() {
	os.MkdirAll(xdgDir.runtimeDir + "/portable/" + confOpts.appID, 0700)
	var controlContent = controlFile
	replacer := strings.NewReplacer(
		"inIdHold",	runtimeInfo.instanceID,
		"idHold",	confOpts.appID,
		"busHold",	xdgDir.runtimeDir + "/app/" + confOpts.appID,
		"friendlyHold",	confOpts.friendlyName,
	)

	os.WriteFile(
		xdgDir.runtimeDir + "/portable/" + confOpts.appID + "/control",
		[]byte(replacer.Replace(controlContent)),
		0700,
	)
}

func writeFlatpakRef() {
	var flatpakRef string = ""
	os.MkdirAll(xdgDir.runtimeDir + "/.flatpak/" + confOpts.appID, 0700)
	os.WriteFile(
		xdgDir.runtimeDir + "/.flatpak/" + confOpts.appID + "/.ref",
		[]byte(flatpakRef),
		0700,
	)
}

func writeInfoFile(ready chan int8) {
	flatpakInfo, err := os.OpenFile("/usr/lib/portable/flatpak-info", os.O_RDONLY, 0600)
	if err != nil {
		pecho("crit", "Failed to read preset Flatpak info")
	}
	defer flatpakInfo.Close()
	infoObj, ioErr := io.ReadAll(flatpakInfo)
	if ioErr != nil {
		pecho("debug", "Failed to read template Flatpak info for I/O error: " + ioErr.Error())
	}
	stringObj := string(infoObj)
	replacer := strings.NewReplacer(
		"placeHolderAppName", confOpts.appID,
		"placeholderInstanceId", runtimeInfo.instanceID,
		"placeholderPath", xdgDir.dataDir + "/" + confOpts.stateDirectory,
	)
	stringObj = replacer.Replace(stringObj)
	os.MkdirAll(xdgDir.runtimeDir + "/.flatpak/" + runtimeInfo.instanceID, 0700)
	os.WriteFile(
		xdgDir.runtimeDir + "/.flatpak/" + runtimeInfo.instanceID + "/info.tmp",
		[]byte(stringObj),
		0700,
	)
	err = os.Rename(
		xdgDir.runtimeDir + "/.flatpak/" + runtimeInfo.instanceID + "/info.tmp",
		xdgDir.runtimeDir + "/.flatpak/" + runtimeInfo.instanceID + "/info",
	)
	if err != nil {
		panic(err)
	}
	os.WriteFile(
		xdgDir.runtimeDir + "/portable/" + confOpts.appID + "/flatpak-info.tmp",
		[]byte(stringObj),
		0700,
	)
	err = os.Rename(
		xdgDir.runtimeDir + "/portable/" + confOpts.appID + "/flatpak-info.tmp",
		xdgDir.runtimeDir + "/portable/" + confOpts.appID + "/flatpak-info",
	)
	if err != nil {
		panic(err)
	}
	ready <- 1
	pecho("debug", "Wrote info file")
}

func getFlatpakInstanceID() {
	if len(runtimeInfo.instanceID) > 0 {
		pecho("debug", "Flatpak instance ID already known")
		return
	}
	controlFile, readErr := os.ReadFile(xdgDir.runtimeDir + "/portable/" + confOpts.appID + "/control")
	instanceID := regexp.MustCompile("instanceId=.*")
	if readErr == nil {
		var rawInstanceID string = string(instanceID.Find(controlFile))
		runtimeInfo.instanceID = tryUnquote(strings.TrimPrefix(rawInstanceID, "instanceId="))
	} else {
		pecho("warn", "Unable to read control file: " + readErr.Error())
	}
	pecho("debug", "Got Flatpak instance ID: " + runtimeInfo.instanceID)
}

func removeWrapCon(paths []string) {
	var wg sync.WaitGroup
	wg.Add(len(paths))
	for _, path := range paths {
		p := path
		go func () {
			err := os.RemoveAll(p)
			wg.Done()
			if err != nil {
				pecho(
					"warn",
					"Unable to remove " + path + ": " + err.Error(),
				)
			}
		} ()
	}
	wg.Wait()
}

func cleanDirs() {
	pecho("info", "Cleaning leftovers")
	var paths = []string{
		xdgDir.runtimeDir + "/portable/" + confOpts.appID,
		xdgDir.runtimeDir + "/app/" + confOpts.appID,
		xdgDir.dataDir + "/applications/" + confOpts.appID + ".desktop",
	}
	getFlatpakInstanceID()
	if len(runtimeInfo.instanceID) > 0 {
		paths = append(
			paths,
			xdgDir.runtimeDir + "/.flatpak/" + confOpts.appID,
			xdgDir.runtimeDir + "/.flatpak/" + runtimeInfo.instanceID,
		)
	} else {
		pecho("warn", "Skipped cleaning Flatpak entries")
	}
	removeWrapCon(paths)
}

func stopAppWorker(conn *dbus.Conn, sdCancelFunc func(), sdContext context.Context) {
	<- stopAppChan
	pecho("debug", "Received a quit request from channel")
	getFlatpakInstanceID()
	var wg sync.WaitGroup
	wg.Add(2)
	go func () {
		defer wg.Done()
		doCleanUnit(conn, sdCancelFunc, sdContext)
	} ()
	go func () {
		defer wg.Done()
		cleanDirs()
	} ()
	close(socketStop)
	//socketStop <- 1
	wg.Wait()
	stopAppDone <- 1
	os.Exit(0)
}

func signalRecvWorker(sigChan chan os.Signal) {
	sig := <- sigChan
	pecho("debug", "Got signal: " + sig.String())
	pecho("info", "Calling stopApp")
	stopApp()

}

func stopApp() {
	stopAppChan <- 1
	<- stopAppDone
}

func lookUpXDG() {
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
		"-u", "app-portable-" + confOpts.appID + "-" + runtimeInfo.instanceID + "-pipewire-container",
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
	argList := []string{}
	argList = append(
		argList,
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

func doCleanUnit(conn *dbus.Conn, sdCancelFunc func(), sdContext context.Context) {
	defer sdCancelFunc()
	var wg sync.WaitGroup
	var units = []string{
		"app-portable-" + confOpts.appID + "-" + runtimeInfo.instanceID + "-pipewire-container.service",
		confOpts.friendlyName + "-" + runtimeInfo.instanceID + "-dbus.service",
	}
	for _, unit := range units {
		wg.Add(1)
		go func (u string) {
			defer wg.Done()
			errBus := conn.KillUnitWithTarget(
				sdContext,
				u,
				dbus.All,
				15,
			)
			if errBus != nil {
				if strings.Contains(u, "-dbus") {
					pecho(
						"warn",
						"User manager returned error: " + errBus.Error(),
					)
				} else {
					pecho(
						"debug",
						"User manager returned error: " + errBus.Error(),
					)
				}
			} else {

			}
		} (unit)
	}
	wg.Wait()
	conn.Close()

	pecho("debug", "Cleaning ready")
}

func startProxy(conn *dbus.Conn, ctx context.Context) {
	dbusProps := []dbus.Property{}
	var wg sync.WaitGroup
	var dbusArgs []string
	var bwInfoPath string = xdgDir.runtimeDir + "/.flatpak/" + runtimeInfo.instanceID + "/bwrapinfo.json"
	type exitStruct struct {
		SIGKILL		[]int32;
		SIGTERM		[]int32;
	}
	var exit exitStruct
	exit.SIGKILL = []int32{9}
	exit.SIGTERM = []int32{15}
	wg.Add(2)
	go func () {
		defer wg.Done()
		dbusArgs = <- busArgChan
	} ()
	wg.Go(func() {
		os.MkdirAll(
			xdgDir.runtimeDir + "/.flatpak/" + runtimeInfo.instanceID,
			0700,
		)
	})
	go func () {
		defer wg.Done()
		os.MkdirAll(
			xdgDir.runtimeDir + "/app/" + confOpts.appID,
			0700,
		)
	} ()
	var unitWants = []string{
		"xdg-document-portal.service",
		"xdg-desktop-portal.service",
	}
	dbusProps = append(
		dbusProps,
		dbus.Property{
			Name: "SuccessExitStatus",
			Value: godbus.MakeVariant(exit),
		},
		dbus.Property{
			Name: "KillMode",
			Value: godbus.MakeVariant("control-group"),
		},
		dbus.Property{
			Name: "StandardError",
			Value: godbus.MakeVariant("file"),
		},
		dbus.Property{
			Name: "StandardErrorFile",
			Value: godbus.MakeVariant(bwInfoPath),
		},
		)
		dbusProps = append(
			dbusProps,
			dbus.PropSlice("portable-" + confOpts.friendlyName + ".slice"),
			dbus.PropWants(unitWants...),
			dbus.PropDescription("D-Bus proxy for portable sandbox " + confOpts.appID),
	)
	wg.Wait()
	dbusProps = append(
		dbusProps,
		dbus.PropExecStart(
			dbusArgs,
			false,
		),
	)
	pecho("info", "Starting D-Bus proxy")

	_, err := conn.StartTransientUnitContext(
		ctx,
		confOpts.friendlyName + "-" + runtimeInfo.instanceID + "-dbus.service",
		"replace",
		dbusProps,
		nil,
	)
	if err != nil {
		pecho("crit", "Could not start D-Bus proxy: " + err.Error())
	}
}

func handleSignal (conn net.Conn) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Text()
		pecho("debug", "Handling signal: " + line)
		switch line {
			case "terminate-now":
				pecho("debug", "Got termination request from socket")
				stopApp()
				return
			default:
				pecho("warn", "Unrecognised signal from socket: " + line)
		}
	}
}

func watchSignalSocket(readyChan chan int8) {
	var signalSocketPath = xdgDir.runtimeDir + "/portable/" + confOpts.appID + "/portable-control/daemon"
	err := os.MkdirAll(xdgDir.runtimeDir + "/portable/" + confOpts.appID + "/portable-control", 0700)
	if err != nil {
		pecho("crit", "Could not create control directory: " + err.Error())
	}
	socket, listenErr := net.Listen("unix", signalSocketPath)
	if listenErr != nil {
		pecho("crit", "Unable to listen for signal: " + listenErr.Error())
		return
	}
	readyChan <- 1
	pecho("debug", "Accepting signals")
	go closeSocketOnDemand(socket)
	for {
		conn, errListen := socket.Accept()
		if errListen != nil {
			pecho("warn", "Could not accept connection: " + errListen.Error())
			break
		}
		go handleSignal(conn)
	}
}

func closeSocketOnDemand (socket net.Listener) {
	<- socketStop
	pecho("debug", "Closing socket on request")
	socket.Close()
}

func startApp() {
	go forceBackgroundPerm()
	pecho("debug", "Calculated arguments for systemd-run: " + strings.Join(runtimeInfo.bwCmd, ", "))
	sdExec := exec.Command("systemd-run", runtimeInfo.bwCmd...)
	sdExec.Stderr = os.Stderr
	sdExec.Stdout = os.Stdout
	sdExec.Stdin = os.Stdin
	<- envsFlushReady
	waitChan(signalWatcherReady, "Signal Watcher")
	// Profiler
	//pprof.Lookup("block").WriteTo(os.Stdout, 1)
	if startAct == "abort" {
		stopApp()
	} else {
		sdExecErr := sdExec.Run()
		if sdExecErr != nil {
			fmt.Println(sdExecErr)
			pecho("warn", "systemd-run returned non 0 exit code")
		}
		stopApp()
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

func instDesktopFile() {
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
}

func setXDGEnvs() {
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
}

func imEnvs () {
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
}

func setupSharedDir () {
	os.MkdirAll(xdgDir.dataDir + "/" + confOpts.stateDirectory + "/Shared", 0700)
	os.Link(
		xdgDir.dataDir + "/" + confOpts.stateDirectory + "/Shared",
		xdgDir.dataDir + "/" + confOpts.stateDirectory + "/共享文件")
}

func miscEnvs () {
	if confOpts.qt5Compat == true {
		addEnv("QT_QPA_PLATFORMTHEME=xdgdesktopportal")
	}
	os.MkdirAll(xdgDir.runtimeDir + "/portable/" + confOpts.appID, 0700)
	var file string = "source " + xdgDir.runtimeDir + "/portable/" + confOpts.appID + "/generated.env"
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
	addEnv("PS1=" + strconv.Quote("╰─>Portable·" + confOpts.appID + "·🤓 ⤔ "))
	addEnv("QT_SCALE_FACTOR=" + os.Getenv("QT_SCALE_FACTOR"))
	addEnv("HOME=" + xdgDir.dataDir + "/" + confOpts.stateDirectory)
	addEnv("XDG_SESSION_TYPE=" + os.Getenv("XDG_SESSION_TYPE"))
	addEnv("WAYLAND_DISPLAY=" + xdgDir.runtimeDir + "/wayland-0")
	addEnv("DBUS_SESSION_BUS_ADDRESS=unix:path=/run/sessionBus")
}

func prepareEnvs() {
	var wg sync.WaitGroup
	wg.Go(func() {
		imEnvs()
	})
	wg.Go(func() {
		setXDGEnvs()
	})
	wg.Go(func() {
		miscEnvs()
	})
	wg.Go(func() {
		userEnvs, err := os.OpenFile(
			xdgDir.dataDir + "/" + confOpts.stateDirectory + "/portable.env",
			os.O_RDONLY,
			0700,
		)
		if err != nil {
			if os.IsNotExist(err) {
				const template = "# This file accepts simple KEY=VAL envs"
				os.WriteFile(
					xdgDir.dataDir + "/" + confOpts.stateDirectory + "/portable.env",
					[]byte(template),
					0700,
				)
			} else {
				pecho(
				"warn",
				"Unable to open file for reading environment variables: " + err.Error(),
				)
			}
		} else {
			defer userEnvs.Close()
			scanner := bufio.NewScanner(userEnvs)
			for scanner.Scan() {
				if scanner.Err() != nil {
					pecho(
					"warn",
					"Could not read user environment variables" + scanner.Err().Error(),
					)
					continue
				}
				line := scanner.Text()
				addEnv(line)
			}
		}
	})

	wg.Go(func() {
		packageEnvs, errPkg := os.OpenFile(confOpts.confPath, os.O_RDONLY, 0700)
		if errPkg != nil {
			pecho(
				"crit",
				"Could not open package config " + confOpts.confPath + ": " + errPkg.Error(),
				)
		} else {
			defer packageEnvs.Close()
			scanner := bufio.NewScanner(packageEnvs)
			for scanner.Scan() {
				if scanner.Err() != nil {
					pecho(
						"crit",
						"Could not read package defined environment variables",
						)
				}
				line := scanner.Text()
				addEnv(line)
			}
		}
	})

	wg.Wait()
}

func genBwArg(
	pwChan chan []string,
	xChan chan []string,
	camChan chan []string,
	inputChan chan []string,
	wayDisplayChan chan []string,
	) {

	miscChan := make(chan []string, 1)
	go miscBinds(miscChan, pwChan)


	if internalLoggingLevel > 1 {
		runtimeInfo.bwCmd = append(runtimeInfo.bwCmd, "--quiet")
	}

	// Define global systemd args first
	runtimeInfo.bwCmd = append(
		runtimeInfo.bwCmd,
		"--user",
		"--pty",
		"--service-type=notify",
		"--wait",
		"--unit=" + "app-portable-" + confOpts.appID + "-" + runtimeInfo.instanceID,
		"--slice=app.slice",
		"-p", "Delegate=yes",
		"-p", "DelegateSubgroup=portable-cgroup",
		"-p", "BindsTo=" + confOpts.friendlyName + "-" + runtimeInfo.instanceID + "-dbus.service",
		"-p", "Description=Portable Sandbox for " + confOpts.friendlyName + " (" + confOpts.appID + ")",
		"-p", "Documentation=https://github.com/Kraftland/portable",
		"-p", "ExitType=cgroup",
		"-p", "SuccessExitStatus=SIGKILL",
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
		"-p", "Environment=instanceId=" + runtimeInfo.instanceID,
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
		"--tmpfs",		"/sys/block",
		"--tmpfs",		"/sys/bus",
		"--ro-bind-try",	"/sys/fs/cgroup", "/sys/fs/cgroup",
		"--dev-bind",		"/sys/class/drm", "/sys/class/drm",
		"--bind-try",		"/sys/devices/system", "/sys/devices/system",
		"--ro-bind",		"/sys/kernel", "/sys/kernel",
		"--ro-bind",		"/sys/devices/virtual", "/sys/devices/virtual",

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
			xdgDir.runtimeDir + "/portable/" + confOpts.appID + "/a11y",
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

	for arg := range miscChan {
		runtimeInfo.bwCmd = append(
			runtimeInfo.bwCmd,
			arg...
		)
	}

	if confOpts.bindInputDevices == true {
		for inputArg := range inputChan {
			runtimeInfo.bwCmd = append(
				runtimeInfo.bwCmd,
				inputArg...
			)
		}
	}


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
		"/usr/lib/portable/helper/helper",
	)
}

func flushEnvs() {
	//os.MkdirAll(xdgDir.runtimeDir + "/portable/" + confOpts.appID, 0700)
	var envs = []string{}

	for env := range envsChan {
		envs = append(
			envs,
			env,
		)
	}

	fd, err := os.OpenFile(
		xdgDir.runtimeDir + "/portable/" + confOpts.appID + "/generated.env",
			os.O_CREATE|os.O_TRUNC|os.O_WRONLY,
			0700,
	)
	if err != nil {
		pecho(
			"crit",
			"Could not open environment variables: " + err.Error(),
		)
	}
	defer fd.Close()
	writer := bufio.NewWriter(fd)
	for _, env := range envs {
		writer.WriteString(env)
		writer.WriteString("\n")
	}
	err = writer.Flush()
	if err != nil {
		pecho("crit", "Could not write environment variables: " + err.Error())
	}
	close(envsFlushReady)
}

func translatePath(input string) (output string) {
	output = strings.ReplaceAll(input, xdgDir.home, xdgDir.dataDir + "/" + confOpts.stateDirectory)
	return
}

func maskDir(path string) (maskArgs []string) {
	maskT, err := os.Stat(path)
	if err == nil && maskT.IsDir() == true {
		pecho("debug", "Masking " + path)
	} else {
		return
	}
	maskArgs = append(
		maskArgs,
		"--tmpfs", path,
	)
	return
}

func miscBinds(miscChan chan []string, pwChan chan []string) {
	var wg sync.WaitGroup
	var miscArgs = []string{}

	wg.Go(func() {
		miscChan <- maskDir("/proc/bus")
	})

	wg.Go(func() {
		miscChan <- maskDir("/proc/driver")
	})

	wg.Go(func() {
		miscChan <- maskDir("/sys/devices/virtual/dmi")
		miscChan <- maskDir("/sys/devices/virtual/block")
		miscChan <- maskDir("/sys/devices/virtual/sound")
	})

	wg.Go(func() {
		_, err := os.Stat("/usr/lib/flatpak-xdg-utils/flatpak-spawn")
		if err == nil {
			miscChan <- []string{
				"--ro-bind",
				"/usr/lib/portable/overlay-usr/flatpak-spawn",
				"/usr/lib/flatpak-xdg-utils/flatpak-spawn",
			}
		}
	})

	wg.Go(func() {
		miscChan <- maskDir("/etc/kernel")
	})

	wg.Go(func() {
		miscChan <- []string{
			"--ro-bind-try",
			xdgDir.confDir + "/fontconfig",
			translatePath(xdgDir.confDir + "/fontconfig"),
		}
	})

	wg.Go(func() {
		miscChan <- []string {
			"--ro-bind-try",
			xdgDir.confDir + "/gtk-3.0/gtk.css",
			translatePath(xdgDir.confDir + "/gtk-3.0/gtk.css"),
		}
	})

	wg.Go(func() {
		miscChan <- []string {
			"--ro-bind-try",
			xdgDir.confDir + "/gtk-3.0/colors.css",
			translatePath(xdgDir.confDir + "/gtk-3.0/colors.css"),
		}
	})

	wg.Go(func() {
		miscChan <- []string {
			"--ro-bind-try",
			xdgDir.confDir + "/gtk-4.0/gtk.css",
			translatePath(xdgDir.confDir + "/gtk-4.0/gtk.css"),
		}
	})

	wg.Go(func() {
		miscChan <- []string{
			"--ro-bind-try",
			xdgDir.confDir + "/qt6ct",
			translatePath(xdgDir.confDir + "/qt6ct"),
		}
	})

	wg.Go(func() {
		miscChan <- []string{
			"--ro-bind-try",
			xdgDir.dataDir + "/fonts",
			translatePath(xdgDir.dataDir + "/fonts"),
		}
	})

	wg.Go(func() {
		miscChan <- []string{
			"--ro-bind-try",
			xdgDir.dataDir + "/icons",
			translatePath(xdgDir.dataDir + "/icons"),
		}
	})

	var validBwBindArgs = make(chan []string, 64)
	var exposedPaths = make(chan string, 16)

	close(runtimeOpt.userExpose)

	var hasValidExpose bool

	for pathMap := range runtimeOpt.userExpose {
		for ori, dest := range pathMap {
			pecho("debug", "Checking " + ori + ", " + dest)
			wg.Go(func() {
				_, err := os.Stat(ori)
				if err != nil {
					pecho("warn", "Could not select origin " + ori + ": " + err.Error())
					return
				}
				hasValidExpose = true
				exposedPaths <- ori
				if strings.HasPrefix(dest, "ro:") {
					validBwBindArgs <- []string{
						"--ro-bind",
						ori,
						strings.TrimSuffix(dest, "ro:"),
					}
				} else if strings.HasPrefix(dest, "dev:") {
					validBwBindArgs <- []string{
						"--dev-bind",
						ori,
						strings.TrimSuffix(dest, "dev:"),
					}
				} else {
					validBwBindArgs <- []string{
						"--bind",
						ori,
						dest,
					}
				}

			})
		}
	}
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
				xdgDir.runtimeDir + "/.flatpak/" + runtimeInfo.instanceID + "-private/run-environ",
			"--ro-bind",
				xdgDir.runtimeDir + "/.flatpak/" + runtimeInfo.instanceID,
				xdgDir.runtimeDir + "/.flatpak/" + runtimeInfo.instanceID,
			"--ro-bind",
				xdgDir.runtimeDir + "/.flatpak/" + runtimeInfo.instanceID,
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

	miscChan <- miscArgs

	wg.Wait()
	if hasValidExpose {
		close(validBwBindArgs)
		close(exposedPaths)
		zenityArgs := []string{
			"--title",
				confOpts.friendlyName,
			"--icon=folder-open-symbolic",
			"--question",
			"--default-cancel",
		}
		var zenityText string
		switch runtimeOpt.userLang {
			case "zh_CN.UTF-8":
				zenityText = "--text=暴露以下路径\n "
			default:
				zenityText = "--text=Exposing the following path\n "
		}
		var invokeZenity bool
		for path := range exposedPaths {
			invokeZenity = true
			zenityText = zenityText + path + " \n"
		}
		if invokeZenity {
			zenityArgs = append(zenityArgs, zenityText)
			zenityRun := exec.Command("/usr/bin/zenity", zenityArgs...)
			zenityRun.Stderr = os.Stderr
			errZenity := zenityRun.Run()
			if errZenity != nil {
				pecho("warn", "User denied exposing")
				//stopApp()
			} else {
				for arg := range validBwBindArgs {
					miscChan <- arg
				}
			}
		}
	}

	close(miscChan)

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

func detectCardStatus(cardList chan []string, cardPath string, cardNamed string) {
	connectors, err := os.ReadDir(cardPath)
	if err != nil {
		pecho(
			"warn",
			"Failed to read GPU connector status: " + err.Error(),
		)
		return
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
			var activeGpus = []string{cardNamed}
			cardList <- activeGpus
			pecho("debug", "Found active GPU: " + cardNamed)
			return
		}
	}
}

func gpuBind(gpuChan chan []string) {
	u := udev.Udev{}
	e := u.NewEnumerate()
	e.AddMatchIsInitialized()
	e.AddMatchSubsystem("drm")
	devs, errUdev := e.Devices()
	if errUdev != nil {
		pecho("warn", "Failed to query udev for GPU info")
	}
	var wg sync.WaitGroup
	var gpuArg = []string{}
	// SHOULD contain strings like card0, card1 etc
	var totalGpus = []string{}
	var activeGpus = []string{}
	var cardSums int = 0
	var cardList = make(chan []string, 512)
	var cardPaths []string

	wg.Add(1)
	go func () {
		defer wg.Done()
		gpuArg = append(
			gpuArg,
			"--tmpfs", "/dev/dri",
			"--tmpfs", "/sys/class/drm",
		)
		for _, path := range nvKernelModulePath {
			gpuArg = append(
				gpuArg,
				maskDir(path)...
			)
		}
	} ()


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
		cardSums++
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
	var argChan = make(chan []string, 128)
	switch cardSums {
		case 0:
			pecho("warn", "Found no GPU")
		default:
			if confOpts.gameMode == true {
				wg.Add(1)
				go func () {
					defer wg.Done()
					setOffloadEnvs()
				} ()
				for _, cardName := range totalGpus {
					wg.Add(1)
					go func (card string, arg chan []string) {
						defer wg.Done()
						bindCard(card, arg)
					} (cardName, argChan)
				}
				wg.Go(
					func() {argChan <- tryBindNv()},
				)
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
						bindCard(card, argChan)
					} (cardName)
				}
			}
	}
	go func () {
		wg.Wait()
		close(argChan)
	} ()
	for arg := range argChan {
		gpuArg = append(
			gpuArg,
			arg...
		)
	}
	gpuChan <- gpuArg
	close(gpuChan)
	var activeGPUList string = strings.Join(activeGpus, ", ")

	// TODO: Drop the debug output below
	pecho("debug", "Generated GPU bind parameters: " + strings.Join(gpuArg, ", "))
	pecho(
	"debug",
	"Total GPU count " + strconv.Itoa(cardSums) + ", active: " + activeGPUList)
}

func setOffloadEnvs() () {
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
}

func bindCard(cardName string, argChan chan []string) {
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
		}
		cardVendor, err := io.ReadAll(cardVendorFd)
		if err != nil {
			pecho("warn", "Failed to parse GPU vendor: " + err.Error())
		}
		if strings.Contains(string(cardVendor), "0x10de") == true {
			pecho("debug", "Found NVIDIA device")
			if confOpts.useZink == true {
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

func tryBindNv() []string {
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
	return nvDevsArg
}

func inputBind(inputBindChan chan []string) {
	var wg sync.WaitGroup
	inputBindArg := []string{}
	inputBindArg = append(
		inputBindArg,
		"--dev-bind-try",	"/sys/class/leds", "/sys/class/leds",
		"--dev-bind-try",	"/sys/class/input", "/sys/class/input",
		"--dev-bind-try",	"/sys/class/hidraw", "/sys/class/hidraw",
		"--dev-bind-try",	"/dev/input", "/dev/input",
		"--dev-bind-try",	"/dev/uinput", "/dev/uinput",
	)

	u := udev.Udev{}
	e := u.NewEnumerate()

	var devArgChan = make(chan []string, 512)

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
			sysSl := strings.SplitN(path, "/", -1)
			sliceLen := len(sysSl)
			if strings.HasPrefix(sysSl[sliceLen - 1], "event") {
				if strings.HasPrefix(sysSl[sliceLen - 2], "input") {
					path = strings.Join(sysSl[0:sliceLen - 3], "/")
				}
			}
			devArgChan <- []string{
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
				devArgChan <- []string{
					"--dev-bind",
					devPath,
					devPath,
				}
			}
			if len(path) > 0 {
				sysPathSlice := strings.SplitN(path, "/", -1)
				sysPathSliceLen := len(sysPathSlice)
				if strings.Contains(sysPathSlice[sysPathSliceLen - 2], "hidraw") {
					path = strings.Join(sysPathSlice[0:sysPathSliceLen - 3], "/")
				}
				devArgChan <- []string{
					"--dev-bind",
					path,
					path,
				}
			}

		} (dev)
	}

	go func () {
		wg.Wait()
		close(devArgChan)
	} ()


	for content := range devArgChan {
		inputBindArg = append(
			inputBindArg,
			content...
		)
	}

	inputBindChan <- inputBindArg
	close(inputBindChan)
	pecho("debug", "Finished calculating input arguments: " + strings.Join(inputBindArg, " "))
}

func multiInstance(miChan chan bool) {
	var socketPath string = xdgDir.runtimeDir + "/portable/"
	socketPath = socketPath + confOpts.appID + "/portable-control/daemon"
	_, errStat := os.Stat(socketPath)
	if errStat != nil && os.IsNotExist(errStat) {
		miChan <- false
		return
	}
	pecho("debug", "Dialing daemon socket...")
	_, err := net.Dial("unix", socketPath)
	socketPath = xdgDir.runtimeDir + "/portable/"
	socketPath = socketPath + confOpts.appID + "/portable-control/auxStart"

	if err != nil {
		if runtimeOpt.miTerminate == true {
			pecho("warn", "Could not find running instance")
			os.Exit(2)
		}
		miChan <- false
		return
	} else {
		pecho("debug", "Another instance running")
		if runtimeOpt.miTerminate == true {
			var daemonPath string = xdgDir.runtimeDir + "/portable/"
			daemonPath = daemonPath + confOpts.appID + "/portable-control/daemon"
			const signal = "terminate-now" + "\n"
			controlSock, dialErr := net.Dial("unix", daemonPath)
			if dialErr != nil {
				pecho("crit", "Could not dial control daemon: " + dialErr.Error())
			}
			_, errWrite := controlSock.Write([]byte(signal))
			if errWrite != nil {
				pecho("crit", "Could not send termination signal: " + errWrite.Error())
			}
			os.Exit(0)
		}
		startAct = "aux"
	}
	if confOpts.dbusWake == true {
		pecho("debug", "Trying to resolve tray ID")
		queryTrayArg := []string{
			"--bus=unix:path=" + xdgDir.runtimeDir + "/app/" + confOpts.appID + "/bus",
			"--dest=org.kde.StatusNotifierWatcher",
			"--type=method_call",
			"--print-reply=literal",
			"/StatusNotifierWatcher",
			"org.freedesktop.DBus.Properties.Get",
			"string:org.kde.StatusNotifierWatcher",
			"string:RegisteredStatusNotifierItems",
		}
		out, err := exec.Command("/usr/bin/dbus-send", queryTrayArg...).CombinedOutput()
		if err != nil {
			pecho("crit", "Could not get tray ID: " + err.Error())
		}
		re := regexp.MustCompile(`org\.kde\.StatusNotifierItem-[0-9-]+`)
		output := string(out)
		match := re.FindString(output)
		trayID := match

		if len(trayID) > 0 {
			pecho("debug", "Got tray ID: " + trayID)
			wakeArgs := []string{
				"--print-reply",
				"--session",
				"--dest=" + trayID,
				"--type=method_call",
				"/StatusNotifierItem",
				"org.kde.StatusNotifierItem.Activate",
				"int32:114514",
				"int32:1919810",
			}
			wakeCmd := exec.Command("dbus-send", wakeArgs...)
			wakeCmd.Stderr = os.Stderr
			wakeCmd.Run()
		} else {
			pecho("crit", "Unable to get tray ID")
		}
	} else {
		startJson, jsonErr := json.Marshal(runtimeOpt.applicationArgs)
		if jsonErr != nil {
			pecho("warn", "Could not marshal application args: " + jsonErr.Error())
		}
		socket, errDial := net.Dial("unix", socketPath)
		if errDial != nil {
			pecho("crit", "Could not dial socket: " + errDial.Error())
		}
		defer socket.Close()
		_, wrErr := socket.Write(startJson)
		if wrErr != nil {
			pecho("crit", "Could not write signal: " + wrErr.Error())
		} else {
			pecho("debug", "Wrote signal: " + string(startJson))
		}
	}
	miChan <- true
}

func atSpiProxy() {
	_, err := os.Stat(xdgDir.runtimeDir + "/at-spi/bus")
	if err != nil {
		pecho("warn", "Could not detect accessibility bus: " + err.Error())
		return
	}
	err = os.MkdirAll(xdgDir.runtimeDir + "/portable/" + confOpts.appID + "/a11y", 0700)
	if err != nil {
		pecho("warn", "Could not create directory for accessibility bus: " + err.Error())
		return
	}
	atspiArgs := []string{
		"--symlink", "/usr/lib64", "/lib64",
		"--ro-bind", "/usr/lib", "/usr/lib",
		"--ro-bind", "/usr/lib64", "/usr/lib64",
		"--ro-bind", "/usr/bin", "/usr/bin",
		"--ro-bind", "/usr/share", "/usr/share",
		"--bind", xdgDir.runtimeDir + "/at-spi", xdgDir.runtimeDir + "/at-spi",
		"--bind",
			xdgDir.runtimeDir + "/portable/" + confOpts.appID + "/a11y",
			xdgDir.runtimeDir + "/portable/" + confOpts.appID + "/a11y",
		"--ro-bind",
			xdgDir.runtimeDir + "/portable/" + confOpts.appID + "/flatpak-info",
			xdgDir.runtimeDir + "/.flatpak-info",
		"--ro-bind",
			xdgDir.runtimeDir + "/portable/" + confOpts.appID + "/flatpak-info",
			"/.flatpak-info",
		"--",
		"/usr/bin/xdg-dbus-proxy",
		"unix:path=" + xdgDir.runtimeDir + "/at-spi/bus",
		xdgDir.runtimeDir + "/portable/" + confOpts.appID + "/a11y/bus",
		"--filter",
		"--sloppy-names",
		"--call=org.a11y.atspi.Registry=org.a11y.atspi.Socket.Embed@/org/a11y/atspi/accessible/root",
		"--call=org.a11y.atspi.Registry=org.a11y.atspi.Socket.Unembed@/org/a11y/atspi/accessible/root",
		"--call=org.a11y.atspi.Registry=org.a11y.atspi.Registry.GetRegisteredEvents@/org/a11y/atspi/registry",
		"--call=org.a11y.atspi.Registry=org.a11y.atspi.DeviceEventController.GetKeystrokeListeners@/org/a11y/atspi/registry/deviceeventcontroller",
		"--call=org.a11y.atspi.Registry=org.a11y.atspi.DeviceEventController.GetDeviceEventListeners@/org/a11y/atspi/registry/deviceeventcontroller",
		"--call=org.a11y.atspi.Registry=org.a11y.atspi.DeviceEventController.NotifyListenersSync@/org/a11y/atspi/registry/deviceeventcontroller",
		"--call=org.a11y.atspi.Registry=org.a11y.atspi.DeviceEventController.NotifyListenersAsync@/org/a11y/atspi/registry/deviceeventcontroller",
	}

	atSpiProxyCmd := exec.Command("bwrap", atspiArgs...)
	atSpiProxyCmd.Stderr = os.Stderr
	if internalLoggingLevel <= 1 {
		atSpiProxyCmd.Stdout = os.Stdout
	}

	sysattr := syscall.SysProcAttr{
		Pdeathsig:		syscall.SIGKILL,
	}
	atSpiProxyCmd.SysProcAttr = &sysattr

	atSpiProxyCmd.Start()
}

func waitChan(tgChan chan int8, chanName string) {
	<- tgChan
}

func main() {
	runtimeOpt.userExpose = make(chan map[string]string, 2048)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	go signalRecvWorker(sigChan)
	go pechoWorker()

	var wg sync.WaitGroup
	go getVariables()
	wg.Add(1)
	go func () {
		defer wg.Done()
		readConf()
	} ()
	sdContext, sdCancelFunc := context.WithCancel(context.Background())
	conn, err := dbus.NewUserConnectionContext(sdContext)
	if err != nil {
		pecho("crit", "Could not connect to user service manager: " + err.Error())
		return
	}
	wg.Add(1)
	go func () {
		defer wg.Done()
		lookUpXDG()
	} ()
	go stopAppWorker(conn, sdCancelFunc, sdContext)

	inputChan := make(chan []string, 512)
	go inputBind(inputChan) // This is fine, since genBwArg takes care of conf switching
	fmt.Println("Portable daemon", version, "starting")
	cmdChan := make(chan int8, 1)
	wg.Wait()

	wayDisplayChan := make(chan[]string, 1)
	go waylandDisplay(wayDisplayChan)

	go cmdlineDispatcher(cmdChan)
	go gpuBind(gpuChan)
	wg.Add(1)
	go func() {
		defer wg.Done()
		instDesktopFile()
	} ()
	wg.Go(func() {
		setupSharedDir()
	})
	genChan := make(chan int8, 2) /* Signals when an ID has been chosen,
		and we signal back when multi-instance is cleared
		in another channel */
	genChanProceed := make(chan int8, 1)
	wg.Add(1)
	go func () {
		defer wg.Done()
		genInstanceID(genChan, genChanProceed)
	} ()
	xChan := make(chan []string, 1)
	go bindXAuth(xChan)
	camChan := make(chan []string, 1)
	go tryBindCam(camChan)

	<- cmdChan


	if startAct == "abort" {
		pecho("warn", "Aborting start")
		//stopApp()
		return
	}


	miChan := make(chan bool, 1)

	// MI
	go multiInstance(miChan)
	wg.Add(1)
	go func () {
		defer wg.Done()
		sanityChecks()
	} ()
	pwSecContextChan := make(chan []string, 1)

	wg.Add(2)
	go func() {
		defer wg.Done()
		prepareEnvs()
	} ()

	if multiInstanceDetected := <- miChan; multiInstanceDetected == true {
		startAct = "aux"
	} else {
	go flushEnvs()
	go setFirewall()
	go watchSignalSocket(signalWatcherReady)
	<- genChan // Stage one, ensures that IDs are actually present
	go func () {
		defer wg.Done()
		genBwArg(pwSecContextChan, xChan, camChan, inputChan, wayDisplayChan)
	} ()
	genChanProceed <- 1
	go calcDbusArg(busArgChan)
	wg.Add(2)
	<- genChan // Stage 2, ensures that info file is ready
	go func() {
		defer wg.Done()
		startProxy(conn, sdContext)
	} ()
	go func() {
		defer wg.Done()
		atSpiProxy()
	} ()

	go pwSecContext(pwSecContextChan)
	wg.Wait()
	close(envsChan)
	startApp()
	}
}