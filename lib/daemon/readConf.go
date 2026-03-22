package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
)

func determineConfType () (legacy bool) {
	modernConfEnv := os.Getenv("PORTABLE_CONF")
	if len(modernConfEnv) > 0 {
		return false
	}
	if len(os.Getenv("_portableConfig")) > 0 || len(os.Getenv("_portalConfig")) > 0 {
		return true
	}
	pecho("crit", "Please specify the PORTABLE_CONF variable for configuration")
	select {}
	//return false
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


func determineModernConfPath(raw string) string {
	type pathInfo struct {
		Path		string
		Priority	int
	}

	pathChan := make(chan pathInfo, 5)
	var wg sync.WaitGroup
	wg.Go(func() {
		path := filepath.Join("/usr/lib/portable/info", raw, "config.toml")
		stat, err := os.Stat(path)
		if err != nil {
			return
		}
		if stat.IsDir() {
			pecho("warn", "Unable to use system-wide configuration: is a directory")
			return
		}
		pathChan <- pathInfo{
			Path:		path,
			Priority:	1,
		}
		pecho("debug", "System-wide configuration available")
	})
	wg.Go(func() {
		path := filepath.Join(xdgDir.confDir, "/portable/info", raw, "config.toml")
		stat, err := os.Stat(path)
		if err != nil {
			return
		}
		if stat.IsDir() {
			pecho("warn", "Unable to use user configuration: is a directory")
			return
		}
		pathChan <- pathInfo{
			Path:		path,
			Priority:	2,
		}
		pecho("debug", "User configuration available")
	})
	wg.Go(func() {
		stat, err := os.Stat(raw)
		if err != nil {
			return
		}
		if stat.IsDir() {
			pecho("warn", "Unable to use absolute path as configuration: is a directory")
			return
		}
		pathChan <- pathInfo{
			Path:		raw,
			Priority:	3,
		}
		pecho("debug", "User absolute path configuration available")
	})

	go func () {
		wg.Wait()
		close(pathChan)
	} ()
	var finalPathInfo pathInfo
	for sig := range pathChan {
		if finalPathInfo.Priority < sig.Priority {
			finalPathInfo = sig
		}
	}
	if len(finalPathInfo.Path) == 0 {
		pecho("crit", "Could not obtain configuration path")
		select {}
	} else {
		pecho("debug", "Using configuration path " + finalPathInfo.Path)
	}
	return finalPathInfo.Path
}

func getConf() Config {
	lookUpXDG()
	var config Config
	pecho("debug", "Attempting to get configuration...")
	if determineConfType() {
		pecho("warn", "Using legacy KEY=VAL configuration, please switch to the new TOML format")
		config = readLegacyConf()
	} else {
		pecho("debug", "Using modern TOML configuration")
		configPath := determineModernConfPath(os.Getenv("PORTABLE_CONF"))
		file, err := os.OpenFile(configPath, os.O_RDONLY, 0700)
		if err != nil {
			pecho("crit", "Could not open configuration: " + err.Error())
			select {}
		}
		reader := bufio.NewReader(file)
		decoder := toml.NewDecoder(reader)
		md ,err := decoder.Decode(&config)
		if err != nil {
			pecho("crit", "Could not decode configuration: " + err.Error())
			select {}
		}
		config.Path = configPath
		switch len(md.Undecoded()) {
			case 0:
			case 1:
				pecho("warn", "Could not decode 1 key in configuration")
			default:
				pecho("warn", "Could not decode " + strconv.Itoa(len(md.Undecoded())) + " keys in configuration")
		}
	}
	sessionType := os.Getenv("XDG_SESSION_TYPE")
	switch sessionType {
		case "wayland":
		case "x11":
			config.Privacy.X11 = true
		default:
			pecho("warn", "Could not obtain session type")
			config.Privacy.X11 = true
	}
	return config
}