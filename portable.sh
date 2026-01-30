#!/bin/bash

# Shellcheck configurations below
# shellcheck disable=SC1090,SC2174,SC2154,SC2129,SC1091,SC2086
# Shellcheck configurations end

function pecho() {
	if [[ "$1" = "debug" ]] && [[ "${PORTABLE_LOGGING}" = "debug" ]]; then
		echo "[Debug] $2" &
	elif [[ "$1" = "info" ]] && [[ "${PORTABLE_LOGGING}" = "info" || "${PORTABLE_LOGGING}" = "debug" ]]; then
		echo "[Info] $2" &
	elif [[ "$1" = "warn" ]]; then
		echo "[Warn] $2" &
	elif [[ "$1" = "crit" ]]; then
		echo "[Critical] $2"
	fi
}

function printHelp() {
	echo "This is Portable, a fast, private and efficient Linux desktop sandbox."
	echo "Visit https://github.com/Kraftland/portable for documentation."
	echo -e "\n"
	echo "Environment variables:"
	echo "	PORTABLE_LOGGING	-> Optional"
	echo "		Possible values: debug, info"
	echo "	_portalConfig		-> Required"
	echo "		Possible values: "
	echo "			Application ID of installed sandbox under /usr/lib/portable/info"
	echo "			Relative or absolute path to a configuration file"
	echo -e "\n"
	echo "Command line arguments (optional):"
	echo "	-v	-	-	-> Verbose output"
	echo "	--actions <action>"
	echo "		debug-shell	-> Enter the sandbox via a bash shell"
	echo "		opendir	-	-> Open the sandbox's home directory"
	echo "		share-files	-> Place files in sandbox's \"Shared\" directory"
	echo "		reset-documents	-> Revoke granted file access permissions"
	echo "		stats	-	-> Show basic status of the sandbox (if running)"
	echo "	--	-	-	-> Any argument after this double dash will be passed to the application"
	echo "	--help	-	-	-> Print this help"
	exit 0
}

if [[ "${_portalConfig}" ]]; then
	export _portableConfig="${_portalConfig}"
	pecho warn "Using legacy configuration variable!"
fi

if [[ -z "${_portableConfig}" ]]; then
	printHelp
elif [[ -r "${_portableConfig}" ]]; then
	pecho info "Configuration specified as absolute path: ${_portableConfig}"
	source "${_portableConfig}"
else
	if [[ -r "/usr/lib/portable/info/${_portableConfig}/config" ]]; then
		pecho info \
			"Configuration specified as global name /usr/lib/portable/info/${_portableConfig}/config"
		source "/usr/lib/portable/info/${_portableConfig}/config"
		export _portableConfig="/usr/lib/portable/info/${_portableConfig}/config"
	elif [[ -r "$(pwd)/${_portableConfig}" ]]; then
		pecho info \
			"Configuration specified as relative path ${_portableConfig}"
		source "$(pwd)/${_portableConfig}"
		declare -g -x
		_portableConfig="$(pwd)/${_portableConfig}"
	else
		pecho crit "Specified configuration not reachable"
		exit 1
	fi
fi

busName="${appID}"
busDir="${XDG_RUNTIME_DIR}/app/${busName}"
busDirAy="${XDG_RUNTIME_DIR}/app/${busName}-a11y"
unitName="app-portable-${appID}"
proxyName="${friendlyName}-dbus"

function sourceXDG() {
	if [[ ! "${XDG_CONFIG_HOME}" ]]; then
		export XDG_CONFIG_HOME="${HOME}/.config"
		pecho info "Guessing XDG Config Home @ ${XDG_CONFIG_HOME}"
	else
		source "${XDG_CONFIG_HOME}/user-dirs.dirs"
		pecho info "XDG Config Home defined @ ${XDG_CONFIG_HOME}"
	fi
	if [[ ! "${XDG_DATA_HOME}" ]]; then
		export XDG_DATA_HOME="${HOME}/.local/share"
	fi
}

