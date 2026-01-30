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

function readyNotify() {
	# Notifies readiness, only usable after warnMulRunning()
	# $1 can be: wait, set, set-fail, init
	# $2 is the item name
	if [[ $1 = "set" ]]; then
		mkdir -p "${XDG_RUNTIME_DIR}/portable/${appID}/ready-${readyDir}/$2/ready" &
		pecho debug "Readiness set for $2" &
	elif [[ $1 = "set-fail" ]]; then
		mkdir -p "${XDG_RUNTIME_DIR}/portable/${appID}/ready-${readyDir}/fail" &
		#rm -rf "${XDG_RUNTIME_DIR}/portable/${appID}/ready-${readyDir}"
	elif [[ $1 = "init" ]]; then
		readyDir="${RANDOM}"
		while [[ -d "${XDG_RUNTIME_DIR}/portable/${appID}/ready-${readyDir}" ]]; do
			readyDir="${RANDOM}"
		done
		pecho debug "Chosen readiness code ${readyDir}"
		mkdir \
			--parents \
			--mode=0700 \
			"${XDG_RUNTIME_DIR}/portable/${appID}/ready-${readyDir}"
	elif [[ $1 = "wait" ]]; then
		if [[ -e "${XDG_RUNTIME_DIR}/portable/${appID}/ready-${readyDir}/$2/ready" ]]; then
			pecho debug "Component $2 ready verified" &
			return 0
		fi
		pecho debug "Waiting for component: $2..." &
		while true; do
			if [[ -e "${XDG_RUNTIME_DIR}/portable/${appID}/ready-${readyDir}/$2/ready" ]]; then
				break
			elif [[ -e "${XDG_RUNTIME_DIR}/portable/${appID}/ready-${readyDir}/fail" ]]; then
				exit 114
			else
				if [[ ! -e "${XDG_RUNTIME_DIR}/portable/${appID}/ready-${readyDir}" ]]; then
					exit 114
				fi
				continue
			fi
		done
		pecho debug "Done waiting for $2..." &
	fi
}

function sanityCheck() {
	mountCheck
	configCheck
	busCheck "${appID}"
	bindCheck
	readyNotify set sanityCheck
}

function busCheck() {
	local busOwn="${appID}"
	if [[ "${busOwn}" = org.mpris.MediaPlayer2$ ]]; then
		pecho crit "appID invalid: prohibited to own entire org.mpris.MediaPlayer2"
		readyNotify set-fail sanityCheck
	elif [[ "${busOwn}" =~ org.freedesktop.impl.* ]]; then
		pecho crit "appID invalid: sandbox escape not allowed"
		readyNotify set-fail sanityCheck
	elif [[ "${busOwn}" =~ org.gtk.vfs.* ]]; then
		pecho crit "appID invalid: full filesystem access not allowed"
		readyNotify set-fail sanityCheck
	fi
}

function bindCheck() {
	if [[ -z "${bwBindPar}" ]]; then
		readyNotify set bindCheck
		return 0
	fi
	if [[ -e "${bwBindPar}" ]]; then
		if [[ -d "${bwBindPar}" ]]; then
			declare -i fileCnt
			declare -i dirCnt
			fileCnt=$(find "${bwBindPar}" -maxdepth 1 -mindepth 1 -type f | wc -l)
			dirCnt=$(find "${bwBindPar}" -maxdepth 1 -mindepth 1 -type d | wc -l)
			if [[ "${fileCnt}" -gt 1 ]]; then
				local trailingF="files"
			else
				local trailingF="file"
			fi
			if [[ "${dirCnt}" -gt 1 ]]; then
				local trailingD="directories"
			else
				local trailingD="directory"
			fi
		else
			local fileCnt=1
			local trailingF="file"
			local dirCnt=0
			local trailingD="directory"
		fi
		if [[ "${LANG}" =~ "zh_CN" ]]; then
			/usr/bin/zenity \
				--title "${friendlyName}" \
				--icon=folder-open-symbolic \
				--question \
				--text="是否暴露路径 ${bwBindPar}: ${fileCnt} 个文件, ${dirCnt} 个子目录"
			local status=$?
			if [[ "${status}" -eq 0 ]]; then
				readyNotify set bindCheck
			else
				readyNotify set-fail bindCheck
			fi
		else
			/usr/bin/zenity \
				--title "${friendlyName}" \
				--icon=folder-open-symbolic \
				--question \
				--text="Expose ${bwBindPar}, containing ${fileCnt} ${trailingF}, ${dirCnt} ${trailingD}?"
			local status=$?
			if [[ "${status}" -eq 0 ]]; then
				readyNotify set bindCheck
			else
				readyNotify set-fail bindCheck
			fi
		fi
	else
		if [[ "${LANG}" =~ "zh_CN" ]]; then
			/usr/bin/zenity \
				--title "${friendlyName}" \
				--warning \
				--text="设定的共享路径不存在"
		else
			/usr/bin/zenity \
				--title "${friendlyName}" \
				--warning \
				--text="Specified shared path does not exist."
		fi
		readyNotify set-fail bindCheck
	fi
}

