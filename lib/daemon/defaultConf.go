package main

import (
	"os"
	"strings"
)

// Returns a new Config object with defaults set
func setDefaultConfOpts() Config {
	var config Config
	sessionType := os.Getenv("XDG_SESSION_TYPE")
	execEnv := os.Getenv("launchTarget")
	if len(execEnv) > 0 {
		execSlice := strings.Split(execEnv, " ")
		config.Exec.Target = execSlice[0]
		config.Exec.Arguments = execSlice[1:]
	}
	switch sessionType {
		case "wayland":
			config.Privacy.X11 = false
		default:
			config.Privacy.X11 = true
	}
	config.Processes.Background = true
	config.Network.Enable = true
	config.Processes.Track = true
	config.Privacy.ClassicNotifications = true
	config.Advanced.Qt5Compat = true
	config.Advanced.FlatpakInfo = true
	config.Advanced.KDEStatus = true

	return config
}