function waylandContext() {
	if [[ -x /usr/bin/wayland-info && -x /usr/bin/way-secure ]]; then
		if [[ "${XDG_SESSION_TYPE}" = "wayland" && "$(/usr/bin/wayland-info)" =~ "wp_security_context_manager_v1" && ${allowSecurityContext} -eq 1 ]]; then
			pecho debug "Wayland security context available"
			securityContext=1
			wayDisplayBind="${XDG_RUNTIME_DIR}/portable/${appID}/wayland.sock"
		else
			pecho warn "Wayland security context not available"
		fi
	else
		pecho debug "Security Context is not available due to missing dependencies"
	fi
}

function execApp() {
	/usr/lib/portable/daemon/portable-daemon $@
}

function appANR() {
	if [[ "${LANG}" =~ "zh_CN" ]]; then
		zenity --title "程序未响应" --icon=utilities-system-monitor-symbolic --default-cancel --question --text="是否结束正在运行的进程?"
		local status=$?
	else
		zenity --title "Application is not responding" --icon=utilities-system-monitor-symbolic --default-cancel --question --text="Do you wish to terminate the running session?"
		local status=$?
	fi
	if [[ "${status}" -eq 0 ]]; then
		stopApp force
	fi
}

function dbusProxy() {
	if [[ ! -S "${XDG_RUNTIME_DIR}/at-spi/bus" ]]; then
		pecho warn "No at-spi bus detected!"
		touch "${busDirAy}/bus"
		return 0
	fi
	systemd-run \
		-- bwrap \
			--symlink /usr/lib64 /lib64 \
			--ro-bind /usr/lib /usr/lib \
			--ro-bind /usr/lib64 /usr/lib64 \
			--ro-bind /usr/bin /usr/bin \
			--ro-bind-try /usr/share /usr/share \
			--bind "${XDG_RUNTIME_DIR}" "${XDG_RUNTIME_DIR}" \
			--ro-bind "${XDG_RUNTIME_DIR}/portable/${appID}/flatpak-info" \
				"${XDG_RUNTIME_DIR}/.flatpak-info" \
			--ro-bind "${XDG_RUNTIME_DIR}/portable/${appID}/flatpak-info" \
				/.flatpak-info \
			-- /usr/bin/xdg-dbus-proxy \
			unix:path="${XDG_RUNTIME_DIR}/at-spi/bus" \
			"${busDirAy}/bus" \
			--filter \
			--sloppy-names \
			--call=org.a11y.atspi.Registry=org.a11y.atspi.Socket.Embed@/org/a11y/atspi/accessible/root \
			--call=org.a11y.atspi.Registry=org.a11y.atspi.Socket.Unembed@/org/a11y/atspi/accessible/root \
			--call=org.a11y.atspi.Registry=org.a11y.atspi.Registry.GetRegisteredEvents@/org/a11y/atspi/registry \
			--call=org.a11y.atspi.Registry=org.a11y.atspi.DeviceEventController.GetKeystrokeListeners@/org/a11y/atspi/registry/deviceeventcontroller \
			--call=org.a11y.atspi.Registry=org.a11y.atspi.DeviceEventController.GetDeviceEventListeners@/org/a11y/atspi/registry/deviceeventcontroller \
			--call=org.a11y.atspi.Registry=org.a11y.atspi.DeviceEventController.NotifyListenersSync@/org/a11y/atspi/registry/deviceeventcontroller \
			--call=org.a11y.atspi.Registry=org.a11y.atspi.DeviceEventController.NotifyListenersAsync@/org/a11y/atspi/registry/deviceeventcontroller


}

function launch() {
	dbusProxy
	pecho info "Launching ${appID}..."
	execApp $@
}

set -m
export \
	pwCam \
	qt5Compat \
	useZink \
	XDG_DATA_HOME \
	stateDirectory \
	XDG_CONFIG_HOME \
	_portableConfig \
	XDG_RUNTIME_DIR \
	appID \
	DISPLAY \
	QT_SCALE_FACTOR \
	waylandOnly \
	instanceId \
	readyDir \
	gameMode \
	GSK_RENDERER=gl
sourceXDG
launch $@