function mountCheck() {
	declare mounts
	mounts="$(systemd-run --quiet --user -P -- findmnt -R)"
	if [[ "${mounts}" =~ "/usr/bin/" ]]; then
		pecho crit "Mountpoints inside /usr/bin! Please unmount them for at least the user service manager"
		readyNotify set-fail sanityCheck
	fi
}

function confEmpty() {
	local varName="$1"
	local varVal="${!varName}"
	if [[ -z "${varVal}" ]]; then
		pecho crit "Config option $1 is empty!"
		readyNotify set-fail sanityCheck
	fi
}

function confBool() {
	local varName="$1"
	local varVal="${!varName}"
	if [[ "${varVal}" = "true" ]] || [[ "${varVal}" = "false" ]]; then
		return 0
	elif [[ -z "${varVal}" ]]; then
		pecho info "Config option ${1} unspecified"
	else
		pecho warn "Config option ${1} should be boolean"
		return 1
	fi
}

function configCheck() {
	for value in appID friendlyName stateDirectory launchTarget; do
		confEmpty ${value}
	done
	unset value
}

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

function manageDirs() {
	createWrapIfNotExist "${XDG_DATA_HOME}/${stateDirectory}"
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

function createWrapIfNotExist() {
	if [[ -d "$*" ]]; then
		return 0
	else
		mkdir \
			--parents \
			--mode=0700 \
			"$@"
	fi
}
# Function used to escape paths for sed processing.
function pathEscape() {
	local str="$*"
	local delimiter="|"
	# Escape the delimiter and &
	str="${str//${delimiter}/\\${delimiter}}"
	str="${str//&/\\&}"
	echo "$str"
}

function passBwrapArgs() {
	local bwArgWrite="$*"
	#echo -e ${bwArgWrite} >>"${XDG_RUNTIME_DIR}/portable/${appID}/bwrapArgs"
	flock --exclusive "${XDG_RUNTIME_DIR}/portable/${appID}/bwrapArgs.lock" 'echo' '-ne' "${bwArgWrite}" >>"${XDG_RUNTIME_DIR}/portable/${appID}/bwrapArgs"
}

function calcBwrapArg() {
	procDriverBind &
	readyNotify wait bindCheck
	if [[ -z "${bwBindPar}" || ! -e "${bwBindPar}" ]]; then
		unset bwBindPar
	else
		passBwrapArgs "--dev-bind\0${bwBindPar}\0${bwBindPar}\0"
	fi
	readyNotify set calcBwrapArg
}

# Translates path based on ~ to state directory
function pathTranslation() {
	sed "s|$(pathEscape "${HOME}")|$(pathEscape "${XDG_DATA_HOME}/${stateDirectory}")|g"
}

function defineRunPath() {
	mkdir \
		--parents \
		--mode=0700 \
		"${XDG_RUNTIME_DIR}/portable/${appID}"
}

function execApp() {
	calcBwrapArg &
	addEnv targetArgs="${targetArgs}"
	readyNotify wait calcBwrapArg
	/usr/lib/portable/daemon/portable-daemon $@
}

function execAppExistDirect() {
	echo "${launchTarget}" "${targetArgs}" > "${XDG_RUNTIME_DIR}/portable/${appID}/startSignal"
}
function execAppExist() {
	unitName="${unitName}-subprocess-$(uuidgen)"
	instanceId=$(grep instance-id "${XDG_RUNTIME_DIR}/portable/${appID}/flatpak-info" | cut -c '13-')
	execApp
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

function warnMulRunning() {
	if [[ "${dbusWake}" = "true" ]]; then
		id=$(dbus-send \
			--bus=unix:path="${busDir}/bus" \
			--dest=org.kde.StatusNotifierWatcher \
			--type=method_call \
			--print-reply=literal /StatusNotifierWatcher \
			org.freedesktop.DBus.Properties.Get \
			string:org.kde.StatusNotifierWatcher \
			string:RegisteredStatusNotifierItems | grep -oP 'org.kde.StatusNotifierItem-\d+-\d+')
		pecho debug "Unique ID: ${id}"
		dbus-send \
			--print-reply \
			--session \
			--dest="${id}" \
			--type=method_call \
			/StatusNotifierItem \
			org.kde.StatusNotifierItem.Activate \
			int32:114514 \
			int32:1919810
		status="$?"
		case $status in
			0)
				exit 0
				;;
			1)
				appANR
				;;
			*)
				appANR
				exit "$status"
				;;
		esac
	else
		pecho info "Skipping D-Bus wake"
	fi
	source "${_portableConfig}"
	execAppExistDirect
	exit "$?"
	# Appears to be unreachable
	# appANR
	# if [[ $? -eq 0 ]]; then
	# 	stopApp force
	# else
	# 	pecho crit "User denied session termination"
	# 	exit "$?"
	# fi
}

