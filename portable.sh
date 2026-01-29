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

function genXAuth() {
	if [[ "${waylandOnly}" = "true" ]] || [[ "${waylandOnly}" = "adaptive" && "${XDG_SESSION_TYPE}" = "wayland" ]]; then
		pecho debug "Wayland only mode enforced"
		#addEnv "DISPLAY"
		xAuthBind="/dev/null"
		return 1
	elif [[ -r "${XAUTHORITY}" ]]; then
		pecho debug "Using authority file from ${XAUTHORITY}"
		xAuthBind="${XAUTHORITY}"
	elif [[ -r "${HOME}/.Xauthority" ]]; then
		pecho debug "Guessing authority as ${HOME}/.Xauthority"
		xAuthBind="${HOME}/.Xauthority"
	else
		pecho warn "Could not determine Xauthority file path"
		xAuthBind="/dev/null"
		unset XAUTHORITY
		xhost +localhost
	fi
	export XAUTHORITY="/run/.Xauthority"
	addEnv "DISPLAY=${DISPLAY}"
	addEnv "XAUTHORITY=${XAUTHORITY}"
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

function inputMethod() {
	if [[ "${waylandOnly}" = "true" ]]; then
		pecho debug "Using Wayland Input Method"
		export QT_IM_MODULE=wayland
		export GTK_IM_MODULE=wayland
		export IBUS_USE_PORTAL=1
		return 0
	elif [[ "${waylandOnly}" =~ "adaptive" && "${XDG_SESSION_TYPE}" =~ "wayland" ]]; then
		pecho debug "Using Wayland Input Method"
		export QT_IM_MODULE=wayland
		export GTK_IM_MODULE=wayland
		export IBUS_USE_PORTAL=1
		return 0
	fi
	if [[ "${XMODIFIERS}" =~ "fcitx" || "${QT_IM_MODULE}" =~ "fcitx" || "${GTK_IM_MODULE}" =~ "fcitx" ]]; then
		export QT_IM_MODULE=fcitx
		export GTK_IM_MODULE=fcitx
	elif [[ "${XMODIFIERS}" =~ "ibus" || "${QT_IM_MODULE}" =~ "ibus" || "${GTK_IM_MODULE}" =~ "ibus" ]]; then
		export QT_IM_MODULE=ibus
		export GTK_IM_MODULE=ibus
		export IBUS_USE_PORTAL=1
	elif [[ "${XMODIFIERS}" =~ "gcin" ]]; then
		export QT_IM_MODULE=ibus
		export GTK_IM_MODULE=gcin
		export LC_CTYPE=zh_TW.UTF-8
	else
		pecho warn "Input Method potentially broken! Please set \$XMODIFIERS properly"
		# Guess the true IM based on running processes
		runningProcess=$(ps -U "$(whoami)")
		if [[ "${runningProcess}" =~ "ibus-daemon" ]]; then
			pecho warn "Guessing Input Method as iBus"
			export QT_IM_MODULE=ibus
			export GTK_IM_MODULE=ibus
			export XMODIFIERS=@im=ibus
		elif [[ "${runningProcess}" =~ "fcitx" ]]; then
			pecho warn "Guessing Input Method as Fcitx"
			export QT_IM_MODULE=fcitx
			export GTK_IM_MODULE=fcitx
			export XMODIFIERS=@im=fcitx
		fi
	fi

}

function setIM() {
	inputMethod
	addEnv "GTK_IM_MODULE=${GTK_IM_MODULE}"
	addEnv "QT_IM_MODULE=${QT_IM_MODULE}"
	readyNotify set im
}

function setConfEnv() {
	if [[ "${qt5Compat}" = "false" ]]; then
		pecho debug "Skipping Qt 5 compatibility workarounds"
	else
		pecho debug "Enabling Qt 5 compatibility workarounds"
		addEnv "QT_QPA_PLATFORMTHEME=xdgdesktopportal"
	fi
	if [[ "${useZink}" = "true" ]]; then
		pecho debug "Enabling Zink..."
		addEnv "__GLX_VENDOR_LIBRARY_NAME=mesa"
		addEnv "MESA_LOADER_DRIVER_OVERRIDE=zink"
		addEnv "GALLIUM_DRIVER=zink"
		addEnv "LIBGL_KOPPER_DRI2=1"
		addEnv "__EGL_VENDOR_LIBRARY_FILENAMES=/usr/share/glvnd/egl_vendor.d/50_mesa.json"
	fi
	readyNotify set setConfEnv
}

function setXdgEnv() {
	addEnv "XDG_CONFIG_HOME=$(echo "${XDG_CONFIG_HOME}" | pathTranslation)"
	addEnv "XDG_DOCUMENTS_DIR=${XDG_DATA_HOME}/${stateDirectory}/Documents"
	addEnv "XDG_DATA_HOME=${XDG_DATA_HOME}/${stateDirectory}/.local/share"
	addEnv "XDG_STATE_HOME=${XDG_DATA_HOME}/${stateDirectory}/.local/state"
	addEnv "XDG_CACHE_HOME=${XDG_DATA_HOME}/${stateDirectory}/cache"
	addEnv "XDG_DESKTOP_DIR=${XDG_DATA_HOME}/${stateDirectory}/Desktop"
	addEnv "XDG_DOWNLOAD_DIR=${XDG_DATA_HOME}/${stateDirectory}/Downloads"
	addEnv "XDG_TEMPLATES_DIR=${XDG_DATA_HOME}/${stateDirectory}/Templates"
	addEnv "XDG_PUBLICSHARE_DIR=${XDG_DATA_HOME}/${stateDirectory}/Public"
	addEnv "XDG_MUSIC_DIR=${XDG_DATA_HOME}/${stateDirectory}/Music"
	addEnv "XDG_PICTURES_DIR=${XDG_DATA_HOME}/${stateDirectory}/Pictures"
	addEnv "XDG_VIDEOS_DIR=${XDG_DATA_HOME}/${stateDirectory}/Videos"
	readyNotify set setXdgEnv
}

function setStaticEnv() {
	addEnv "GDK_DEBUG=portals"
	addEnv "GTK_USE_PORTAL=1"
	addEnv "QT_AUTO_SCREEN_SCALE_FACTOR=1"
	addEnv "QT_ENABLE_HIGHDPI_SCALING=1"
	addEnv "PS1='╰─>Portable Sandbox·${appID}·🧐⤔ '"
	addEnv "QT_SCALE_FACTOR=${QT_SCALE_FACTOR}"
	addEnv "HOME=${XDG_DATA_HOME}/${stateDirectory}"
	addEnv "XDG_SESSION_TYPE=${XDG_SESSION_TYPE}"
	addEnv "WAYLAND_DISPLAY=wayland-0"
	addEnv "DBUS_SESSION_BUS_ADDRESS=unix:path=/run/sessionBus"
	echo "source /run/portable-generated.env" > "${XDG_RUNTIME_DIR}/portable/${appID}/bashrc"
	readyNotify set setStaticEnv
}

function genNewEnv() {
	if [[ ! -e "${XDG_DATA_HOME}/${stateDirectory}/portable.env" ]]; then
		touch "${XDG_DATA_HOME}/${stateDirectory}/portable.env"
	fi
	if [[ -s "${XDG_DATA_HOME}/${stateDirectory}/portable.env" ]]; then
		cat "${XDG_DATA_HOME}/${stateDirectory}/portable.env" >> "${XDG_RUNTIME_DIR}/portable/${appID}/portable-generated.env"
	else
		echo "# Envs" >> "${XDG_DATA_HOME}/${stateDirectory}/portable.env"
		echo "isPortableEnvPresent=1" >> "${XDG_DATA_HOME}/${stateDirectory}/portable.env"
	fi
	mkdir \
		--parents \
		--mode=0700 \
		"${XDG_DATA_HOME}/${stateDirectory}/.config" &
	rm -r "${XDG_DATA_HOME}/${stateDirectory}/Shared"
	mkdir \
		--parents \
		--mode=0700 \
		"${XDG_DATA_HOME}/${stateDirectory}/Shared" &
	ln -sfr \
		"${XDG_DATA_HOME}/${stateDirectory}/Shared" \
		"${XDG_DATA_HOME}/${stateDirectory}/共享文件" &
	readyNotify set genNewEnv
}

function importEnv() {
	cat "${_portableConfig}" > "${XDG_RUNTIME_DIR}/portable/${appID}/portable-generated.env"
	setIM &
	setXdgEnv &
	setConfEnv &
	setStaticEnv &
	ln -srf \
		"${XDG_RUNTIME_DIR}/portable/${appID}/portable-generated.env" \
		"${XDG_DATA_HOME}/${stateDirectory}/portable-generated.env" &
	genNewEnv &
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

function procDriverBind() {
	if [[ -d /proc/driver ]]; then
		passBwrapArgs "--tmpfs\0/proc/driver\0"
	fi
	if [[ -d "/proc/bus" ]]; then
		passBwrapArgs "--tmpfs\0/proc/bus\0"
	fi
	readyNotify set procDriverBind
}

# Pass NUL separated arguments!
function calcMountArgv2() {
	if [[ "${mountInfo}" = "false" ]]; then
		pecho debug "Not mounting flatpak-info..."
	else
		passBwrapArgs "--ro-bind\0${XDG_RUNTIME_DIR}/portable/${appID}/flatpak-info\0/.flatpak-info\0--ro-bind\0${XDG_RUNTIME_DIR}/portable/${appID}/flatpak-info\0${XDG_RUNTIME_DIR}/.flatpak-info\0--ro-bind\0${XDG_RUNTIME_DIR}/portable/${appID}/flatpak-info\0${XDG_DATA_HOME}/${stateDirectory}/.flatpak-info\0"
	fi
	passBwrapArgs "--ro-bind-try\0${XDG_CONFIG_HOME}/fontconfig\0$(echo "${XDG_CONFIG_HOME}" | pathTranslation)/fontconfig\0--ro-bind-try\0${XDG_CONFIG_HOME}/gtk-3.0/gtk.css\0$(echo "${XDG_CONFIG_HOME}" | pathTranslation)/gtk-3.0/gtk.css\0--ro-bind-try\0${XDG_CONFIG_HOME}/gtk-3.0/colors.css\0$(echo "${XDG_CONFIG_HOME}" | pathTranslation)/gtk-3.0/colors.css\0--ro-bind-try\0${XDG_CONFIG_HOME}/gtk-4.0/gtk.css\0$(echo "${XDG_CONFIG_HOME}" | pathTranslation)/gtk-4.0/gtk.css\0--ro-bind-try\0${XDG_CONFIG_HOME}/qt6ct\0$(echo "${XDG_CONFIG_HOME}" | pathTranslation)/qt6ct\0--ro-bind-try\0${XDG_DATA_HOME}/fonts\0${XDG_DATA_HOME}/fonts\0--ro-bind-try\0${XDG_DATA_HOME}/fonts\0$(echo "${XDG_DATA_HOME}" | pathTranslation)/fonts\0--ro-bind-try\0${XDG_DATA_HOME}/icons\0${XDG_DATA_HOME}/icons\0--ro-bind-try\0${XDG_DATA_HOME}/icons\0$(echo "${XDG_DATA_HOME}" | pathTranslation)/icons\0"
	readyNotify set calcMountArgv2
}

function calcBwrapArg() {

	if [ -e /usr/lib/flatpak-xdg-utils/flatpak-spawn ]; then
		passBwrapArgs "--ro-bind\0/usr/lib/portable/overlay-usr/flatpak-spawn\0/usr/lib/flatpak-xdg-utils/flatpak-spawn\0"
	fi
	passBwrapArgs "--ro-bind\0${xAuthBind}\0/run/.Xauthority\0--ro-bind\0${busDir}/bus\0/run/sessionBus\0--ro-bind-try\0${busDirAy}\0${XDG_RUNTIME_DIR}/at-spi\0--dir\0/run/host\0--bind\0${XDG_RUNTIME_DIR}/doc/by-app/${appID}\0${XDG_RUNTIME_DIR}/doc\0--ro-bind\0/dev/null\0${XDG_RUNTIME_DIR}/.flatpak/${instanceId}-private/run-environ\0--ro-bind\0${XDG_RUNTIME_DIR}/.flatpak/${instanceId}\0${XDG_RUNTIME_DIR}/.flatpak/${instanceId}\0--ro-bind\0${XDG_RUNTIME_DIR}/.flatpak/${instanceId}\0${XDG_RUNTIME_DIR}/flatpak-runtime-directory\0--ro-bind-try\0${wayDisplayBind}\0${XDG_RUNTIME_DIR}/wayland-0\0--ro-bind-try\0/run/systemd/resolve/stub-resolv.conf\0/run/systemd/resolve/stub-resolv.conf\0--bind\0${XDG_RUNTIME_DIR}/systemd/notify\0${XDG_RUNTIME_DIR}/systemd/notify\0" # Run binds
	passBwrapArgs "--bind\0${XDG_DATA_HOME}/${stateDirectory}\0${HOME}\0--bind\0${XDG_DATA_HOME}/${stateDirectory}\0${XDG_DATA_HOME}/${stateDirectory}\0" # HOME binds
	procDriverBind &
	calcMountArgv2 &
	passBwrapArgs "--ro-bind-try\0${XDG_RUNTIME_DIR}/pulse\0${XDG_RUNTIME_DIR}/pulse\0" # PulseAudio Bind!
	passBwrapArgs "--ro-bind\0/etc\0/etc\0--tmpfs\0/etc/kernel\0"
	passBwrapArgs "--tmpfs\0/proc/1\0--tmpfs\0/usr/share/applications\0--tmpfs\0${HOME}/options\0--tmpfs\0${XDG_DATA_HOME}/${stateDirectory}/options\0--tmpfs\0${HOME}/.var\0--tmpfs\0${XDG_DATA_HOME}/${stateDirectory}/.var\0--bind\0${XDG_DATA_HOME}/${stateDirectory}\0${XDG_DATA_HOME}/${stateDirectory}/.var/app/${appID}\0--bind\0${XDG_DATA_HOME}/${stateDirectory}\0${HOME}/.var/app/${appID}\0--tmpfs\0${HOME}/.var/app/${appID}/options\0--tmpfs\0${XDG_DATA_HOME}/${stateDirectory}/.var/app/${appID}/options\0" # Hide some entries
	readyNotify wait bindCheck
	if [[ -z "${bwBindPar}" || ! -e "${bwBindPar}" ]]; then
		unset bwBindPar
	else
		passBwrapArgs "--dev-bind\0${bwBindPar}\0${bwBindPar}\0"
	fi
	readyNotify wait procDriverBind
	readyNotify wait calcMountArgv2
	passBwrapArgs "--\0/usr/lib/portable/helper"
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
	addEnv _portableDebug="${_portableDebug}"
	addEnv _portableBusActivate="${_portableBusActivate}"
	termExec
	terminateOnRequest &
	readyNotify wait calcBwrapArg
	/usr/lib/portable/daemon/portable-daemon $@
}

function terminateOnRequest() {
	if [[ -e "${XDG_RUNTIME_DIR}/portable/${appID}/ready-${readyDir}/fail" ]]; then
		pecho warn "One or more components failed during startup, terminating now..."
		stopApp force
	fi
	pecho debug "Established termination watches"
	while true; do
		if [[ ! -e "${XDG_RUNTIME_DIR}/portable/${appID}/startSignal" ]]; then
			pecho warn "startSignal is missing! Stopping application"
			stopApp force
		fi
		inotifywait \
			--quiet \
			"${XDG_RUNTIME_DIR}/portable/${appID}/startSignal"
		if grep -q "terminate-now" "${XDG_RUNTIME_DIR}/portable/${appID}/startSignal"; then
			stopApp force
		fi
	done
}

function execAppExistDirect() {
	echo "${launchTarget}" "${targetArgs}" > "${XDG_RUNTIME_DIR}/portable/${appID}/startSignal"
}

function termExec() {
	trap "stopApp force" SIGTERM SIGINT SIGHUP SIGQUIT SIGILL SIGABRT SIGUSR1 SIGSEGV
}

function execAppExist() {
	genXAuth
	importEnv
	unitName="${unitName}-subprocess-$(uuidgen)"
	instanceId=$(grep instance-id "${XDG_RUNTIME_DIR}/portable/${appID}/flatpak-info" | cut -c '13-')
	execApp
}

function shareFile() {
	if [[ ${trashAppUnsafe} -eq 1 ]]; then
		zenity \
			--error \
			--title "Sandbox disabled" \
			--text "Feature is intended for sandbox users"
		pecho crit "Sandbox is disabled"
		exit 1
	fi
	fileList=$(zenity --file-selection --multiple | tail -n 1)
	IFS='|' read -r -a filePaths <<< "${fileList}"
	for filePath in "${filePaths[@]}"; do
		pecho info "User selected path: ${filePath}"
		cp -a \
			"${filePath}" \
			"${XDG_DATA_HOME}/${stateDirectory}/Shared"
	done
	exit 0
}
function addEnv() {
	flock -x "${XDG_RUNTIME_DIR}/portable/${appID}/portable-generated.env.lock" \
		/usr/lib/portable/addEnv "$@"
}
function detectNv(){
	if ls /dev/nvidia* &> /dev/null; then
		pecho debug "NVIDIA GPU present"
		export nvExist=1
	fi
}

# Meant to run after bindNvDevIfExist() or detectNv()
function setNvOffloadEnv() {
	addEnv "VK_LOADER_DRIVERS_DISABLE="
	detectNv
	if [[ "${nvExist}" = 1 ]]; then
		pecho debug "Specifying environment variables for dGPU utilization: NVIDIA"
		addEnv "__NV_PRIME_RENDER_OFFLOAD=1"
		addEnv "__VK_LAYER_NV_optimus=NVIDIA_only"
		addEnv "__GLX_VENDOR_LIBRARY_NAME=nvidia"
		addEnv "VK_LOADER_DRIVERS_SELECT=nvidia_icd.json"
	else
		pecho debug "Specifying environment variables for dGPU utilization: Mesa"
		addEnv "DRI_PRIME=1"
	fi
}

# $1=card[0-9], sets renderIndex in form of renderD128, etc
function cardToRender() {
	unset renderIndex
	declare sysfsPath devPath
	sysfsPath="/sys$(udevadm info /sys/class/drm/$1 --query=path)"
	devPath="${sysfsPath}"
	declare -g renderIndex
	renderIndex="$(basename "$(find "${sysfsPath}/../" -maxdepth 1 -mindepth 1 -name 'render*' -print -quit)")" # head is not needed since find exits on first match
	pecho debug "Translated $1 to ${renderIndex}"
}

# $1 as arg name, $2 as value
function passDevArgs() {
	echo "$2" >"${XDG_RUNTIME_DIR}/portable/${appID}/devstore/$1"
}

# $1 as arg name.
function getDevArgs() {
	export "$1=$(cat "${XDG_RUNTIME_DIR}/portable/${appID}/devstore/$1")" 2>/dev/null
}

# Take video card number as input $1, e.g. card0, and prints out card's PCI path
function resolvePCICard() {
	declare sysfsPath
	sysfsPath="$(udevadm info /sys/class/drm/$1 --query=path)"
	echo "/sys${sysfsPath}" | sed "s|drm/$1||g"
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
	importEnv &
	genXAuth

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
	readyNotify wait im
	readyNotify wait setXdgEnv
	readyNotify wait setConfEnv
	readyNotify wait setStaticEnv
	readyNotify wait genNewEnv
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

function execAppUnsafe() {
	#importEnv
	inputMethod
	source "${XDG_RUNTIME_DIR}/portable/${appID}/portable-generated.env"
	pecho info "GTK_IM_MODULE is ${GTK_IM_MODULE}"
	pecho info "QT_IM_MODULE is ${QT_IM_MODULE}"
	systemd-run --user \
		-p Slice="portable-${friendlyName}.slice" \
		-p Environment=QT_AUTO_SCREEN_SCALE_FACTOR="${QT_AUTO_SCREEN_SCALE_FACTOR}" \
		-p Environment=QT_ENABLE_HIGHDPI_SCALING="${QT_ENABLE_HIGHDPI_SCALING}" \
		-p Environment=GTK_IM_MODULE="${GTK_IM_MODULE}" \
		-p Environment=QT_IM_MODULE="${QT_IM_MODULE}" \
		-p Environment=XMODIFIERS="${XMODIFIERS}" \
		-p EnvironmentFile=-"${XDG_DATA_HOME}/${stateDirectory}/portable.env" \
		-u "${unitName}" \
		--tty \
		${launchTarget}
}

function enableSandboxFunc() {
	pecho info "Sandboxing confirmed"
	mkdir \
		--parents \
		--mode=0700 \
		"${XDG_DATA_HOME}/${stateDirectory}/options"
	touch "${XDG_DATA_HOME}/${stateDirectory}/options/sandbox"
	return 0
}

function questionFirstLaunch() {
	if [[ ! -f "${XDG_DATA_HOME}/${stateDirectory}/options/sandbox" ]]; then
		if [[ "${LANG}" =~ "zh_CN" ]]; then
			/usr/bin/zenity \
				--title "${friendlyName}" \
				--icon=security-medium-symbolic \
				--question \
				--text="为 ${appID} 启用沙盒?"
		else
			/usr/bin/zenity \
				--title "Portable" \
				--icon=security-medium-symbolic \
				--question \
				--text="Enable sandbox for ${friendlyName}(${appID})?"
		fi
		if [[ $? -eq 1 ]]; then
			if [[ "${LANG}" =~ "zh_CN" ]]; then
				zenity \
					--question \
					--default-cancel \
					--title "确认操作" \
					--icon=security-low-symbolic \
					--text "若要更改设定, 运行 _portableConfig=\"${_portableConfig}\" portable --actions f5aaebc6-0014-4d30-beba-72bce57e0650"
			else
				zenity \
					--question \
					--default-cancel \
					--title "Confirm action" \
					--icon=security-low-symbolic \
					--text "Change this anytime via command: _portableConfig=\"${_portableConfig}\" portable --actions f5aaebc6-0014-4d30-beba-72bce57e0650"
			fi
			if [[ $? -eq 1 ]]; then
				pecho info "User enabled sandbox late"
				enableSandboxFunc &
				return 0
			else
				pecho warn "User disabled sandbox!"
				mkdir \
					--parents \
					--mode=0700 \
					"${XDG_DATA_HOME}/${stateDirectory}/options"
				echo "disableSandbox" >> "${XDG_DATA_HOME}/${stateDirectory}/options/sandbox"
				export trashAppUnsafe=1
			fi
		else
			enableSandboxFunc &
			return 0
		fi
	elif [[ $(cat "${XDG_DATA_HOME}/${stateDirectory}/options/sandbox") =~ "disableSandbox" ]]; then
		export trashAppUnsafe=1
	fi
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

function stopSlice() {
	systemctl \
		--user stop \
		"app-portable-${friendlyName}.slice" 2>/dev/null
	systemctl \
		--user stop \
		"portable-${friendlyName}.slice" 2>/dev/null
}

function cleanDirs() {
	source "${XDG_RUNTIME_DIR}/portable/${appID}/control"
	pecho debug "Cleaning leftovers..."
	if [[ -n "${instanceId}" ]]; then
		rm -rf "${XDG_RUNTIME_DIR}/.flatpak/${instanceId}"
	else
		pecho warn "Clean shutdown not possible due to missing information: instanceId"
	fi
	if [[ -n "${busDir}" ]]; then
		rm -rf "${busDir}"
	else
		pecho warn "Clean shutdown not possible due to missing information: busDir"
	fi
	if [[ -n "${appID}" ]]; then
		rm -rf "${XDG_RUNTIME_DIR}/.flatpak/${appID}"
		rm -rf "${XDG_RUNTIME_DIR}/portable/${appID}"
		rm -rf \
			"${XDG_DATA_HOME}/applications/${appID}.desktop" \
			2>/dev/null
	else
		pecho warn "Clean shutdown not possible due to missing information: appID"
	fi
	if [[ -e "${busDirAy}" ]]; then
		rm -rf "${busDirAy}"
	else
		pecho debug "Clean shutdown not possible due to missing information: busDirAy"
	fi
}

function stopApp() {
	if [[ "$*" =~ "external" ]]; then
		stopSlice
		exit 0
	elif [[ "$*" =~ "force" ]]; then
		pecho info "Force stop is called, killing service"
		stopSlice &
		systemctl \
			--user kill \
			-sSIGKILL \
			"${unitName}.service" 2>/dev/null &
	fi
	exit 0
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
		elif [[ "$1" = "--dbus-activation" ]]; then
			declare -g -i _portableBusActivate
			_portableBusActivate=1
		elif [[ "$1" =~ ^-v|--verbose$ ]]; then
			declare -g PORTABLE_LOGGING
			PORTABLE_LOGGING=debug
			declare -r -x PORTABLE_LOGGING
		elif [[ "$1" = "--actions" ]]; then
			shift
			cmdArgCount+=1
			if [[ "$1" = "debug-shell" ]]; then
				declare -g -i -x _portableDebug
				_portableDebug=1
				declare -r _portableDebug
			elif [[ "$1" =~ ^opendir|openhome$ ]]; then
				/usr/bin/xdg-open "${XDG_DATA_HOME}/${stateDirectory}"
				exit $?
			elif [[ "$1" =~ ^share-files|share-file$ ]]; then
				shareFile
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
if [[ "$*" = "--actions quit" ]]; then
	stopApp external
fi
questionFirstLaunch
manageDirs
cmdlineDispatcherv2 $@
launch $@