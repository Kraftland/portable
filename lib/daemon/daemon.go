package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"github.com/coreos/go-systemd/v22/dbus"
	sdutil "github.com/coreos/go-systemd/v22/util"
	godbus "github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	udev "github.com/jochenvg/go-udev"
)

func sanityChecks(config Config) {
	if sdutil.IsRunningSystemd() == false {
		pecho("crit", "Portable requires the systemd service manager")
	}
	var appIDValid bool = true
	if len("top.kimiblock.portable." + config.Metadata.AppID) > 255 {
		appIDValid = false
		pecho("warn", "Application ID too long")
	}
	if strings.Contains(config.Metadata.AppID, "org.freedesktop.impl") == true {
		appIDValid = false
	} else if strings.Contains(config.Metadata.AppID, "org.gtk.vfs") == true {
		appIDValid = false
	} else if config.Metadata.AppID == "org.mpris.MediaPlayer2" {
		appIDValid = false
	} else if len(config.Metadata.AppID) == 0 {
		appIDValid = false
	} else if len(strings.Split(config.Metadata.AppID, ".")) < 2 {
		appIDValid = false
	}
	if appIDValid == false {
		abortChan <- true
		pecho("crit", "Invalid appID: " + config.Metadata.AppID)
	}
	if len(config.Metadata.FriendlyName) == 0 {
		pecho("crit", "Could not parse friendlyName")
	}
	if len(config.Metadata.StateDirectory) == 0 {
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
	stdout, errP := mountCheckCmd.Output()
	if errP != nil {
		pecho("crit", "Failed to pipe findmnt output: " + errP.Error())
		select {}
	}
	scanner := bufio.NewScanner(strings.NewReader(string(stdout)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "/usr/bin/") {
			abortChan <- true
			pecho("crit", "Found mount points inside /usr/bin")
		}
	}
}

func addEnv(envToAdd string) {
	envsChan <- envToAdd
}

func bytesToMb(bytes int) float64 {
	var div float64 = 1024 * 1024
	return float64(bytes) / div
}

func getVariables(exposeChan chan map[string]string) {
	runtimeOpt.userLang = os.Getenv("LANG")
	bindVar := os.Getenv("bwBindPar")
	if len(bindVar) > 0 {
		bindVar, err := filepath.Abs(bindVar)
		if err != nil {
			pecho("warn", "Could not resolve absolute path: " + err.Error())
			return
		}
		pecho("warn",
		"The legacy bwBindPar has been deprecated! Please read documents about --expose flags",
		)

		bwBindParMap := map[string]string{
			bindVar:		bindVar,
		}
		exposeChan <- bwBindParMap
	}
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

func setFirewall(config Config) {
	var wg sync.WaitGroup
	denyList := config.Network.FilterDest
	if ! config.Network.Filter {
		pecho("debug", "Network filtering disabled")
		return
	}

	pecho("debug", "Decoded raw deny list: " + strings.Join(denyList, ", "))
	type IncomingSig struct {
		CgroupNested		string
		RawDenyList		[]string
		SandboxEng		string
		AppID			string
	}
	var sig IncomingSig
	sig.RawDenyList = denyList
	sig.AppID = config.Metadata.AppID
	sig.SandboxEng = "top.kimiblock.portable"
	sig.CgroupNested = "app.slice/app-portable-" + config.Metadata.AppID + "-" + runtimeInfo.instanceID + ".service/portable-cgroup"

	rules := make(chan string, 1)
	wg.Go(func() {
		go dialNetsock(rules)
	})



	jsonObj, encodeErr := json.Marshal(sig)
	if encodeErr != nil {
		pecho("crit", "Could not decode network restriction list: " + encodeErr.Error())
		return
	}

	rules <- string(jsonObj)
	close(rules)
	wg.Wait()


}

func dialNetsock(rules chan string) {
	// Copied from netsock
	type ResponseSignal struct {
		Success			bool
		Log			string
	}
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
		return
	}
	defer respPtr.Body.Close()

	decoder := json.NewDecoder(respPtr.Body)

	err := decoder.Decode(&resp)
	if err != nil {
		pecho("warn", "Could not decode response from netsock: " + err.Error())
		return
	}
	if resp.Success == true {
		pecho("debug", "Firewall active")
	} else {
		pecho("warn", "netsock respond with: " + resp.Log)
	}
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

func genInstanceID(genInfo chan int8, proceed chan int8, config Config) {
	var wg sync.WaitGroup
	pecho("debug", "Generating instance ID")
	for {
		idCandidate := rand.Intn(2147483647)
		pecho("debug", "Trying instance ID: " + strconv.Itoa(idCandidate))
		_, err := os.Stat(xdgDir.runtimeDir + "/.flatpak/" + strconv.Itoa(idCandidate))
		if os.IsNotExist(err) {
			runtimeInfo.instanceID = strconv.Itoa(idCandidate)
			genInfo <- 1
			break
		} else if err != nil {
			pecho("crit", "Could not stat instance path:", err)
		} else {
			pecho("warn", "Unable to use instance ID " + strconv.Itoa(idCandidate))
		}
	}
	dirs := []string {
		filepath.Join(xdgDir.dataDir, config.Metadata.StateDirectory),
		filepath.Join(xdgDir.runtimeDir, ".flatpak", config.Metadata.AppID, "xdg-run"),
		filepath.Join(xdgDir.runtimeDir, "/.flatpak/", config.Metadata.AppID, "tmp"),
	}

	wg.Go(func() {
		err := os.MkdirAll(filepath.Join(xdgDir.runtimeDir, "portable", config.Metadata.AppID, "portable-control"), 0700)
		if err != nil {
			pecho("crit", "Could not create control directory: " + err.Error())
		}
	})

	wg.Go(func() {
		mkdirPool(dirs)
	})

	<- proceed
	wg.Go(func() {
		generatePasswdFile(config)
	})
	wg.Go(func() {
		generateNsswitch(config)
	})
	wg.Add(2)
	go func () {
		defer wg.Done()
		writeInfoFile(genInfo, config)
	} ()
	go func () {
		defer wg.Done()
		writeFlatpakRef(config)
	} ()
	wg.Wait()
}

func writeFlatpakRef(config Config) {
	var flatpakRef string = ""
	os.MkdirAll(filepath.Join(xdgDir.runtimeDir, ".flatpak", config.Metadata.AppID), 0700)
	os.WriteFile(
		filepath.Join(xdgDir.runtimeDir, ".flatpak", config.Metadata.AppID, ".ref"),
		[]byte(flatpakRef),
		0700,
	)
}

func writeInfoFile(ready chan int8, config Config) {
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
		"placeHolderAppName",		config.Metadata.AppID,
		"placeholderInstanceId",	runtimeInfo.instanceID,
		"placeholderPath",		filepath.Join(xdgDir.dataDir, config.Metadata.StateDirectory),
	)
	stringObj = replacer.Replace(stringObj)
	err = os.MkdirAll(filepath.Join(xdgDir.runtimeDir, ".flatpak", runtimeInfo.instanceID), 0700)
	if err != nil {
		pecho("crit", "Could not create .flatpak path: " + err.Error())
	}
	err = os.WriteFile(
		filepath.Join(xdgDir.runtimeDir, ".flatpak", runtimeInfo.instanceID, "info.tmp"),
		[]byte(stringObj),
		0700,
	)
	if err != nil {
		pecho("crit", "Could not create .flatpak path: " + err.Error())
	}
	err = os.Rename(
		filepath.Join(xdgDir.runtimeDir, ".flatpak", runtimeInfo.instanceID, "info.tmp"),
		filepath.Join(xdgDir.runtimeDir, ".flatpak", runtimeInfo.instanceID, "info"),
	)
	if err != nil {
		pecho("crit", "Could not create .flatpak path: " + err.Error())
	}
	err = os.WriteFile(
		filepath.Join(xdgDir.runtimeDir, "portable", config.Metadata.AppID, "flatpak-info.tmp"),
		[]byte(stringObj),
		0700,
	)
	if err != nil {
		pecho("crit", "Could not create .flatpak path: " + err.Error())
	}
	err = os.Rename(
		filepath.Join(xdgDir.runtimeDir, "portable", config.Metadata.AppID, "flatpak-info.tmp"),
		filepath.Join(xdgDir.runtimeDir, "portable", config.Metadata.AppID, "flatpak-info"),
	)
	if err != nil {
		pecho("crit", "Could not create .flatpak path: " + err.Error())
	}
	ready <- 1
	pecho("debug", "Wrote info file")
}

func removeWrapChan(pathChan chan string) {
	var wg sync.WaitGroup
	for path := range pathChan {
		wg.Go(func() {
			err := os.RemoveAll(path)
			if err != nil {
				pecho(
					"warn",
					"Unable to remove " + path + ": " + err.Error(),
				)
			}
		})
	}
	wg.Wait()
}

func cleanDirs(config Config) {
	var wg sync.WaitGroup
	pathChan := make(chan string, 8)
	wg.Go(func() {removeWrapChan(pathChan)})
	pecho("info", "Cleaning leftovers")
	if len(config.Metadata.AppID) == 0 {
		return
	}
	pathChan <- filepath.Join(
		xdgDir.runtimeDir,
		"/portable/",
		config.Metadata.AppID,
	)
	pathChan <- filepath.Join(
		xdgDir.runtimeDir,
		"app",
		config.Metadata.AppID,
	)
	if runtimeOpt.writtenDesktop {
		pathChan <- filepath.Join(
			xdgDir.dataDir,
			"applications",
			config.Metadata.AppID + ".desktop",
		)
	}
	localID := runtimeInfo.instanceID
	if len(localID) == 0 {
		localID = runtimeInfo.instanceID
	}
	if len(localID) > 0 {
		pathChan <- filepath.Join(
			xdgDir.runtimeDir,
			".flatpak",
			config.Metadata.AppID,
		)
		pathChan <- filepath.Join(
			xdgDir.runtimeDir,
			".flatpak",
			localID,
		)
	} else {
		pecho("warn", "Skipped cleaning Flatpak entries")
	}
	close(pathChan)
	wg.Wait()
}

func stopAppWorker(conn *dbus.Conn, sdCancelFunc func(), sdContext context.Context, busconn *godbus.Conn, config Config) {
	<- stopAppChan
	pecho("debug", "Received a quit request from channel")
	var wg sync.WaitGroup
	wg.Go(func() {
		if busconn == nil {
			pecho("warn", "Race detected: bus already terminated")
			return
		}
		reply, err := busconn.ReleaseName("top.kimiblock.portable." + config.Metadata.AppID)
		if err != nil {
			pecho("warn", "Could not request bus to release name: " + err.Error())
			switch reply {
				case godbus.ReleaseNameReplyReleased:
					pecho("debug", "Successfully released bus name")
				default:
					pecho("warn", "Could not release D-Bus name: " + reply.String())
			}
		}
	})
	wg.Go(func() {
		doCleanUnit(conn, sdCancelFunc, sdContext, config)
	})
	wg.Go(func() {
		cleanDirs(config)
	})
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

func pwSecContext(pwChan chan []string, config Config) {
	var pwProxySocket string
	if config.Privacy.PipeWire == false {
		close(pwChan)
		return
	}
	pwSecCmd := []string{
		"stdbuf",
		"-oL",
		"/usr/bin/pw-container",
		"-P",
		`{ "pipewire.sec.engine": "top.kimiblock.portable", "pipewire.access": "restricted" }`,
	}
	pwSecRun := exec.Command(pwSecCmd[0], pwSecCmd[1:]...)
	pwSecRun.Stderr = os.Stderr
	procAttrs := &syscall.SysProcAttr{
		Pdeathsig:		syscall.SIGKILL,
	}
	pwSecRun.SysProcAttr = procAttrs
	stdout, pipeErr := pwSecRun.StdoutPipe()

	scanner := bufio.NewScanner(stdout)
	err := pwSecRun.Start()
	if err != nil {
		pecho("warn", "Failed to start up PipeWire proxy: " + err.Error() + pipeErr.Error())
		close(pwChan)
		return
	}
	for scanner.Scan() {
		stringObj, found := strings.CutPrefix(scanner.Text(), "new socket: ")
		if ! found {
			continue
		} else {
			pwProxySocket = stringObj
			break
		}
	}
	pwChan <- []string{
		"--bind",
		pwProxySocket,
		filepath.Join(xdgDir.runtimeDir, "pipewire-0"),
	}
	close(pwChan)
	pecho("debug", "pw-container available at " + pwProxySocket)
	err = pwSecRun.Wait()
	if err != nil {
		pecho("warn", "Could not wait pw-container: " + err.Error())
	}
}

func calcDbusArg(argChan chan []string, config Config) {
	argList := []string{
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
			filepath.Join(xdgDir.runtimeDir, "portable", config.Metadata.AppID, "flatpak-info"),
			filepath.Join(xdgDir.runtimeDir, ".flatpak-info"),
		"--ro-bind",
			filepath.Join(xdgDir.runtimeDir, "portable", config.Metadata.AppID, "flatpak-info"),
			"/.flatpak-info",
		"--",
		"/usr/bin/xdg-dbus-proxy",
		os.Getenv("DBUS_SESSION_BUS_ADDRESS"),
		filepath.Join(xdgDir.runtimeDir, "app", config.Metadata.AppID, "/bus"),
		"--filter",
		"--own=com.belmoussaoui.ashpd.demo",
		"--talk=org.unifiedpush.Distributor.*",
		"--own=" + config.Metadata.AppID,
		"--own=" + config.Metadata.AppID + ".*",
		"--talk=com.canonical.AppMenu.Registrar",
		"--see=org.a11y.Bus",
		"--call=org.a11y.Bus=org.a11y.Bus.GetAddress@/org/a11y/bus",
		"--call=org.a11y.Bus=org.freedesktop.DBus.Properties.Get@/org/a11y/bus",
		"--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Request.*@/org/freedesktop/portal/desktop/request/*",
		"--call=org.freedesktop.portal.Desktop=*@/org/freedesktop/portal/desktop/session/*",

		"--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Account.*@/org/freedesktop/portal/desktop",
		"--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Camera.*@/org/freedesktop/portal/desktop",
		"--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Clipboard.*@/org/freedesktop/portal/desktop",
		"--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Email.*@/org/freedesktop/portal/desktop",
		"--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.FileChooser.*@/org/freedesktop/portal/desktop",
		"--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Location.*@/org/freedesktop/portal/desktop",
		"--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.InputCapture.*@/org/freedesktop/portal/desktop",
		"--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.MemoryMonitor.*@/org/freedesktop/portal/desktop",
		"--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.NetworkMonitor.*@/org/freedesktop/portal/desktop",
		"--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Notification.*@/org/freedesktop/portal/desktop",
		"--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.OpenURI.*@/org/freedesktop/portal/desktop",
		"--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.PowerProfileMonitor.*@/org/freedesktop/portal/desktop",
		"--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Print.*@/org/freedesktop/portal/desktop",
		"--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.ProxyResolver.*@/org/freedesktop/portal/desktop",
		"--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.RemoteDesktop.*@/org/freedesktop/portal/desktop",
		"--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.ScreenCast.*@/org/freedesktop/portal/desktop",
		"--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Screenshot.*@/org/freedesktop/portal/desktop",
		"--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Secret.*@/org/freedesktop/portal/desktop",
		"--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Settings.*@/org/freedesktop/portal/desktop",
		"--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Trash.*@/org/freedesktop/portal/desktop",
		"--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Usb.*@/org/freedesktop/portal/desktop",
		"--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Wallpaper.*@/org/freedesktop/portal/desktop",
		"--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.GlobalShortcuts.*@/org/freedesktop/portal/desktop",

		"--call=org.freedesktop.portal.Desktop=org.freedesktop.DBus.Properties.*@/org/freedesktop/portal/desktop/*",

		"--broadcast=org.freedesktop.portal.*=@/org/freedesktop/portal/*",

		"--call=top.kimiblock.portable." + config.Metadata.AppID + "=top.kimiblock.Portable.Controller.Stop@/top/kimiblock/portable/daemon",
		"--broadcast=top.kimiblock.portable." + config.Metadata.AppID + "=top.kimiblock.Portable.Controller.AuxStart@/top/kimiblock/portable/daemon",
		"--call=top.kimiblock.portable." + config.Metadata.AppID + "=top.kimiblock.Portable.IPC.*@/top/kimiblock/portable/IPC",

		// FileManager1 endpoints
		"--call=org.freedesktop.FileManager1=org.freedesktop.FileManager1.*@/org/freedesktop/FileManager1",

		"--call=org.kde.StatusNotifierWatcher=*@/StatusNotifierWatcher",
		"--broadcast=org.kde.StatusNotifierWatcher=*@/StatusNotifierWatcher",

	}

	if config.Advanced.KDEStatus {
		argList = append(argList,
			"--call=org.kde.JobViewServer=org.kde.JobViewServerV2.requestView@/JobViewServer", // This is for adding jobs to KDE
			"--call=org.kde.JobViewServer=org.kde.JobViewV3.update@/org/kde/notificationmanager/jobs/*",
			"--call=org.kde.JobViewServer=org.kde.JobViewV3.terminate@/org/kde/notificationmanager/jobs/*",
		)
	}

	pecho("debug", "Expanding built-in rules")

	allowedTalks := []string{
		"org.freedesktop.portal.Documents",
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

	if internalLoggingLevel <= 1 {
		argList = append(argList, "--log")
	}
	err := os.MkdirAll(xdgDir.runtimeDir + "/doc/by-app/" + config.Metadata.AppID, 0700)
	if err != nil {
		pecho("crit", "Could not create documents path: " + err.Error())
	}

	// Shitty MPRIS calc code
	mprisOwnList := []string{}
	/* Take an app ID top.kimiblock.test for example
		appIDSplit would have 3 substrings
		appIDSepNum would be 3
		so appIDSplit[3 - 1] should be the last part
	*/
	appIDSplit := strings.Split(config.Metadata.AppID, ".")
	appIDSegNum := len(appIDSplit)
	var appIDLastSeg string = appIDSplit[appIDSegNum - 1]
	mprisOwnList = append(
		mprisOwnList,
		"--own=org.mpris.MediaPlayer2." + config.Metadata.AppID,
		"--own=org.mpris.MediaPlayer2." + config.Metadata.AppID + ".*",
		"--own=org.mpris.MediaPlayer2." + appIDLastSeg,
		"--own=org.mpris.MediaPlayer2." + appIDLastSeg + ".*",
	)
	if len(config.Advanced.MprisName) == 0 {
		pecho("debug", "Using default MPRIS own name")
	} else {
		for _, name := range config.Advanced.MprisName {
			mprisOwnList = append(
				mprisOwnList,
				"--own=org.mpris.MediaPlayer2." + name,
				"--own=org.mpris.MediaPlayer2." + name + ".*",
			)
		}

	}

	if config.Privacy.ClassicNotifications {
		argList = append(
			argList,
			"--broadcast=org.freedesktop.Notifications=org.freedesktop.Notifications.*@/org/freedesktop/Notification",
			"--call=org.freedesktop.Notifications=org.freedesktop.Notifications.*@/org/freedesktop/Notifications",
		)
	}

	if config.System.InhibitSuspend {
		argList = append(
			argList,
			"--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Inhibit.*@/org/freedesktop/portal/desktop",
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

func doCleanUnit(conn *dbus.Conn, sdCancelFunc func(), sdContext context.Context, config Config) {
	defer sdCancelFunc()
	var wg sync.WaitGroup
	var units = []string{
		config.Metadata.FriendlyName + "-" + runtimeInfo.instanceID + "-dbus.service",
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

func startProxy(conn *dbus.Conn, ctx context.Context, config Config) {
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
			xdgDir.runtimeDir + "/app/" + config.Metadata.AppID,
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
			dbus.PropSlice("portable-" + config.Metadata.FriendlyName + ".slice"),
			dbus.PropWants(unitWants...),
			dbus.PropDescription("D-Bus proxy for portable sandbox " + config.Metadata.AppID),
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
		config.Metadata.FriendlyName + "-" + runtimeInfo.instanceID + "-dbus.service",
		"replace",
		dbusProps,
		nil,
	)
	if err != nil {
		pecho("crit", "Could not start D-Bus proxy: " + err.Error())
	}
}


type DBusPingRequest struct {}
type DBusControlRequest struct {
	Conn		*godbus.Conn
}

func (m *DBusControlRequest) Stop() (*godbus.Error) {
	if len(runtimeInfo.instanceID) > 0 {
		pecho("debug", "Stopping on Bus request")
		stopApp()
		return nil
	} else {
		pecho("warn", "Could not obtain runtime ID")
		err := errors.New("Could not obtain runtime ID")
		busErr := godbus.MakeFailedError(err)
		return busErr
	}
}

func (m *DBusPingRequest) Ping() (string, *godbus.Error) {
	return "Pong", nil
}

type DBusInfoRequest struct {
	Conn		*godbus.Conn
	SdConn		*dbus.Conn
	Config		Config
}

func (m *DBusInfoRequest) GetInfo() ([]string, *godbus.Error) {
	var reply = []string{
		"Daemon version: " + strconv.FormatFloat(float64(version), 'f', 0, 64),
		"Instance ID: " + runtimeInfo.instanceID,
		"Unit name: " + "app-portable-" + m.Config.Metadata.AppID + "-" + runtimeInfo.instanceID,
	}
	if runtimeInfo.instanceID == "" {
		return []string{}, godbus.MakeFailedError(errors.New("Instance ID unknown"))
	}
	if m.Conn == nil || m.SdConn == nil {
		return []string{}, godbus.MakeFailedError(errors.New("Connection not available"))
	}
	ctx := context.TODO()
	ctxDdl, cancelFunc := context.WithDeadline(ctx, time.Now().Add(1 * time.Second))
	propMap := make(map[string]any)
	propMap, err := m.SdConn.GetAllPropertiesContext(
		ctxDdl,
		"app-portable-" + m.Config.Metadata.AppID + "-" + runtimeInfo.instanceID + ".service",
	)
	cancelFunc()
	if err != nil {
		return reply, godbus.MakeFailedError(err)
	}
	reply = append(reply, appendProps(propMap)...)
	//fmt.Println(propMap)
	return reply, nil
}

func parseNum(v any) (float64) {
	switch n := v.(type) {
		case uint64:
			return float64(n)
		default:
			typeInfo := reflect.TypeOf(v)
			pecho("warn", "Could not parse type " + typeInfo.String())
			return 0
	}
}

func parseStr(v any) (string) {
	switch n := v.(type) {
		case string:
			return string(n)
		default:
			typeInfo := reflect.TypeOf(v)
			pecho("warn", "Could not parse type " + typeInfo.String())
			return ""
	}
}

func appendProps(m map[string]interface{}) []string {
	var ret = []string{}

	cpuTime, ok := m["CPUUsageNSec"]
	if ok {
		timeDur := time.Duration(parseNum(cpuTime)) * time.Nanosecond
		ret = append(ret,
			"CPU time: " + timeDur.String(),
		)
	} else {
		pecho("warn", "Missing property: CPUUsageNSec")
	}

	memUse, ok := m["MemoryCurrent"]
	if ok {
		memRaw := bytesToMb(int(parseNum(memUse)))
		ret = append(ret,
			"Memory usage: " + strconv.FormatFloat(memRaw, 'f', 2, 64) + "M",
		)
	} else {
		pecho("warn", "Missing property: MemoryCurrent")
	}

	cgroup, ok := m["ControlGroup"]
	if ok {
		cgName := parseStr(cgroup)
		ret = append(ret,
			"Control Group: " + cgName,
		)
	} else {
		pecho("warn", "Missing property: ControlGroup")
	}

	return ret
}

func listenBusStub(conn *godbus.Conn, sdConn *dbus.Conn, config Config) {
	ready := make(chan int8, 1)
	go busListener(conn, ready, sdConn, config)
	<- ready

}

func busListener(conn *godbus.Conn, ready chan int8, sdConn *dbus.Conn, config Config) {
	req := new(DBusPingRequest)
	controller := new(DBusControlRequest)
	info := new(DBusInfoRequest)
	info.SdConn = sdConn
	info.Config = config
	info.Conn = conn
	controller.Conn = conn
	objPath := godbus.ObjectPath("/top/kimiblock/portable/daemon")
	node := &introspect.Node{
		//Name:		"top.kimiblock.portable." + confOpts.appID,
		Interfaces:	[]introspect.Interface{
			{
				Name:		"top.kimiblock.portable.Info",
				Methods:	[]introspect.Method{
					{
						Name:	"GetInfo",
						Args:	[]introspect.Arg{
							{
								Name:	"Info",
								Type:	"as",
								Direction:	"out",
							},
						},
					},
				},
			},
			{
				Name:		"top.kimiblock.portable." + config.Metadata.AppID,
				Methods:	[]introspect.Method{
					{
						Name:	"Ping",
						Args:	[]introspect.Arg{
							{
								Name:	"Return",
								Type:	"s",
								Direction:	"out",
							},
						},
					},
				},
			},
			{
				Name:		"top.kimiblock.Portable.Controller",
				Methods:	[]introspect.Method{
					{
						Name:	"Stop",
					},
				},
			},
		},
	}
	err := conn.Export(introspect.NewIntrospectable(node), objPath, "org.freedesktop.DBus.Introspectable")
	if err != nil {
		pecho("crit", "Could not export bus method: " + err.Error())
		return
	}
	err = conn.Export(info, objPath, "top.kimiblock.portable.Info")
	if err != nil {
		pecho("crit", "Could not export bus method: " + err.Error())
		return
	}
	err = conn.Export(req, objPath, "top.kimiblock.portable." + config.Metadata.AppID)
	if err != nil {
		pecho("crit", "Could not export bus method: " + err.Error())
		return
	}
	err = conn.Export(controller, objPath, "top.kimiblock.Portable.Controller")
	if err != nil {
		pecho("crit", "Could not export bus method: " + err.Error())
		return
	}

	ready <- 1
	select {}
}

func startApp(config Config) {
	go forceBackgroundPerm(config)
	sdArgs := append(
		runtimeInfo.bwCmd,
		runtimeOpt.applicationArgs...,
	)
	pecho("debug", "Calculated arguments for systemd-run: " + strings.Join(sdArgs, ", "))
	sdExec := exec.Command("systemd-run", sdArgs...)
	sdExec.Stderr = os.Stderr
	sdExec.Stdout = os.Stdout
	sdExec.Stdin = os.Stdin
	<- envsFlushReady
	// Profiler
	//pprof.Lookup("block").WriteTo(os.Stdout, 1)
	if false {
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

func forceBackgroundPerm(config Config) {
	conn, err := godbus.SessionBus()
	var perms bool
	if err != nil {
		pecho("warn", "Could not set background limits:", err)
		return
	}
	if ! config.Processes.Background {
		pecho("debug", "Restricting background for", config.Metadata.AppID)
		perms = false
	} else {
		pecho("debug", "Unrestricting background limits")
		perms = true
	}

	obj := conn.Object(
		"org.freedesktop.impl.portal.PermissionStore",
		"/org/freedesktop/impl/portal/PermissionStore",
	)
	call := obj.Call(
		"org.freedesktop.impl.portal.PermissionStore.SetPermission",
		0,
		"background",
		perms,
		"background",
		config.Metadata.AppID,
		[]string{"yes"},
	)
	if call.Err != nil {
		pecho("warn", "Could not set background limits:", call.Err)
		return
	}
}

func instDesktopFile(config Config) {
	var wg sync.WaitGroup
	wg.Go(func() {
		err := os.MkdirAll(
			filepath.Join(
				xdgDir.dataDir,
				"applications",
				),
			0700,
		)
		if err != nil {
			pecho("crit", "Could not create directory:", err)
		}
	})
	xdgDataDirs := strings.SplitSeq(strings.TrimSpace(os.Getenv("XDG_DATA_DIRS")), ":")
	for val := range xdgDataDirs {
		if strings.Contains(val, "/var/lib/flatpak") {
			continue
		}
		statPath := filepath.Join(
			val,
			"applications",
			config.Metadata.AppID + ".desktop",
		)
		_, err := os.Stat(statPath)
		if err == nil {
			pecho("debug", "Found desktop file: " + statPath)
			return
		}
	}

	var hardcodedDesktopPath = []string{filepath.Join(xdgDir.dataDir, "applications")}

	for _, path := range hardcodedDesktopPath {
		statPath := filepath.Join(
			path,
			config.Metadata.AppID + ".desktop",
		)
		_, err := os.Stat(statPath)
		if err == nil {
			pecho("debug", "Found desktop file: " + statPath)
			return
		}
	}

	const templateDesktopFile string = "[Desktop Entry]\nName=placeholderName\nExec=env _portableConfig=placeholderConfig portable\nTerminal=false\nType=Application\nIcon=image-missing\nComment=Application info missing\n"
	replacer := strings.NewReplacer(
		"placeholderName",		"Portable sandbox: " + config.Metadata.AppID,
		"placeholderConfig",		config.Path,
	)
	wg.Wait()
	file, err := os.OpenFile(
		filepath.Join(
			xdgDir.dataDir,
			"applications",
			config.Metadata.AppID + ".desktop",
		),
		os.O_WRONLY|os.O_TRUNC|os.O_CREATE,
		0700,
	)
	if err != nil {
		pecho("warn", "Could not open .desktop file for writing: " + err.Error())
		pecho("warn", "Non-existent .desktop file may result in Portals crashing")
		return
	}
	defer file.Close()
	_, err = replacer.WriteString(file, templateDesktopFile)
	if err != nil {
		pecho("warn", "Could not write .desktop file: " + err.Error())
		pecho("warn", "Non-existent .desktop file may result in Portals crashing")
		return
	}
	runtimeOpt.writtenDesktop = true
	pecho("debug", "Done installing stub file")
	pecho("warn", "You should supply your own .desktop file")
}

func setXDGEnvs(config Config) {
	addEnv("XDG_CONFIG_HOME=" + translatePath(xdgDir.confDir, config))
	addEnv("XDG_DOCUMENTS_DIR=" + filepath.Join(xdgDir.dataDir, config.Metadata.StateDirectory, "Documents"))
	addEnv("XDG_DATA_HOME=" + filepath.Join(xdgDir.dataDir, config.Metadata.StateDirectory, ".local/share"))
	addEnv("XDG_STATE_HOME=" + filepath.Join(xdgDir.dataDir, config.Metadata.StateDirectory, ".local/state"))
	addEnv("XDG_CACHE_HOME=" + filepath.Join(xdgDir.dataDir, config.Metadata.StateDirectory, "cache"))
	addEnv("XDG_DESKTOP_DIR=" + filepath.Join(xdgDir.dataDir, config.Metadata.StateDirectory, "Desktop"))
	addEnv("XDG_DOWNLOAD_DIR=" + filepath.Join(xdgDir.dataDir, config.Metadata.StateDirectory, "Downloads"))
	addEnv("XDG_TEMPLATES_DIR=" + filepath.Join(xdgDir.dataDir, config.Metadata.StateDirectory, "Templates"))
	addEnv("XDG_PUBLICSHARE_DIR=" + filepath.Join(xdgDir.dataDir, config.Metadata.StateDirectory, "Public"))
	addEnv("XDG_MUSIC_DIR=" + filepath.Join(xdgDir.dataDir, config.Metadata.StateDirectory, "Music"))
	addEnv("XDG_PICTURES_DIR=" + filepath.Join(xdgDir.dataDir, config.Metadata.StateDirectory, "Pictures"))
	addEnv("XDG_VIDEOS_DIR=" + filepath.Join(xdgDir.dataDir, config.Metadata.StateDirectory, "Videos"))
}

func imEnvs (config Config) {
	addEnv("IBUS_USE_PORTAL=1")
	var imKind string
	if ! config.Privacy.X11 {
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
							scanner := bufio.NewScanner(commFd)
							for scanner.Scan() {
								line := scanner.Text()
								if strings.Contains(line, "fcitx") {
									pecho("debug", "Guessing IM: Fcitx")
									addEnv("GTK_IM_MODULE=fcitx")
									addEnv("QT_IM_MODULE=fcitx")
									break
								} else if strings.Contains(line, "ibus") {
									pecho("debug", "Guessing IM: iBus")
									addEnv("QT_IM_MODULE=ibus")
									addEnv("GTK_IM_MODULE=ibus")
									break
								}
							}
							commFd.Close()
						}
					}
				}
		}
	}
}

func setupSharedDir (config Config) {
	err := os.MkdirAll(
		filepath.Join(xdgDir.dataDir, config.Metadata.StateDirectory, "Shared"),
		0700,
	)
	if err != nil {
		pecho("warn", "Could not setup shared directory: " + err.Error())
	}
	err = os.Symlink(
		filepath.Join(xdgDir.dataDir, config.Metadata.StateDirectory, "Shared"),
		filepath.Join(xdgDir.dataDir, config.Metadata.StateDirectory, "共享文件"),
	)
	if err != nil {
		if os.IsExist(err) {} else {
			pecho("warn", "Could not setup shared directory: " + err.Error())
		}
	}
}

func miscEnvs (config Config) {
	if config.Advanced.Qt5Compat {
		addEnv("QT_QPA_PLATFORMTHEME=xdgdesktopportal")
	}
	err := os.MkdirAll(xdgDir.runtimeDir + "/portable/" + config.Metadata.AppID, 0700)
	if err != nil {
		pecho("crit", "Could not create runtime directory: " + err.Error())
	}
	var file string = "source " + filepath.Join(xdgDir.runtimeDir, "portable", config.Metadata.AppID, "generated.env") + "\n"
	wrErr := os.WriteFile(
		filepath.Join(xdgDir.runtimeDir, "portable", config.Metadata.AppID, "bashrc"),
		[]byte(file),
		0700)
	if wrErr != nil {
		pecho("warn", "Unable to write bashrc: " + wrErr.Error())
	}
	addEnv("GDK_DEBUG=portals")
	addEnv("GTK_USE_PORTAL=1")
	addEnv("QT_AUTO_SCREEN_SCALE_FACTOR=1")
	addEnv("QT_ENABLE_HIGHDPI_SCALING=1")
	addEnv("PS1=" + strconv.Quote("╰─>Portable·" + config.Metadata.AppID + "·🤓 ⤔ "))
	addEnv("QT_SCALE_FACTOR=" + os.Getenv("QT_SCALE_FACTOR"))
	addEnv("HOME=" + filepath.Join(xdgDir.dataDir, config.Metadata.StateDirectory))
	addEnv("XDG_SESSION_TYPE=" + os.Getenv("XDG_SESSION_TYPE"))
	addEnv("WAYLAND_DISPLAY=" + filepath.Join(xdgDir.runtimeDir, "wayland-0"))
	addEnv("DBUS_SESSION_BUS_ADDRESS=unix:path=/run/sessionBus")
}

func prepareEnvs(config Config) {
	var wg sync.WaitGroup
	wg.Go(func() {
		if config.Advanced.Landlock {
			addEnv("_portableEnableLandlock=1")
		}
		if ! config.Advanced.FlatpakInfo {
			addEnv("_portableNoFlatpakInfo=1")
		}
		if config.System.InhibitOnBehalf {
			addEnv("_portableInhibit=1")
		}
		addEnv("appID=" + config.Metadata.AppID)
		addEnv("friendlyName=" + config.Metadata.FriendlyName)
		addEnv("_portableLaunchTarget=" + config.Exec.Target)
		if config.BusActivation.Enable {
			actSlice := append([]string{config.BusActivation.Target}, config.BusActivation.Arguments...)
			jsonObj, err := json.Marshal(actSlice)
			if err != nil {
				pecho("warn", "Could not marshal application arguments: " + err.Error())
			} else {
				addEnv("_portableBusActivateArgs=" + string(jsonObj))
			}
		}

	})
	wg.Go(func() {
		imEnvs(config)
	})
	wg.Go(func() {
		setXDGEnvs(config)
	})
	wg.Go(func() {
		miscEnvs(config)
	})
	wg.Go(func() {
		statePath := filepath.Join(xdgDir.dataDir, config.Metadata.StateDirectory, "portable.env")
		userEnvs, err := os.OpenFile(
			statePath,
			os.O_RDONLY,
			0700,
		)
		if err != nil {
			if os.IsNotExist(err) {
				const template = "# This file accepts simple KEY=VAL envs"
				os.WriteFile(
					statePath,
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

	wg.Wait()
}

func genBwArg(
	xChan chan []string,
	camChan chan []string,
	inputChan chan []string,
	wayDisplayChan chan []string,
	miscChan	chan []string,
	config Config,
	) {

	if internalLoggingLevel > 1 {
		runtimeInfo.bwCmd = append(runtimeInfo.bwCmd, "--quiet")
	}

	// Define global systemd args first
	runtimeInfo.bwCmd = append(
		runtimeInfo.bwCmd,
		"--user",
		"--service-type=notify",
		"--wait",
		"--pty",
		"--unit=" + "app-portable-" + config.Metadata.AppID + "-" + runtimeInfo.instanceID,
		"--slice=app.slice",
		"-p", "Delegate=yes",
		"-p", "DelegateSubgroup=portable-cgroup",
		"-p", "BindsTo=" + config.Metadata.FriendlyName + "-" + runtimeInfo.instanceID + "-dbus.service",
		"-p", "Description=Portable Sandbox for " + config.Metadata.FriendlyName + " (" + config.Metadata.AppID + ")",
		"-p", "Documentation=https://github.com/Kraftland/portable",
		"-p", "ExitType=cgroup",
		"-p", "SuccessExitStatus=SIGKILL",
		"-p", "NotifyAccess=all",
		"-p", "TimeoutStartSec=infinity",
		"-p", "SecureBits=noroot-locked",
		"-p", "NoNewPrivileges=yes",
		"-p", "KillMode=control-group",
		"-p", "MemoryHigh=90%",
		"-p", "IPAccounting=yes",
		"-p", "MemoryPressureWatch=yes",
		"-p", "SyslogIdentifier=portable-" + config.Metadata.AppID,
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
		"EnvironmentFile=" + filepath.Join(xdgDir.runtimeDir, "portable", config.Metadata.AppID, "generated.env"),
		"-p", "Environment=instanceId=" + runtimeInfo.instanceID,
		"-p", "Environment=busDir=" + filepath.Join(xdgDir.runtimeDir, "app", config.Metadata.AppID),
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
		"-p", "WorkingDirectory=" + filepath.Join(xdgDir.dataDir, config.Metadata.StateDirectory),
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

	if ! config.Network.Enable {
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

		"--perms",		"0755",
		"--dir",		"/host",
		"--ro-bind-try",	"/opt", "/opt",
		"--ro-bind",		"/usr", "/usr",
		"--overlay-src",	"/usr/bin",
		"--overlay-src",	"/usr/lib/portable/overlay-usr",
		"--ro-overlay",		"/usr/bin",
		"--symlink",		"/usr/lib", "/lib",
		"--symlink",		"/usr/lib", "/lib64",
		"--symlink",		"/usr/bin", "/bin",
		"--symlink",		"/usr/bin", "/sbin",
		// Tmp binds
		"--tmpfs",		"/tmp",

		// Dev binds
		"--dev",		"/dev",
		"--tmpfs",		"/dev/shm",
		"--dev-bind-try",	"/dev/mali", "/dev/mali",
		"--dev-bind-try",	"/dev/mali0", "/dev/mali0",
		"--dev-bind-try",	"/dev/umplock", "/dev/umplock",
		"--mqueue",		"/dev/mqueue",
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
		"--bind-try",		"/sys/devices/system", "/sys/devices/system",
		"--ro-bind",		"/sys/kernel", "/sys/kernel",
		"--ro-bind",		"/sys/devices/virtual", "/sys/devices/virtual",

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

		"--ro-bind-try",	"/var/cache/fontconfig", "/var/cache/fontconfig",

		// Run binds
		"--bind",
			filepath.Join(xdgDir.runtimeDir, "portable", config.Metadata.AppID),
			"/run",
		"--bind",
			filepath.Join(xdgDir.runtimeDir, "portable", config.Metadata.AppID),
			filepath.Join(xdgDir.runtimeDir, "portable", config.Metadata.AppID),

		"--ro-bind",
			filepath.Join(xdgDir.runtimeDir, "app", config.Metadata.AppID, "bus"),
			"/run/sessionBus",
		"--ro-bind-try",
			filepath.Join(xdgDir.runtimeDir, "portable", config.Metadata.AppID, "a11y"),
			filepath.Join(xdgDir.runtimeDir, "at-spi"),
		"--dir",		"/run/host",
		"--bind",
			filepath.Join(xdgDir.runtimeDir, "doc/by-app", config.Metadata.AppID),
			filepath.Join(xdgDir.runtimeDir, "doc"),
		"--ro-bind-try",
			"/run/systemd/resolve/stub-resolv.conf",
			"/run/systemd/resolve/stub-resolv.conf",
		"--bind",
			filepath.Join(xdgDir.runtimeDir, "systemd/notify"),
			filepath.Join(xdgDir.runtimeDir, "systemd/notify"),
		"--ro-bind-try",
			filepath.Join(xdgDir.runtimeDir, "pulse"),
			filepath.Join(xdgDir.runtimeDir, "pulse"),

		// HOME binds
		"--bind",
			filepath.Join(xdgDir.dataDir, config.Metadata.StateDirectory),
			xdgDir.home,
		"--bind",
			filepath.Join(xdgDir.dataDir, config.Metadata.StateDirectory),
			filepath.Join(xdgDir.dataDir, config.Metadata.StateDirectory),

		"--ro-bind",		"/etc", "/etc",
		"--ro-bind-try",	filepath.Join(xdgDir.runtimeDir, "portable", config.Metadata.AppID, "passwd"), "/etc/passwd",
		"--ro-bind-try",	filepath.Join(xdgDir.runtimeDir, "portable", config.Metadata.AppID, "nsswitch"), "/etc/nsswitch.conf",

		// Privacy mounts
		"--tmpfs",		"/proc/1",
		"--tmpfs",		"/usr/share/applications",
		"--tmpfs",		filepath.Join(xdgDir.home, "options"),
		"--tmpfs",		filepath.Join(xdgDir.dataDir, config.Metadata.StateDirectory, "options"),

	)


	xArgs := <- xChan
	runtimeInfo.bwCmd = append(
		runtimeInfo.bwCmd,
		xArgs...
	)

	select {
		case wayArgs := <- wayDisplayChan:
			runtimeInfo.bwCmd = append(
				runtimeInfo.bwCmd,
				wayArgs...
			)
		default:
			pecho("warn", "Could not find a working Wayland socket: either the compositor is not started, or WAYLAND_DISPLAY is pointed to a wrong address")
	}




	for arg := range miscChan {
		runtimeInfo.bwCmd = append(
			runtimeInfo.bwCmd,
			arg...
		)
	}

	if config.Privacy.Input {
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

func flushEnvs(config Config) {
	var builder strings.Builder

	for env := range envsChan {
		builder.WriteString(env)
		builder.WriteString("\n")
	}

	fd, err := os.OpenFile(
		xdgDir.runtimeDir + "/portable/" + config.Metadata.AppID + "/generated.env",
			os.O_CREATE|os.O_TRUNC|os.O_WRONLY,
			0700,
	)
	if err != nil {
		pecho(
			"crit",
			"Could not open environment variables: " + err.Error(),
		)
		return
	}
	defer fd.Close()
	writer := bufio.NewWriter(fd)
	writer.WriteString(builder.String())
	err = writer.Flush()
	if err != nil {
		pecho("crit", "Could not write environment variables: " + err.Error())
	}
	close(envsFlushReady)
}

func translatePath(input string, config Config) (output string) {
	output = strings.ReplaceAll(input, xdgDir.home, filepath.Join(xdgDir.dataDir, config.Metadata.StateDirectory))
	return
}

// Does not take empty input
func questionExpose(paths []string, config Config) bool {
	zenityArgs := []string{
		"--title",
			config.Metadata.FriendlyName,
		"--icon=folder-open-symbolic",
		"--question",
		"--default-cancel",
	}
	var zenityText strings.Builder
	switch runtimeOpt.userLang {
		case "zh_CN.UTF-8":
			zenityText.WriteString("--text=授权以下路径: \n ")
		default:
			zenityText.WriteString("--text=Exposing the following path: \n ")
	}
	for _, val := range paths {
		zenityText.WriteString(val)
		zenityText.WriteString("\n")
	}
	zenityArgs = append(zenityArgs, zenityText.String())

	attrs := syscall.SysProcAttr{
		Pdeathsig:		syscall.SIGKILL,
	}
	zenityCmd := exec.Command("zenity", zenityArgs...)
	zenityCmd.SysProcAttr = &attrs
	zenityCmd.Stderr = os.Stderr
	if internalLoggingLevel <= 1 {
		zenityCmd.Stdout = os.Stdout
	}
	err := zenityCmd.Run()
	if err != nil {
		pecho("warn", "Could not ask for permission: " + err.Error())
		return false
	}
	return true
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

func miscBinds(miscChan chan []string, pwChan chan []string, config Config, exposeChan chan map[string]string, docMap chan PassFiles) {

	defer close(docMap)
	var wg sync.WaitGroup

	wg.Go(func() {
		miscChan <- engageExpose(exposeChan, config, docMap)
	})

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
			translatePath(filepath.Join(xdgDir.confDir, "fontconfig"), config),
		}
	})

	wg.Go(func() {
		miscChan <- []string {
			"--ro-bind-try",
			xdgDir.confDir + "/gtk-3.0/gtk.css",
			translatePath(xdgDir.confDir + "/gtk-3.0/gtk.css", config),
		}
	})

	wg.Go(func() {
		miscChan <- []string {
			"--ro-bind-try",
			xdgDir.confDir + "/gtk-3.0/colors.css",
			translatePath(xdgDir.confDir + "/gtk-3.0/colors.css", config),
		}
	})

	wg.Go(func() {
		miscChan <- []string {
			"--ro-bind-try",
			xdgDir.confDir + "/gtk-4.0/gtk.css",
			translatePath(xdgDir.confDir + "/gtk-4.0/gtk.css", config),
		}
	})

	wg.Go(func() {
		miscChan <- []string{
			"--ro-bind-try",
			xdgDir.confDir + "/qt6ct",
			translatePath(xdgDir.confDir + "/qt6ct", config),
		}
	})

	wg.Go(func() {
		miscChan <- []string{
			"--ro-bind-try",
			xdgDir.dataDir + "/fonts",
			translatePath(xdgDir.dataDir + "/fonts", config),
		}
	})

	wg.Go(func() {
		miscChan <- []string{
			"--ro-bind-try",
			xdgDir.dataDir + "/icons",
			translatePath(xdgDir.dataDir + "/icons", config),
		}
	})

	if config.Advanced.FlatpakInfo {
		miscChan <- []string{
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
				xdgDir.runtimeDir + "/portable/" + config.Metadata.AppID + "/flatpak-info",
				"/.flatpak-info",
			"--ro-bind",
				xdgDir.runtimeDir + "/portable/" + config.Metadata.AppID + "/flatpak-info",
				xdgDir.runtimeDir + "/.flatpak-info",
			"--ro-bind",
				xdgDir.runtimeDir + "/portable/" + config.Metadata.AppID + "/flatpak-info",
				xdgDir.dataDir + "/" + config.Metadata.StateDirectory + "/.flatpak-info",
			"--tmpfs",		xdgDir.home + "/.var",
			"--tmpfs",		xdgDir.dataDir + "/" + config.Metadata.StateDirectory + "/.var",
			"--bind",
				xdgDir.dataDir + "/" + config.Metadata.StateDirectory,
				xdgDir.dataDir + "/" + config.Metadata.StateDirectory + "/.var/app/" + config.Metadata.AppID,
			"--tmpfs",
				xdgDir.dataDir + "/" + config.Metadata.StateDirectory + "/.var/app/" + config.Metadata.AppID + "/options",
		}
	}

	wg.Wait()
	for pw := range pwChan {
		miscChan <- pw
	}
	close(miscChan)
}

func bindXAuth(xauthChan chan []string, config Config) {
	var xArg = []string{}
	if config.Privacy.X11 {
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

func tryBindCam(camChan chan []string, config Config) {
	camArg := []string{}
	if config.Privacy.Cameras {
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
	inputBindChan <- []string{
		"--dev-bind-try",	"/sys/class/leds", "/sys/class/leds",
		"--dev-bind-try",	"/sys/class/input", "/sys/class/input",
		"--dev-bind-try",	"/sys/class/hidraw", "/sys/class/hidraw",
		"--dev-bind-try",	"/dev/input", "/dev/input",
		"--dev-bind-try",	"/dev/uinput", "/dev/uinput",
	}

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

func atSpiProxy(conn *godbus.Conn, config Config) {
	a11yBusObj := conn.Object(
		"org.a11y.Bus",
		"/org/a11y/bus",
	)
	ctx := context.TODO()
	timeDeadline := time.Now().Add(5 * time.Millisecond)
	ctxChild, cancelFunc := context.WithDeadline(ctx, timeDeadline)
	call := a11yBusObj.CallWithContext(ctxChild, "org.a11y.Bus.GetAddress", godbus.FlagNoAutoStart)
	cancelFunc()
	if call.Err != nil {
		pecho("warn", "Could not get accessibility bus address: " + call.Err.Error())
		return
	}

	var a11yAddr string
	err := call.Store(&a11yAddr)
	if err != nil {
		pecho("warn", "Could not decode accessibility bus address: " + err.Error())
		return
	}
	err = os.MkdirAll(
		filepath.Join(xdgDir.runtimeDir, "portable", config.Metadata.AppID, "a11y"),
		0700,
	)
	if err != nil {
		pecho("warn", "Could not create directory for accessibility bus: " + err.Error())
		return
	}
	atspiArgs := []string{
		"--die-with-parent",
		"--symlink", "/usr/lib64", "/lib64",
		"--ro-bind", "/usr/lib", "/usr/lib",
		"--ro-bind", "/usr/lib64", "/usr/lib64",
		"--ro-bind", "/usr/bin", "/usr/bin",
		"--ro-bind", "/usr/share", "/usr/share",
		"--bind", xdgDir.runtimeDir + "/at-spi", xdgDir.runtimeDir + "/at-spi",
		"--bind",
			xdgDir.runtimeDir + "/portable/" + config.Metadata.AppID + "/a11y",
			xdgDir.runtimeDir + "/portable/" + config.Metadata.AppID + "/a11y",
		"--ro-bind",
			xdgDir.runtimeDir + "/portable/" + config.Metadata.AppID + "/flatpak-info",
			xdgDir.runtimeDir + "/.flatpak-info",
		"--ro-bind",
			xdgDir.runtimeDir + "/portable/" + config.Metadata.AppID + "/flatpak-info",
			"/.flatpak-info",
		"--",
		"/usr/bin/xdg-dbus-proxy",
		a11yAddr,
		xdgDir.runtimeDir + "/portable/" + config.Metadata.AppID + "/a11y/bus",
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
		Pdeathsig:		syscall.SIGTERM,
	}
	atSpiProxyCmd.SysProcAttr = &sysattr

	atSpiProxyCmd.Start()
}
func main() {
	exposeChan := make(chan map[string]string, 16)
	miChan := make(chan bool, 1)
	var wg sync.WaitGroup
	var config Config
	wg.Go(func() {
		config = getConf()
	})
	sigChan := make(chan os.Signal, 1)
	var busConn *godbus.Conn
	go signalRecvWorker(sigChan)
	go pechoWorker()
	wayDisplayChan := make(chan[]string, 1)

	var sdContext context.Context
	var sdCancelFunc context.CancelFunc
	var conn *dbus.Conn
	wg.Go(func() {
		var err error
		sdContext, sdCancelFunc = context.WithCancel(context.Background())
		conn, err = dbus.NewUserConnectionContext(sdContext)
		if err != nil {
			pecho("crit", "Could not connect to user service manager: " + err.Error())
			return
		}
	})




	inputChan := make(chan []string, 4)
	go inputBind(inputChan) // This is fine, since genBwArg takes care of conf switching
	pecho("info", "Portable daemon", version)
	cmdChan := make(chan int8, 1)
	wg.Wait()
	wg.Go(func() {
		var err error
		busConn, err = godbus.SessionBus()
		if err != nil {
			panic("Could not connect to session bus: " + err.Error())
		}
		if busConn.SupportsUnixFDs() == false {
			panic("D-Bus has no support for passing File Descriptors")
		}
		reply, err := busConn.RequestName("top.kimiblock.portable." + config.Metadata.AppID, godbus.NameFlagDoNotQueue)
		if err != nil {
			pecho("crit", "Could not request bus name: " + err.Error())
			return
		}
		switch reply {
			case godbus.RequestNameReplyPrimaryOwner:
				pecho("debug", "Successfully requested ownership of bus name")
				go stopAppWorker(conn, sdCancelFunc, sdContext, busConn, config)
			case godbus.RequestNameReplyExists:
				pecho("info", "Another instance is currently running")
				miChan <- true
				return
			default:
				pecho("crit", "Could not obtain D-Bus name: " + reply.String())
		}
		miChan <- false

	})
	go waylandDisplay(wayDisplayChan)
	// This is fine to do concurrently, since miscBind runs later and we have wg.Wait in middle
	wg.Go(func() {
		getVariables(exposeChan)
	})


	go cmdlineDispatcher(cmdChan, config, exposeChan)
	go gpuBind(gpuChan, config)
	miscChan := make(chan []string, 10240)
	pwSecContextChan := make(chan []string, 1)

	wg.Go(func() {
		instDesktopFile(config)
	})
	wg.Go(func() {
		setupSharedDir(config)
	})
	genChan := make(chan int8, 2) /* Signals when an ID has been chosen,
		and we signal back when multi-instance is cleared
		in another channel */
	genChanProceed := make(chan int8, 1)
	wg.Add(1)
	go func () {
		defer wg.Done()
		genInstanceID(genChan, genChanProceed, config)
	} ()
	xChan := make(chan []string, 1)
	go bindXAuth(xChan, config)
	camChan := make(chan []string, 1)
	go tryBindCam(camChan, config)

	<- cmdChan
	docsMap := make(chan PassFiles, 1)
	go miscBinds(miscChan, pwSecContextChan, config, exposeChan, docsMap)

	abortChan <- false
	if abort := <- abortChan; abort {
		pecho("warn", "Aborting start sequence")
		return
	}

	if multiInstanceDetected := <- miChan; multiInstanceDetected {
		wakeInstance(config, docsMap)
	} else {
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	wg.Go(func() {
		prepareEnvs(config)
	})
	wg.Go(func() {
		setFirewall(config)
	})
	wg.Go(func() {
		sanityChecks(config)
	})
	wg.Go(func() {
		listenBusStub(busConn, conn, config)
	})
	go flushEnvs(config)

	<- genChan // Stage one, ensures that IDs are actually present
	wg.Go(func() {
		genBwArg(xChan, camChan, inputChan, wayDisplayChan, miscChan, config)
	})
	genChanProceed <- 1
	go calcDbusArg(busArgChan, config)
	<- genChan // Stage 2, ensures that info file is ready
	wg.Go(func() {
		startProxy(conn, sdContext, config)
	})
	wg.Go(func() {
		atSpiProxy(busConn, config)
	})

	go pwSecContext(pwSecContextChan, config)
	wg.Wait()
	close(envsChan)
	startApp(config)
	if busConn != nil {
		busConn.Close()
	}
	}
}