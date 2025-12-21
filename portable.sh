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
	echo "Visit https://github.com/Kraftland/portable for documentation and information."
	echo "To get started, specify a configuration file using the environment variable \"\${_portableConfig}\""
	exit 0
}

if [[ "${_portalConfig}" ]]; then
	export _portableConfig="${_portalConfig}"
	pecho warn "Using legacy configuration variable!"
fi

if [[ -z "${_portableConfig}" ]] || [[ $1 = "--help" ]]; then
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
		export _portableConfig="$(pwd)/${_portableConfig}"
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
				break
			else
				if [[ ! -e "${XDG_RUNTIME_DIR}/portable/${appID}/ready-${readyDir}" ]]; then
					exit 114
					break
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
			local fileCnt=$(find "${bwBindPar}" -maxdepth 1 -mindepth 1 -type f | wc -l)
			local dirCnt=$(find "${bwBindPar}" -maxdepth 1 -mindepth 1 -type d | wc -l)
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
	local mounts="$(systemd-run --quiet --user -P -- findmnt -R)"
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
}

function waylandDisplay() {
	if [[ "${XDG_SESSION_TYPE}" = "x11" ]]; then
		pecho warn "Running on X11, be warned!"
		wayDisplayBind="/$(uuidgen)/$(uuidgen)"
		return 0
	fi
	if [[ -z "${WAYLAND_DISPLAY}" ]]; then
		pecho debug "WAYLAND_DISPLAY not set, defaulting to wayland-0"
		wayDisplayBind="${XDG_RUNTIME_DIR}/wayland-0"
	fi
	if [[ -f "${WAYLAND_DISPLAY}" ]]; then
		pecho debug "Wayland display is specified as an absolute path"
		wayDisplayBind="${WAYLAND_DISPLAY}"
	elif [[ "${WAYLAND_DISPLAY}" =~ "wayland-" ]]; then
		pecho debug "Detected Wayland display as ${WAYLAND_DISPLAY}"
		wayDisplayBind="${XDG_RUNTIME_DIR}/${WAYLAND_DISPLAY}"
	fi
	waylandContext
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
	addEnv "XAUTHORITY=${XAUTHORITY}"
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

function inputBindv2() {
	if [[ "${bindInputDevices}" = "true" ]]; then
		bwInputArg="--dev-bind-try\0/sys/class/leds\0/sys/class/leds\0--dev-bind-try\0/sys/class/input\0/sys/class/input\0--dev-bind-try\0/sys/class/hidraw\0/sys/class/hidraw\0--dev-bind-try\0/dev/input\0/dev/input\0--dev-bind-try\0/dev/uinput\0/dev/uinput\0"
		for _device in /dev/hidraw*; do
			if [[ -e "${_device}" ]]; then
				bwInputArg="${bwInputArg}--dev-bind\0${_device}\0${_device}\0"
			fi
		done
		pecho warn "Detected input preference as expose."
		passBwrapArgs "${bwInputArg}"
	else
		bwInputArg=""
		pecho debug "Not exposing input devices."
	fi
	readyNotify set inputBindv2
}

function procDriverBind() {
	if [[ -d /proc/driver ]]; then
		passBwrapArgs "--tmpfs\0/proc/driver\0"
	fi
	readyNotify set procDriverBind
}

function bindNvDevIfExistv2(){
	if ls /dev/nvidia* &> /dev/null; then
		pecho debug "Binding NVIDIA GPUs in Game Mode / Single output configurations"
		for _card in /dev/nvidia*; do
			if [[ -e "${_card}" ]]; then
				bwSwitchableGraphicsArg="${bwSwitchableGraphicsArg}--dev-bind\0${_card}\0${_card}\0"
			fi
		done
		export nvExist=1
	fi
}

function setDiscBindArgv2() {
	export bwSwitchableGraphicsArg='--dev-bind\0/sys/bus/pci\0/sys/bus/pci\0'
	bwSwitchableGraphicsArg="${bwSwitchableGraphicsArg}--dev-bind\0$(find /sys/devices -maxdepth 1 -name 'pci*' | head -n 1)\0$(find /sys/devices -maxdepth 1 -name 'pci*' | head -n 1)\0"
}

function hybridBindv2() {
	local cardSums="$(find /sys/class/drm -name 'card*' -not -name '*-*' | wc -l)"
	if [[ "${cardSums}" -eq 1 || "${PORTABLE_ASSUME_SINGLE_GPU}" -eq 114514 ]]; then
		bwSwitchableGraphicsArg=""
		pecho debug "Single GPU"
		#setDiscBindArgv2
		bindNvDevIfExistv2
		local vCards="$(find /sys/class/drm -name 'card*' -not -name '*-*')"
		local vCards="$(basename ${vCards})"
		bwSwitchableGraphicsArg="${bwSwitchableGraphicsArg}--dev-bind\0$(resolvePCICard "${vCards}")\0$(resolvePCICard "${vCards}")\0"
	elif [[ "${cardSums}" -eq 0 ]]; then
		bwSwitchableGraphicsArg="--tmpfs\0/dev/dri\0--tmpfs\0/sys/class/drm\0"
		pecho warn "No GPU detected!"
		setDiscBindArgv2
		bindNvDevIfExistv2
	elif [[ "${gameMode}" = "true" ]]; then
		pecho debug "Game Mode enabled on hybrid graphics"
		setDiscBindArgv2
		bindNvDevIfExistv2
		setNvOffloadEnv
	else
		bwSwitchableGraphicsArg="--tmpfs\0/dev/dri\0--tmpfs\0/sys/class/drm\0"
		local activeCardSum=0
		activeCards="placeholder"
		for vCards in $(find /sys/class/drm -name 'card*' -not -name '*-*'); do
			pecho debug "Working on ${vCards}"
			for file in $(find -L "${vCards}" -maxdepth 2 -name status 2>/dev/null); do
				pecho debug "Inspecting ${file}"
				if grep -q "disconnected" "${file}"; then
					continue
				else
					pecho debug "Active GPU"
					activeCardSum=$(("${activeCardSum}"+1))
					if [[ "${activeCards}" = "placeholder" ]]; then
						activeCards="$(basename "${vCards}")"
					else
						activeCards="${activeCards} $(basename "${vCards}")"
					fi
					break
				fi
			done
		done
		if [[ "${activeCardSum}" -le 1 ]]; then
			for _module in $(find /sys/module -maxdepth 1 -type d -name 'nvidia*'); do
				bwSwitchableGraphicsArg="${bwSwitchableGraphicsArg}--tmpfs\0${_module}\0"
			done
			pecho debug "${activeCardSum} card active, identified as ${activeCards}"
			addEnv "VK_LOADER_DRIVERS_DISABLE='nvidia_icd.json'"
			cardToRender "${activeCards}"
			bwSwitchableGraphicsArg="${bwSwitchableGraphicsArg}--dev-bind-try\0/sys/class/drm/${activeCards}\0/sys/class/drm/${activeCards}\0--dev-bind-try\0/dev/dri/${activeCards}\0/dev/dri/${activeCards}\0--dev-bind\0/dev/dri/${renderIndex}\0/dev/dri/${renderIndex}\0--dev-bind\0/sys/class/drm/${renderIndex}\0/sys/class/drm/${renderIndex}\0--dev-bind\0$(resolvePCICard "${activeCards}")\0$(resolvePCICard "${activeCards}")\0"
		else
			pecho warn "Multiple GPU outputs detected! Report bugs if found."
			pecho debug "${activeCardSum} cards active"
			for vCards in ${activeCards}; do
			# TODO: What happens to non NVIDIA, more than 1 active GPU hybrid configuration?
				if grep -q '0x10de' "/sys/class/drm/${vCards}/device/vendor"; then
					addEnv "VK_LOADER_DRIVERS_DISABLE=nvidia_icd.json"
					continue
				else
					cardToRender "${vCards}"
					pecho debug "Binding ${renderIndex}"
					bwSwitchableGraphicsArg="${bwSwitchableGraphicsArg}--dev-bind-try\0/sys/class/drm/${vCards}\0/sys/class/drm/${vCards}\0--dev-bind-try\0/dev/dri/${vCards}\0/dev/dri/${vCards}\0--dev-bind\0/dev/dri/${renderIndex}\0/dev/dri/${renderIndex}\0--dev-bind\0/sys/class/drm/${renderIndex}\0/sys/class/drm/${renderIndex}\0--dev-bind\0$(resolvePCICard "${vCards}")\0$(resolvePCICard "${vCards}")\0"
					addEnv 'DRI_PRIME=0'
				fi
			done
		fi
	fi
	pecho debug "(v2) Generated GPU bind parameter: ${bwSwitchableGraphicsArg}"
	passBwrapArgs "${bwSwitchableGraphicsArg}"
	readyNotify set hybridBindv2
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

function pwBindCalc() {
	if [[ "${bindPipewire}" = 'true' ]]; then
		readyNotify wait pwSecContext
		getBusArgs pwSecContext
		passBwrapArgs "--bind-try\0${pwSecContext}\0${XDG_RUNTIME_DIR}/pipewire-0\0"
		pecho debug "Bound PipeWire socket w/ security context"
	fi
	readyNotify set pwBindCalc
}

function cameraBindv2() {
	local bwCamPar=""
	if [[ "${bindCameras}" = "true" ]]; then
		pecho debug "Detecting Camera..."
		for _camera in /dev/video*; do
			if [[ -e "${_camera}" ]]; then
				bwCamPar="${bwCamPar}--dev-bind\0${_camera}\0${_camera}\0"
			fi
		done
		pecho debug "Generated Camera bind parameter: ${bwCamPar}"
		passBwrapArgs "${bwCamPar}"
	fi
	readyNotify set cameraBindv2
}

function calcBwrapArg() {
	echo "false" > "${XDG_RUNTIME_DIR}/portable/${appID}/startSignal"
	sync "${XDG_RUNTIME_DIR}/portable/${appID}/startSignal"
	rm -f "${XDG_RUNTIME_DIR}/portable/${appID}/bwrapArgs"

	# Build sd-run args first!
	if [[ "${bindNetwork}" = "false" ]]; then
		pecho info "Network access disabled via config"
		sdNetArg="PrivateNetwork=yes"
	else
		sdNetArg="PrivateNetwork=no"
		pecho debug "Network access allowed"
	fi
	passBwrapArgs "--quiet\0--user\0--pty\0--service-type=notify-reload\0--wait\0-u\0${unitName}\0--slice=app.slice\0-p\0Delegate=yes\0-p\0BindsTo=${proxyName}.service\0-p\0Description=Portable Sandbox for ${appID}\0-p\0Documentation=https://github.com/Kraftland/portable\0-p\0ExitType=cgroup\0-p\0NotifyAccess=all\0-p\0TimeoutStartSec=infinity\0-p\0OOMPolicy=stop\0-p\0SecureBits=noroot-locked\0-p\0KillMode=control-group\0-p\0MemoryHigh=90%\0-p\0ManagedOOMSwap=kill\0-p\0ManagedOOMMemoryPressure=kill\0-p\0IPAccounting=yes\0-p\0MemoryPressureWatch=yes\0-p\0SyslogIdentifier=portable-${appID}\0-p\0SystemCallLog=@privileged @debug @cpu-emulation @obsolete io_uring_enter io_uring_register io_uring_setup @resources\0-p\0SystemCallLog=~@sandbox\0-p\0PrivateIPC=yes\0-p\0ProtectClock=yes\0-p\0CapabilityBoundingSet=\0-p\0RestrictSUIDSGID=yes\0-p\0LockPersonality=yes\0-p\0RestrictRealtime=yes\0-p\0ProtectProc=invisible\0-p\0ProcSubset=pid\0-p\0PrivateUsers=yes\0-p\0UMask=077\0-p\0OOMScoreAdjust=100\0-p\0NoNewPrivileges=yes\0-p\0ProtectControlGroups=private\0-p\0DelegateSubgroup=portable-cgroup\0-p\0PrivateMounts=yes\0-p\0KeyringMode=private\0-p\0TimeoutStopSec=10s\0-p\0Environment=instanceId=${instanceId}\0-p\0Environment=busDir=${busDir}\0-p\0${sdNetArg}\0-p\0WorkingDirectory=${XDG_DATA_HOME}/${stateDirectory}\0"

	passBwrapArgs "-p\0ReloadSignal=SIGALRM\0-p\0EnvironmentFile=${XDG_RUNTIME_DIR}/portable/${appID}/portable-generated.env\0"
	passBwrapArgs "-p\0SystemCallFilter=~@clock\0-p\0SystemCallFilter=~@cpu-emulation\0-p\0SystemCallFilter=~@debug\0-p\0SystemCallFilter=~@module\0-p\0SystemCallFilter=~@obsolete\0-p\0SystemCallFilter=~@raw-io\0-p\0SystemCallFilter=~@reboot\0-p\0SystemCallFilter=~@swap\0-p\0SystemCallErrorNumber=EAGAIN\0-p\0ProtectHome=no\0-p\0UnsetEnvironment=GNOME_SETUP_DISPLAY\0-p\0UnsetEnvironment=PIPEWIRE_REMOTE\0-p\0UnsetEnvironment=PAM_KWALLET5_LOGIN\0-p\0UnsetEnvironment=GTK2_RC_FILES\0-p\0UnsetEnvironment=ICEAUTHORITY\0-p\0UnsetEnvironment=MANAGERPID\0"

	# Do we really need this?
	#passBwrapArgs "-p\0LimitNOFILE=524288\0"


	passBwrapArgs "--\0bwrap\0--new-session\0--unshare-cgroup-try\0--unshare-ipc\0--unshare-uts\0--unshare-pid\0--unshare-user\0" # Unshares
	passBwrapArgs "--tmpfs\0/tmp\0--bind-try\0/tmp/.X11-unix\0/tmp/.X11-unix\0--bind-try\0/tmp/.XIM-unix\0/tmp/.XIM-unix\0" # /tmp binds
	passBwrapArgs "--dev\0/dev\0--tmpfs\0/dev/shm\0--dev-bind-try\0/dev/mali\0/dev/mali\0--dev-bind-try\0/dev/mali0\0/dev/mali0\0--dev-bind-try\0/dev/umplock\0/dev/umplock\0--mqueue\0/dev/mqueue\0--dev-bind\0/dev/dri\0/dev/dri\0--dev-bind-try\0/dev/udmabuf\0/dev/udmabuf\0--dev-bind-try\0/dev/ntsync\0/dev/ntsync\0--dir\0/top.kimiblock.portable\0" # Dev binds
	passBwrapArgs "--tmpfs\0/sys\0--ro-bind-try\0/sys/module\0/sys/module\0--ro-bind-try\0/sys/dev/char\0/sys/dev/char\0--tmpfs\0/sys/devices\0--ro-bind-try\0/sys/fs/cgroup\0/sys/fs/cgroup\0--ro-bind-try\0/sys/fs/cgroup/portable-cgroup\0/sys/fs/cgroup/portable-cgroup\0--dev-bind\0/sys/class/drm\0/sys/class/drm\0" # sys entries
	inputBindv2 &
	passBwrapArgs "--bind\0/usr\0/usr\0--overlay-src\0/usr/bin\0--overlay-src\0/usr/lib/portable/overlay-usr\0--ro-overlay\0/usr/bin\0--proc\0/proc\0--dev-bind-try\0/dev/null\0/dev/null\0--ro-bind-try\0/dev/null\0/proc/uptime\0--ro-bind-try\0/dev/null\0/proc/modules\0--ro-bind-try\0/dev/null\0/proc/cmdline\0--ro-bind-try\0/dev/null\0/proc/diskstats\0--ro-bind-try\0/dev/null\0/proc/devices\0--ro-bind-try\0/dev/null\0/proc/config.gz\0--ro-bind-try\0/dev/null\0/proc/mounts\0--ro-bind-try\0/dev/null\0/proc/loadavg\0--ro-bind-try\0/dev/null\0/proc/filesystems\0--symlink\0/usr/lib\0/lib\0--symlink\0/usr/lib\0/lib64\0--symlink\0/usr/bin\0/bin\0--symlink\0/usr/bin\0/sbin\0"
	if [ -e /usr/lib/flatpak-xdg-utils/flatpak-spawn ]; then
		passBwrapArgs "--ro-bind\0/usr/lib/portable/overlay-usr/flatpak-spawn\0/usr/lib/flatpak-xdg-utils/flatpak-spawn\0"
	fi
	passBwrapArgs "--perms\00000\0--tmpfs\0/boot\0--perms\00000\0--tmpfs\0/srv\0--perms\00000\0--tmpfs\0/root\0--perms\00000\0--tmpfs\0/media\0--perms\00000\0--tmpfs\0/mnt\0--tmpfs\0/home\0--tmpfs\0/var\0--symlink\0/run\0/var/run\0--symlink\0/run/lock\0/var/lock\0--tmpfs\0/var/empty\0--tmpfs\0/var/lib\0--perms\00000\0--tmpfs\0/var/log\0--perms\00000\0--tmpfs\0/var/opt\0--perms\00000\0--tmpfs\0/var/spool\0--tmpfs\0/var/tmp\0--ro-bind-try\0/var/cache/fontconfig\0/var/cache/fontconfig\0--ro-bind-try\0/opt\0/opt\0" # Create various directories for FHS
	passBwrapArgs "--bind\0${XDG_RUNTIME_DIR}/portable/${appID}\0/run\0--bind\0${XDG_RUNTIME_DIR}/portable/${appID}\0${XDG_RUNTIME_DIR}/portable/${appID}\0--ro-bind-try\0/run/systemd/userdb/io.systemd.Home\0/run/systemd/userdb/io.systemd.Home\0--ro-bind\0${xAuthBind}\0/run/.Xauthority\0--ro-bind\0${busDir}/bus\0/run/sessionBus\0--ro-bind-try\0${busDirAy}\0${XDG_RUNTIME_DIR}/at-spi\0--dir\0/run/host\0--bind\0${XDG_RUNTIME_DIR}/doc/by-app/${appID}\0${XDG_RUNTIME_DIR}/doc\0--ro-bind\0/dev/null\0${XDG_RUNTIME_DIR}/.flatpak/${instanceId}-private/run-environ\0--ro-bind\0${XDG_RUNTIME_DIR}/.flatpak/${instanceId}\0${XDG_RUNTIME_DIR}/.flatpak/${instanceId}\0--ro-bind\0${XDG_RUNTIME_DIR}/.flatpak/${instanceId}\0${XDG_RUNTIME_DIR}/flatpak-runtime-directory\0--ro-bind-try\0${wayDisplayBind}\0${XDG_RUNTIME_DIR}/wayland-0\0--ro-bind-try\0/run/systemd/resolve/stub-resolv.conf\0/run/systemd/resolve/stub-resolv.conf\0--bind\0${XDG_RUNTIME_DIR}/systemd/notify\0${XDG_RUNTIME_DIR}/systemd/notify\0" # Run binds

	passBwrapArgs "--bind\0${XDG_DATA_HOME}/${stateDirectory}\0${HOME}\0--bind\0${XDG_DATA_HOME}/${stateDirectory}\0${XDG_DATA_HOME}/${stateDirectory}\0" # HOME binds
	calcMountArgv2 &
	pwBindCalc &
	cameraBindv2 &
	passBwrapArgs "--ro-bind-try\0${XDG_RUNTIME_DIR}/pulse\0${XDG_RUNTIME_DIR}/pulse\0" # PulseAudio Bind!
	hybridBindv2 &
	procDriverBind &
	passBwrapArgs "--ro-bind\0/etc\0/etc\0--tmpfs\0/etc/kernel\0"
	passBwrapArgs "--tmpfs\0/proc/1\0--tmpfs\0/usr/share/applications\0--tmpfs\0${HOME}/options\0--tmpfs\0${XDG_DATA_HOME}/${stateDirectory}/options\0--tmpfs\0${HOME}/.var\0--tmpfs\0${XDG_DATA_HOME}/${stateDirectory}/.var\0--bind\0${XDG_DATA_HOME}/${stateDirectory}\0${XDG_DATA_HOME}/${stateDirectory}/.var/app/${appID}\0--bind\0${XDG_DATA_HOME}/${stateDirectory}\0${HOME}/.var/app/${appID}\0--tmpfs\0${HOME}/.var/app/${appID}/options\0--tmpfs\0${XDG_DATA_HOME}/${stateDirectory}/.var/app/${appID}/options\0" # Hide some entries
	readyNotify wait bindCheck
	if [[ -z "${bwBindPar}" || ! -e "${bwBindPar}" ]]; then
		unset bwBindPar
	else
		passBwrapArgs "--dev-bind\0${bwBindPar}\0${bwBindPar}\0"
	fi
	readyNotify wait procDriverBind
	readyNotify wait inputBindv2
	readyNotify wait hybridBindv2
	readyNotify wait pwBindCalc # PW binds
	readyNotify wait cameraBindv2
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
	desktopWorkaround &
	addEnv targetArgs="${targetArgs}"
	addEnv _portableDebug="${_portableDebug}"
	addEnv _portableBusActivate="${_portableBusActivate}"
	termExec
	readyNotify wait generateFlatpakInfo
	terminateOnRequest &
	readyNotify wait calcBwrapArg
	xargs \
		-0 \
		-a "${XDG_RUNTIME_DIR}/portable/${appID}/bwrapArgs" \
		systemd-run
	stopApp
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

function desktopWorkaround() {
	dbus-send --session \
		--dest=org.freedesktop.impl.portal.PermissionStore \
		/org/freedesktop/impl/portal/PermissionStore \
		org.freedesktop.impl.portal.PermissionStore.SetPermission \
		string:"background" boolean:true string:"background" string:"${appID}" array:string:"yes" &
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
	local symOrig="$(realpath /sys/class/drm/"$1"/../)"
	renderIndex="$(find "${symOrig}" -maxdepth 1 -mindepth 1 -name 'render*' -print -quit)"
	renderIndex="$(basename "${renderIndex}")"
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
	readlink --quiet --no-newline --canonicalize /sys/class/drm/$1/../../
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

function genInstanceID() {
	instanceId=$(shuf -i 1024000000-9999999999 -n 1)
	while [[ -d "${XDG_RUNTIME_DIR}/.flatpak/${instanceId}" ]]; do
		pecho debug "Instance ID collision detected!"
		instanceId=$(shuf -i 1024000000-9999999999 -n 1)
	done

}

function generateFlatpakInfo() {
	pecho debug "Installing flatpak-info..."
	install /usr/lib/portable/flatpak-info \
		"${XDG_RUNTIME_DIR}/portable/${appID}/flatpak-info"
	sed -i "s|placeHolderAppName|${appID}|g" \
		"${XDG_RUNTIME_DIR}/portable/${appID}/flatpak-info"
	sed -i "s|placeholderInstanceId|${instanceId}|g" \
		"${XDG_RUNTIME_DIR}/portable/${appID}/flatpak-info"
	sed -i "s|placeholderPath|${XDG_DATA_HOME}/${stateDirectory}|g" \
		"${XDG_RUNTIME_DIR}/portable/${appID}/flatpak-info"
	mkdir \
		--parents \
		--mode=0700 \
		"${XDG_RUNTIME_DIR}/.flatpak/${instanceId}"
	install "${XDG_RUNTIME_DIR}/portable/${appID}/flatpak-info" \
		"${XDG_RUNTIME_DIR}/.flatpak/${instanceId}/info"
	mkdir \
		--parents \
		--mode=0700 \
		"${XDG_RUNTIME_DIR}/.flatpak/${appID}/xdg-run"
	mkdir \
		--parents \
		--mode=0700 \
		"${XDG_RUNTIME_DIR}/.flatpak/${appID}/tmp"
	touch "${XDG_RUNTIME_DIR}/.flatpak/${appID}/.ref"
	echo "instanceId=${instanceId}" > "${XDG_RUNTIME_DIR}/portable/${appID}/control"
	echo "appID=${appID}" >> "${XDG_RUNTIME_DIR}/portable/${appID}/control"
	echo "busDir=${busDir}" >> "${XDG_RUNTIME_DIR}/portable/${appID}/control"
	echo "busDirAy=${busDirAy}" >> "${XDG_RUNTIME_DIR}/portable/${appID}/control"
	echo "friendlyName=${friendlyName}" >> "${XDG_RUNTIME_DIR}/portable/${appID}/control"
	if [[ -f "/usr/share/applications/${appID}.desktop" ]]; then
		pecho debug "Application desktop file detected"
	else
		pecho warn ".desktop file missing!"
		cat <<- 'EOF' > "${XDG_RUNTIME_DIR}/portable/${appID}/desktop.file"
			[Desktop Entry]
			Name=placeholderName
			Exec=env _portableConfig=placeholderConfig portable
			Terminal=false
			Type=Application
			Icon=image-missing
			Comment=Application info missing
			Categories=Utility;
		EOF
		sed -i \
			"s|placeholderConfig|$(pathEscape "${_portableConfig}")|g" \
			"${XDG_RUNTIME_DIR}/portable/${appID}/desktop.file"
		sed -i \
			"s|placeholderName|$(pathEscape "${appID}")|g" \
			"${XDG_RUNTIME_DIR}/portable/${appID}/desktop.file"
		install -Dm600 \
			"${XDG_RUNTIME_DIR}/portable/${appID}/desktop.file" \
			"${XDG_DATA_HOME}/applications/${appID}.desktop"
	fi
	readyNotify set generateFlatpakInfo
}

function resetUnit() {
	if [[ $(systemctl --user is-failed "${1}".service) = "failed" ]]; then
		pecho warn "${1} failed last time"
		systemctl --user reset-failed "${1}".service
	fi
}

function addDbusArg() {
	if [[ -z "${extraDbusArgs}" ]]; then
		extraDbusArgs="$*"
	else
		extraDbusArgs="${extraDbusArgs} $*"
	fi
}

# $1 as arg name, $2 as value
function passBusArgs() {
	echo "$2" >"${XDG_RUNTIME_DIR}/portable/${appID}/busstore/$1"
}

# $1 as arg name.
function getBusArgs() {
	export "$1=$(cat "${XDG_RUNTIME_DIR}/portable/${appID}/busstore/$1")" 2>/dev/null
}

function cleanDUnits() {
	systemctl --user kill -sSIGKILL \
		"${friendlyName}*" \
		"${unitName}" \
		"${proxyName}*".service \
		"${proxyName}-a11y" \
		"${friendlyName}"-wayland-proxy \
		"${unitName}-pipewire-container" \
		"${friendlyName}-subprocess*".service 2>/dev/null
	systemctl --user reset-failed \
		"${friendlyName}*" \
		"${unitName}" \
		"${proxyName}*".service \
		"${proxyName}-a11y" \
		"${friendlyName}"-wayland-proxy \
		"${unitName}-pipewire-container" \
		"${friendlyName}-subprocess*".service 2>/dev/null &
	systemctl --user clean "${friendlyName}*" \
		"${unitName}" \
		"${friendlyName}-subprocess*".service \
		"${proxyName}*".service \
		"${proxyName}-a11y" \
		"${friendlyName}"-wayland-proxy \
		"${friendlyName}*"-pipewire-container.service 2>/dev/null
	readyNotify set cleanDUnits
}

function dbusArg() {
	mkdir -p "${XDG_RUNTIME_DIR}/portable/${appID}/busstore"
	if [[ "${PORTABLE_LOGGING}" = "debug" ]]; then
		proxyArg="--log"
	fi
	if [[ "${XDG_CURRENT_DESKTOP}" = "GNOME" ]]; then
		local featureSet="Location"
		pecho info "Enabling GNOME exclusive features: ${featureSet}"
		addDbusArg \
			"--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Location --call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Location.*"
	fi
	mkdir \
		--parents \
		--mode=0700 \
		"${XDG_RUNTIME_DIR}/doc/by-app/${appID}"
	local \
	defaultMprisOwn="--own=org.mpris.MediaPlayer2.${appID##*.}.* --own=org.mpris.MediaPlayer2.${appID##*.} --own=org.mpris.MediaPlayer2.${appID} --own=org.mpris.MediaPlayer2.${appID}.*"
	if [[ -n "${mprisName}" ]]; then
		local mprisBus="org.mpris.MediaPlayer2.${mprisName}"
		addDbusArg \
			"--own=${mprisBus} --own=${mprisBus}.* ${defaultMprisOwn}"
	else
		addDbusArg \
			"${defaultMprisOwn}"
	fi
	if [[ "${allowGlobalShortcuts}" = "true" ]]; then
		addDbusArg \
			"--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.GlobalShortcuts --call=org.freedesktop.portal.Desktop=org.freedesktop.portal.GlobalShortcuts.*"
	fi
	if [[ "${allowInhibit}" = "true" ]]; then
		addDbusArg "--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Inhibit --call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Inhibit.*"
	fi
	pecho debug "Extra D-Bus arguments: ${extraDbusArgs}"
	passBusArgs extraDbusArgs "${extraDbusArgs}"
	passBusArgs proxyArg "${proxyArg}"
	readyNotify set dbusArg
}

function writeInfo() {
	pecho debug "Waiting for bwrapinfo.json"
	until grep child-pid -q "${XDG_RUNTIME_DIR}/.flatpak/${instanceId}/bwrapinfo.json.original" 1>/dev/null 2>/dev/null; do
		inotifywait \
			-e modify,create,attrib,close \
			--quiet \
			"${XDG_RUNTIME_DIR}/.flatpak/${instanceId}" \
			1>/dev/null
	done
	head -n 1 \
		"${XDG_RUNTIME_DIR}/.flatpak/${instanceId}/bwrapinfo.json.original" \
		> "${XDG_RUNTIME_DIR}/.flatpak/${instanceId}/bwrapinfo.json"
	pecho debug "bwrapinfo.json installed"
	readyNotify set writeInfo
}

function pwSecContext() {
	if [[ "${bindPipewire}" = 'true' ]]; then
		pecho debug "Pipewire security context enabled"
		rm -f "${XDG_RUNTIME_DIR}/portable/${appID}/pipewire-socket"
		systemd-run \
			--user \
			--quiet \
			-p Slice="portable-${friendlyName}.slice" \
			-u "${unitName}-pipewire-container" \
			-p KillMode=control-group \
			-p After="pipewire.service" \
			-p Wants="pipewire.service" \
			-p StandardOutput="file:${XDG_RUNTIME_DIR}/portable/${appID}/pipewire-socket" \
			-p SuccessExitStatus=SIGKILL \
			-p Requires=pipewire.service \
			-- \
			"stdbuf" \
			"-oL" \
			"/usr/bin/pw-container" \
			"-P" \
			'{ "pipewire.sec.engine": "top.kimiblock.portable", "pipewire.access": "restricted" }'

		if grep -q "new socket" "${XDG_RUNTIME_DIR}/portable/${appID}/pipewire-socket"; then
			pecho debug "Pipewire socket created"
		else
			while true; do
				sleep 0.0001s
				if [[ ! -d "${XDG_RUNTIME_DIR}/portable/${appID}" || ! -e "${XDG_RUNTIME_DIR}/portable/${appID}/ready-${readyDir}" ]]; then
					break
				elif grep -q "new socket" "${XDG_RUNTIME_DIR}/portable/${appID}/pipewire-socket"; then
					break
				fi
			done
			pecho debug "Pipewire socket created after waiting"
		fi
		passBusArgs \
			pwSecContext \
			"$(cat "${XDG_RUNTIME_DIR}/portable/${appID}/pipewire-socket" | sed 's|new socket: ||g')"
	fi
	readyNotify set pwSecContext
}

function dbusProxy() {
	genInstanceID
	generateFlatpakInfo &
	importEnv &
	dbusArg &
	cleanDUnits &
	genXAuth
	waylandDisplay
	mkdir \
		--parents \
		--mode=0700 \
		"${busDir}"
	mkdir \
		--parents \
		--mode=0700 \
		"${busDirAy}"
	pecho info "Starting D-Bus Proxy @ ${busDir}..."
	readyNotify wait dbusArg
	readyNotify wait cleanDUnits
	pwSecContext &
	getBusArgs extraDbusArgs
	getBusArgs proxyArg
	systemd-run \
		--user \
		--quiet \
		-p Slice="portable-${friendlyName}.slice" \
		-u "${proxyName}" \
		-p KillMode=control-group \
		-p Wants="xdg-document-portal.service xdg-desktop-portal.service" \
		-p After="xdg-document-portal.service xdg-desktop-portal.service" \
		-p SuccessExitStatus=SIGKILL \
		-p StandardError="file:${XDG_RUNTIME_DIR}/.flatpak/${instanceId}/bwrapinfo.json" \
		-- bwrap \
			--json-status-fd 2 \
			--unshare-all \
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
			"${DBUS_SESSION_BUS_ADDRESS}" \
			"${busDir}/bus" \
			${proxyArg} \
			--filter \
			--own=org.kde.StatusNotifierItem-2-1 \
			--own=org.kde.StatusNotifierItem-3-1 \
			--own=org.kde.StatusNotifierItem-4-1 \
			--own=org.kde.StatusNotifierItem-5-1 \
			--own=org.kde.StatusNotifierItem-6-1 \
			--own=org.kde.StatusNotifierItem-7-1 \
			--own=org.kde.StatusNotifierItem-8-1 \
			--own=org.kde.StatusNotifierItem-9-1 \
			--own=org.kde.StatusNotifierItem-10-1 \
			--own=org.kde.StatusNotifierItem-11-1 \
			--own=org.kde.StatusNotifierItem-12-1 \
			--own=org.kde.StatusNotifierItem-13-1 \
			--own=org.kde.StatusNotifierItem-14-1 \
			--own=org.kde.StatusNotifierItem-15-1 \
			--own=org.kde.StatusNotifierItem-16-1 \
			--own=org.kde.StatusNotifierItem-17-1 \
			--own=org.kde.StatusNotifierItem-18-1 \
			--own=org.kde.StatusNotifierItem-19-1 \
			--own=org.kde.StatusNotifierItem-20-1 \
			--own=org.kde.StatusNotifierItem-21-1 \
			--own=org.kde.StatusNotifierItem-22-1 \
			--own=org.kde.StatusNotifierItem-23-1 \
			--own=org.kde.StatusNotifierItem-24-1 \
			--own=org.kde.StatusNotifierItem-25-1 \
			--own=org.kde.StatusNotifierItem-26-1 \
			--own=org.kde.StatusNotifierItem-27-1 \
			--own=org.kde.StatusNotifierItem-28-1 \
			--own=org.kde.StatusNotifierItem-29-1 \
			--own=com.belmoussaoui.ashpd.demo \
			--talk="org.unifiedpush.Distributor.*" \
			--own="${appID}" \
			--own="${appID}".* \
			--talk=org.freedesktop.Notifications \
			--talk=org.kde.StatusNotifierWatcher \
			--call=org.freedesktop.Notifications.*=* \
			--see=org.a11y.Bus \
			--call=org.a11y.Bus=org.a11y.Bus.GetAddress@/org/a11y/bus \
			--call=org.a11y.Bus=org.freedesktop.DBus.Properties.Get@/org/a11y/bus \
			--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Screenshot --call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Screenshot.Screenshot \
			--see=org.freedesktop.portal.Request \
			--talk=com.canonical.AppMenu.Registrar \
			--call=org.freedesktop.portal.Desktop=org.freedesktop.DBus.Properties.GetAll \
			--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Session.Close \
			--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Settings.ReadAll \
			--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Email.ComposeEmail \
			--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Usb \
			--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Usb.* \
			--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.ProxyResolver.Lookup \
			--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.ProxyResolver.Lookup.* \
			--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.ScreenCast \
			--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.ScreenCast.* \
			--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Account.GetUserInformation \
			--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Camera.* \
			--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Camera \
			--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.RemoteDesktop.* \
			--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.RemoteDesktop \
			--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Settings.Read \
			--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Request \
			--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Documents.* \
			--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Documents \
			--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Device \
			--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Device.* \
			--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.FileChooser.* \
			--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.FileChooser \
			--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.FileTransfer.* \
			--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.FileTransfer \
			--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Notification.* \
			--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Notification \
			--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Print.* \
			--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Print \
			--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.NetworkMonitor.* \
			--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.NetworkMonitor \
			--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.OpenURI.* \
			--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.OpenURI \
			--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Fcitx.* \
			--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Fcitx \
			--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.IBus.* \
			--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.IBus \
			--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.IBus \
			--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Secret \
			--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.Secret.RetrieveSecret \
			${extraDbusArgs} \
			--call=org.freedesktop.portal.Desktop=org.freedesktop.DBus.Properties.Get@/org/freedesktop/portal/desktop \
			--talk=org.freedesktop.portal.Documents \
			--call=org.freedesktop.portal.Documents=* \
			--talk=org.freedesktop.portal.FileTransfer \
			--call=org.freedesktop.portal.FileTransfer=* \
			--talk=org.freedesktop.portal.FileTransfer.* \
			--call=org.freedesktop.portal.FileTransfer.*=* \
			--talk=org.freedesktop.portal.Notification \
			--call=org.freedesktop.portal.Notification=* \
			--talk=org.freedesktop.portal.Print \
			--call=org.freedesktop.portal.Print=* \
			--talk=org.freedesktop.FileManager1 \
			--call=org.freedesktop.FileManager1=* \
			--talk=org.freedesktop.portal.OpenURI \
			--call=org.freedesktop.portal.OpenURI=* \
			--talk=org.freedesktop.portal.OpenURI.OpenURI \
			--call=org.freedesktop.portal.OpenURI.OpenURI=* \
			--talk=org.freedesktop.portal.OpenURI.OpenFile \
			--call=org.freedesktop.portal.OpenURI.OpenFile=* \
			--talk=org.freedesktop.portal.Fcitx \
			--call=org.freedesktop.portal.Fcitx=* \
			--talk=org.freedesktop.portal.Fcitx.* \
			--call=org.freedesktop.portal.Fcitx.*=* \
			--talk=org.freedesktop.portal.IBus \
			--call=org.freedesktop.portal.IBus=* \
			--talk=org.freedesktop.portal.IBus.* \
			--call=org.freedesktop.portal.IBus.*=* \
			--call=org.freedesktop.portal.Request=* \
			--broadcast=org.freedesktop.portal.*=@/org/freedesktop/portal/*
	#writeInfo &
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
	resetUnit "${proxyName}"
	resetUnit "${friendlyName}" &
	resetUnit "${proxyName}-a11y" &
	resetUnit "${friendlyName}-wayland-proxy"
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
					--text "用户数据将不再被保护"
			else
				zenity \
					--question \
					--default-cancel \
					--title "Confirm action" \
					--icon=security-low-symbolic \
					--text "User data may be compromised"
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
	if [[ "$*" =~ "--actions" && "$*" =~ "debug-shell" ]]; then
		declare -g _portableDebug=1
	elif [[ "$*" =~ "--dbus-activation" ]]; then
		declare -g _portableBusActivate=1
	fi
	if systemctl --user --quiet is-failed "${unitName}.service"; then
		pecho warn "${appID} failed last time"
		systemctl --user reset-failed "${unitName}.service" &
	fi
	if systemctl --user --quiet is-active "${unitName}.service"; then
		warnMulRunning "$@"
	elif systemctl --user --quiet is-active "${friendlyName}.service"; then
		warnMulRunning "$@"
	fi
	sanityCheck &
	if [[ ${trashAppUnsafe} -eq 1 ]]; then
		pecho warn "Launching ${appID} (unsafe)..."
		execAppUnsafe
	else
		dbusProxy
		pecho info "Launching ${appID}..."
		execApp
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
		systemctl \
			--user kill \
			-sSIGKILL \
			"${unitName}.service" 2>/dev/null &
	fi
	stopSlice &
	cleanDirs &
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

function cmdlineDispatcher() {
	if [[ "$*" =~ "f5aaebc6-0014-4d30-beba-72bce57e0650" ]] && [[ "$*" =~ "--actions" ]]; then
		rm -f "${XDG_DATA_HOME}/${stateDirectory}/options/sandbox"
		questionFirstLaunch
	fi
	if [[ "$*" =~ "--actions" ]] && [[ "$*" =~ "opendir" ]]; then
		export targetArgs=""
		/usr/bin/xdg-open "${XDG_DATA_HOME}/${stateDirectory}"
		exit "$?"
	fi
	if [[ "$*" =~ "--actions" ]] && [[ "$*" =~ "share-files" ]]; then
		export targetArgs=""
		shareFile
	fi
	if [[ "$*" =~ "--actions" ]] && [[ "$*" =~ "reset-documents" ]]; then
		export targetArgs=""
		resetDocuments
	fi
	if [[ "$*" =~ "--actions" ]] && [[ "$*" =~ "stat" ]]; then
		export targetArgs=""
		showStats
	fi
	while [[ $# -gt 0 ]]; do
		if [[ "$1" = "--" ]]; then
			shift
			break
		fi
		shift
	done
	export targetArgs="$*"
	pecho info "Application argument interpreted as: ${targetArgs}"
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
cmdlineDispatcher "$@"
launch "$@"