function dbusProxy() {
	if [[ ${securityContext} -eq 1 ]]; then
		rm -rf "${XDG_RUNTIME_DIR}/portable/${appID}/wayland.sock"
		systemd-run \
			--user \
			--quiet \
			-p Slice="portable-${friendlyName}.slice" \
			-u "${friendlyName}"-wayland-proxy \
			-p BindsTo="${proxyName}.service" \
			-p Environment=WAYLAND_DISPLAY="${WAYLAND_DISPLAY}" \
   			-p Environment=XDG_SESSION_TYPE="${XDG_SESSION_TYPE}" \
			-- \
			way-secure \
				-e top.kimiblock.portable \
				-a "${appID}" \
				-i "${instanceId}" \
				--socket-path "${XDG_RUNTIME_DIR}/portable/${appID}/wayland.sock"
	fi
	if [[ ! -S "${XDG_RUNTIME_DIR}/at-spi/bus" ]]; then
		pecho warn "No at-spi bus detected!"
		touch "${busDirAy}/bus"
		return 0
	fi
	systemd-run \
		--user \
		--quiet \
		-p Slice="portable-${friendlyName}.slice" \
		-u "${proxyName}-a11y" \
		-p RestartMode=direct \
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
	if systemctl --user --quiet is-failed "${unitName}.service"; then
		pecho warn "${appID} failed last time"
		systemctl --user reset-failed "${unitName}.service" &
	fi
	if systemctl --user --quiet is-active "${unitName}.service"; then
		warnMulRunning
	elif systemctl --user --quiet is-active "${friendlyName}.service"; then
		warnMulRunning
	fi
	sanityCheck &
	if [[ ${trashAppUnsafe} -eq 1 ]]; then
		pecho warn "Launching ${appID} (unsafe)..."
		execAppUnsafe
	else
		dbusProxy
		pecho info "Launching ${appID}..."
		execApp $@
	fi
}

function resetDocuments() {
	flatpak permission-reset "${appID}"
	exit $?
}

function showStats() {
	systemctl --user \
		status \
		"${unitName}"

	exit 0
}

function cmdlineDispatcherv2() {
	declare -i cmdArgCount
	declare trailingS
	cmdArgCount=0
	while true; do
		if [[ -z $* ]]; then
			break
		elif [[ $1 = "--" ]]; then
			shift
			declare -g -x targetArgs="$*"
			declare -r targetArgs
			break
		elif [[ "$1" = "--actions" ]]; then
			shift
			cmdArgCount+=1
			if [[ "$1" =~ ^opendir|openhome$ ]]; then
				/usr/bin/xdg-open "${XDG_DATA_HOME}/${stateDirectory}"
				exit $?
			elif [[ "$1" =~ ^reset-documents|reset-document$ ]]; then
				resetDocuments
			elif [[ "$1" =~ ^stats|stat$ ]]; then
				showStats
			elif [[ "$1" = "f5aaebc6-0014-4d30-beba-72bce57e0650" ]]; then
				rm -f "${XDG_DATA_HOME}/${stateDirectory}/options/sandbox"
				questionFirstLaunch
			else
				pecho warn "Unrecognised action: $1"
			fi
		elif [[ "$1" =~ ^--help|-h$ ]]; then
			printHelp
		else
			pecho warn "Unrecognised argument: $1!"
		fi
		cmdArgCount+=1
		shift
	done
	if [[ "${cmdArgCount}" -eq 1 ]]; then
		trailingS=""
	else
		trailingS="s"
	fi
	pecho info "Resolution of portable command line arguments finished with ${cmdArgCount} argument${trailingS}"
	pecho info "Application argument interpreted as: \"${targetArgs}\""
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
defineRunPath
readyNotify init
manageDirs
cmdlineDispatcherv2 $@
launch $@