#!/bin/bash

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

if [[ "${_portalConfig}" ]]; then
	export _portableConfig="${_portalConfig}"
	pecho warn "Using legacy configuration variable!"
fi

if [[ -z "${_portableConfig}" ]]; then
	pecho crit "No portable config specified!"
	exit 1
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
		pecho crit "Specified config cannot be found or read!"
		exit 1
	fi
fi

busName="${appID}"
busDir="${XDG_RUNTIME_DIR}/app/${busName}"
busDirAy="${XDG_RUNTIME_DIR}/app/${busName}-a11y"
unitName="${friendlyName}"
proxyName="${friendlyName}-dbus"

function readyNotify() {
	# Notifies readiness, only usable after warnMulRunning()
	# $1 can be: wait, set, set-fail, init
	# $2 is the item name
	if [[ ${trashAppUnsafe} -eq 1 ]]; then
		return 1
	fi
	if [[ $1 = "set" ]]; then
		echo "ready" >"${XDG_RUNTIME_DIR}/portable/${appID}/ready-${readyDir}/$2"
	elif [[ $1 = "set-fail" ]]; then
		echo "abort" >"${XDG_RUNTIME_DIR}/portable/${appID}/ready-${readyDir}/$2"
	elif [[ $1 = "init" ]]; then
		readyDir="$(xxd -p -l 5 /dev/urandom)"
		while [[ -d "${XDG_RUNTIME_DIR}/portable/${appID}/ready-${readyDir}" ]]; do
			readyDir="$(xxd -p -l 7 /dev/urandom)"
		done
		pecho debug "Chosen readiness code ${readyDir}"
		mkdir \
			--parents \
			--mode=0700 \
			"${XDG_RUNTIME_DIR}/portable/${appID}/ready-${readyDir}"
	elif [[ $1 = "wait" ]]; then
		while [[ ! -f "${XDG_RUNTIME_DIR}/portable/${appID}/ready-${readyDir}/$2" ]]; do
			if [[ ! -d "${XDG_RUNTIME_DIR}/portable/${appID}/ready-${readyDir}" ]]; then
				break
			fi
			inotifywait \
				-e create \
				--quiet \
				"${XDG_RUNTIME_DIR}/portable/${appID}/ready-${readyDir}" 1>/dev/null
		done
		if grep -q abort "${XDG_RUNTIME_DIR}/portable/${appID}/ready-${readyDir}/$2"; then
			pecho crit "Component $2 failed! Bailing out..."
			stopApp force &
			exit 2
		elif [[ ! -d "${XDG_RUNTIME_DIR}/portable/${appID}/ready-${readyDir}" ]]; then
			pecho crit "Readiness notify failed"
		else
			pecho debug "Component $2 ready, status validated"
		fi
	fi
}

function sanityCheck() {
	mountCheck
	configCheck
	bindCheck
	readyNotify set sanityCheck
}

function bindCheck() {
	if [[ -z "${bwBindPar}" ]]; then
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
			if [[ $? -eq 1 ]]; then
				unset bwBindPar
			fi
		else
			/usr/bin/zenity \
				--title "${friendlyName}" \
				--icon=folder-open-symbolic \
				--question \
				--text="Expose ${bwBindPar}, containing ${fileCnt} ${trailingF}, ${dirCnt} ${trailingD}?"
			if [[ $? -eq 1 ]]; then
				unset bwBindPar
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
		readyNotify set-fail sanityCheck
	fi
}

function mountCheck() {
	local mounts="$(systemd-run --user -P -- findmnt -R)"
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
	mainPid=$
	for value in appID friendlyName stateDirectory launchTarget; do
		confEmpty ${value}
	done
	for value in allowInhibit bindInputDevices bindCameras bindPipewire gameMode dbusWake bindNetwork pwCam useZink qt5Compat; do
		confBool "${value}"
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
	export XDG_DOCUMENTS_DIR="$(xdg-user-dir DOCUMENTS)"
}

function manageDirs() {
	createWrapIfNotExist "${XDG_DATA_HOME}/${stateDirectory}"
	rm -r "${XDG_DATA_HOME}/${stateDirectory}/Shared"
	mkdir \
		--parents \
		--mode=0700 \
		"${XDG_DATA_HOME}/${stateDirectory}/Shared" &
	ln -sfr \
		"${XDG_DATA_HOME}/${stateDirectory}/Shared" \
		"${XDG_DATA_HOME}/${stateDirectory}/共享文件" &
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
		pecho warn "Security Context is not available due to missing dependencies"
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
	if [[ "${pwCam}" = "true" ]]; then
		pecho debug "Enabling pw-v4l2 preload..."
		addEnv "LD_PRELOAD=${LD_PRELOAD} $(ls /usr/lib/pipewire-* -d | head -n 1)/v4l2/libpw-v4l2.so" &
	else
		addEnv "LD_PRELOAD=${LD_PRELOAD}" &
	fi
	if [[ "${qt5Compat}" = "false" ]]; then
		pecho debug "Skipping Qt 5 compatibility workarounds" &
	else
		pecho debug "Enabling Qt 5 compatibility workarounds" &
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
	echo "source /run/portable-generated.env" > "${XDG_DATA_HOME}/${stateDirectory}/.bashrc"
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
	readyNotify wait im
	readyNotify wait setXdgEnv
	readyNotify wait setConfEnv
	readyNotify wait setStaticEnv
	readyNotify wait genNewEnv
	readyNotify set importEnv
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
	desktopWorkaround &
	mkdir \
		--parents \
		--mode=0700 \
		"${XDG_DATA_HOME}/${stateDirectory}/.config" &
	if [[ -z "${bwBindPar}" || ! -e "${bwBindPar}" ]]; then
		unset bwBindPar
	else
		pecho warn "bwBindPar is ${bwBindPar}"
	fi
	if [[ -d /proc/driver ]]; then
		procDriverBind="--tmpfs /proc/driver"
	else
		unset procDriverBind
	fi
	echo "false" > "${XDG_RUNTIME_DIR}/portable/${appID}/startSignal"
	sync "${XDG_RUNTIME_DIR}/portable/${appID}/startSignal"
	getDevArgs pipewireBinding
	getDevArgs sdNetArg
	getDevArgs bwInputArg
	getDevArgs bwCamPar
	getDevArgs bwSwitchableGraphicsArg
	termExec
	readyNotify wait importEnv
	readyNotify wait generateFlatpakInfo
	readyNotify wait deviceBinding
	terminateOnRequest &
	systemd-run \
	--remain-after-exit \
	--user \
	${sdOption} \
	--service-type=notify \
	--wait \
	-u "${unitName}" \
	-p BindsTo="${proxyName}.service" \
	-p Description="Portable Sandbox for ${appID}" \
	-p Documentation="https://github.com/Kraftland/portable" \
	-p Slice="portable-${friendlyName}.slice" \
	-p ExitType=cgroup \
	-p NotifyAccess=all \
	-p TimeoutStartSec=infinity \
	-p OOMPolicy=stop \
	-p SecureBits=noroot-locked\
	-p KillMode=control-group \
	-p LimitCORE=0 \
	-p CPUAccounting=yes \
	-p StartupCPUWeight=idle \
	-p StartupIOWeight=1 \
	-p MemoryHigh=90% \
	-p ManagedOOMSwap=kill \
	-p ManagedOOMMemoryPressure=kill \
	-p IPAccounting=yes \
	-p MemoryPressureWatch=yes \
	-p EnvironmentFile="${XDG_RUNTIME_DIR}/portable/${appID}/portable-generated.env" \
	-p SystemCallFilter=~@clock \
	-p SystemCallFilter=~@cpu-emulation \
	-p SystemCallFilter=~@debug \
	-p SystemCallFilter=~@module \
	-p SystemCallFilter=~@obsolete \
	-p SystemCallFilter=~@resources \
	-p SystemCallFilter=~@raw-io \
	-p SystemCallFilter=~@reboot \
	-p SystemCallFilter=~@swap \
	-p SystemCallErrorNumber=EPERM \
	-p SyslogIdentifier="portable-${appID}" \
	-p SystemCallLog='@privileged @debug @cpu-emulation @obsolete io_uring_enter io_uring_register io_uring_setup' \
	-p SystemCallLog='~@sandbox' \
	-p PrivateIPC=yes \
	-p ProtectClock=yes \
	-p CapabilityBoundingSet= \
	-p RestrictSUIDSGID=yes \
	-p LockPersonality=yes \
	-p RestrictRealtime=yes \
	-p ProtectProc=invisible \
	-p ProcSubset=pid \
	-p ProtectHome=no \
	-p PrivateUsers=yes \
	-p UMask=077 \
	-p DevicePolicy=strict \
	-p NoNewPrivileges=yes \
	-p ProtectControlGroups=yes \
	-p PrivateMounts=yes \
	-p KeyringMode=private \
	-p TimeoutStopSec=20s \
	-p Environment=instanceId="${instanceId}" \
	-p Environment=busDir="${busDir}" \
	-p "${sdNetArg}" \
	-p Environment=HOME="${XDG_DATA_HOME}/${stateDirectory}" \
	-p WorkingDirectory="${XDG_DATA_HOME}/${stateDirectory}" \
	-p Environment=WAYLAND_DISPLAY="${wayDisplayBind}" \
	-p Environment=XAUTHORITY="${XAUTHORITY}" \
	-p UnsetEnvironment=GNOME_SETUP_DISPLAY \
	-- \
	bwrap --new-session \
		--unshare-cgroup-try \
		--unshare-ipc \
		--unshare-uts \
		--unshare-pid \
		--unshare-user \
		--ro-bind "${XDG_RUNTIME_DIR}/portable/${appID}/flatpak-info" \
			/.flatpak-info \
		--dir /tmp \
		--bind-try /tmp/.X11-unix /tmp/.X11-unix \
		--bind-try /tmp/.XIM-unix /tmp/.XIM-unix \
		--dev /dev \
		--mqueue /dev/mqueue \
		--dev-bind /dev/dri /dev/dri \
		${bwSwitchableGraphicsArg} \
		--dev-bind-try /dev/udmabuf /dev/udmabuf \
		--tmpfs /sys \
		--ro-bind /sys/module/ /sys/module/ \
		--ro-bind /sys/dev/char /sys/dev/char \
		--ro-bind /sys/devices /sys/devices \
		--tmpfs /sys/devices/virtual/dmi \
		--dir /sys/class \
		--symlink /dev/dri/ /sys/class/drm \
		${bwInputArg} \
		--bind /usr /usr \
		--overlay-src /usr/bin \
		--overlay-src /usr/lib/portable/overlay-usr \
		--tmp-overlay /usr/bin \
		--ro-bind /usr/lib/portable/helper \
			/usr/lib/flatpak-xdg-utils/flatpak-spawn \
		--proc /proc \
		--ro-bind-try /dev/null /proc/uptime \
		--ro-bind-try /dev/null /proc/modules \
		--ro-bind-try /dev/null /proc/cmdline \
		--ro-bind-try /dev/null /proc/diskstats \
		--ro-bind-try /dev/null /proc/devices \
		--ro-bind-try /dev/null /proc/config.gz \
		--ro-bind-try /dev/null /proc/version \
		--ro-bind-try /dev/null /proc/mounts \
		--ro-bind-try /dev/null /proc/loadavg \
		--ro-bind-try /dev/null /proc/filesystems \
		--ro-bind-try /dev/null /proc/partitions \
		--ro-bind-try /dev/null /proc/stat \
		--ro-bind-try /dev/null /proc/vmstat \
		${procDriverBind} \
		--tmpfs /proc/1 \
		--tmpfs /usr/share/applications \
		--ro-bind /etc /etc \
		--tmpfs /etc/kernel \
		--symlink /usr/lib /lib \
		--symlink /usr/lib /lib64 \
		--symlink /usr/bin /bin \
		--symlink /usr/bin /sbin \
		--ro-bind-try /opt /opt \
		--bind "${XDG_RUNTIME_DIR}/portable/${appID}" \
			/run \
		--ro-bind "${xAuthBind}" \
			"/run/.Xauthority" \
		--bind "${XDG_RUNTIME_DIR}/portable/${appID}" \
			"${XDG_RUNTIME_DIR}/portable/${appID}" \
		--bind "${busDir}" "${XDG_RUNTIME_DIR}" \
		--bind "${busDirAy}" "${XDG_RUNTIME_DIR}/at-spi" \
		--dir /run/host \
		--ro-bind "${XDG_RUNTIME_DIR}/portable/${appID}/flatpak-info" \
			"${XDG_RUNTIME_DIR}/.flatpak-info" \
		--ro-bind-try "${XDG_RUNTIME_DIR}/pulse" \
			"${XDG_RUNTIME_DIR}/pulse" \
		${pipewireBinding} \
		--bind "${XDG_RUNTIME_DIR}/doc/by-app/${appID}" \
			"${XDG_RUNTIME_DIR}/doc" \
		--ro-bind /dev/null \
			"${XDG_RUNTIME_DIR}/.flatpak/${instanceId}-private/run-environ" \
		--ro-bind "${XDG_RUNTIME_DIR}/.flatpak/${instanceId}" \
			"${XDG_RUNTIME_DIR}/.flatpak/${instanceId}" \
		--ro-bind "${XDG_RUNTIME_DIR}/.flatpak/${instanceId}" \
			"${XDG_RUNTIME_DIR}/flatpak-runtime-directory" \
		--bind "${XDG_DATA_HOME}/${stateDirectory}" "${HOME}" \
		--bind "${XDG_DATA_HOME}/${stateDirectory}" \
			"${XDG_DATA_HOME}/${stateDirectory}" \
		--ro-bind-try "${XDG_DATA_HOME}/icons" \
			"${XDG_DATA_HOME}/icons" \
		--ro-bind-try "${XDG_DATA_HOME}/icons" \
			"$(echo "${XDG_DATA_HOME}" | pathTranslation)/icons" \
		--ro-bind "${XDG_RUNTIME_DIR}/portable/${appID}/flatpak-info" \
			"${XDG_DATA_HOME}/${stateDirectory}/.flatpak-info" \
		--ro-bind-try "${wayDisplayBind}" \
				"${wayDisplayBind}" \
		--ro-bind-try "${XDG_CONFIG_HOME}/fontconfig" \
			"${XDG_CONFIG_HOME}/fontconfig" \
		--ro-bind-try "${XDG_CONFIG_HOME}/fontconfig" \
			"$(echo "${XDG_CONFIG_HOME}" | pathTranslation)/fontconfig" \
		--ro-bind-try "${XDG_CONFIG_HOME}/gtk-3.0/gtk.css" \
			"$(echo "${XDG_CONFIG_HOME}" | pathTranslation)/gtk-3.0/gtk.css" \
		--ro-bind-try "${XDG_CONFIG_HOME}/gtk-4.0/gtk.css" \
			"$(echo "${XDG_CONFIG_HOME}" | pathTranslation)/gtk-4.0/gtk.css" \
		--ro-bind-try "${XDG_DATA_HOME}/fonts" \
			"${XDG_DATA_HOME}/fonts" \
		--ro-bind-try "${XDG_DATA_HOME}/fonts" \
			"$(echo "${XDG_DATA_HOME}" | pathTranslation)/fonts" \
		--ro-bind-try "/run/systemd/resolve/stub-resolv.conf" \
			"/run/systemd/resolve/stub-resolv.conf" \
		--size 1 \
		--perms 0000 \
		--tmpfs "${HOME}/options" \
		--perms 0000 \
		--size 1 \
		--tmpfs "${XDG_DATA_HOME}/${stateDirectory}/options" \
		--perms 0000 \
		--size 1 \
		--tmpfs "${HOME}/.var" \
		--perms 0000 \
		--size 1 \
		--tmpfs "${XDG_DATA_HOME}/${stateDirectory}/.var" \
		--bind "${XDG_DATA_HOME}/${stateDirectory}" \
			"${XDG_DATA_HOME}/${stateDirectory}/.var/app/${appID}" \
		--bind "${XDG_DATA_HOME}/${stateDirectory}" \
			"${HOME}/.var/app/${appID}" \
		--tmpfs "${HOME}/.var/app/${appID}/options" \
		--tmpfs "${XDG_DATA_HOME}/${stateDirectory}/.var/app/${appID}/options" \
		--bind "${XDG_RUNTIME_DIR}/systemd/notify" \
			"${XDG_RUNTIME_DIR}/systemd/notify" \
		${bwCamPar} \
		${bwBindPar:+--dev-bind "${bwBindPar}" "${bwBindPar}"} \
		-- \
			/usr/lib/portable/helper ${launchTarget} "${targetArgs[@]}"

		stopApp
}

function terminateOnRequest() {
	pecho debug "Established termination watches"
	while true; do
		if [[ ! -r "${XDG_RUNTIME_DIR}/portable/${appID}/startSignal" ]]; then
			pecho warn "startSignal is missing! Stopping application"
			stopApp
		fi
		inotifywait \
			--quiet \
			"${XDG_RUNTIME_DIR}/portable/${appID}/startSignal" 1>/dev/null
		if [[ "$(cat "${XDG_RUNTIME_DIR}/portable/${appID}/startSignal")" =~ "terminate-now" ]]; then
			stopApp force
			exit 0
			break
		fi
	done
}

function execAppExistDirect() {
	echo "${launchTarget}" "${targetArgs[@]}" > "${XDG_RUNTIME_DIR}/portable/${appID}/startSignal"
}

function termExec() {
	trap "stopApp force" SIGTERM SIGINT SIGHUP SIGQUIT SIGKILL
}

function execAppExist() {
	genXAuth
	importEnv
	unitName="${unitName}-subprocess-$(uuidgen)"
	instanceId=$(grep instance-id "${XDG_RUNTIME_DIR}/portable/${appID}/flatpak-info" | cut -c '13-')
	execApp
	exit $?
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

function bindNvDevIfExist(){
	if ls /dev/nvidia* &> /dev/null; then
		pecho debug "Binding NVIDIA GPUs in Game Mode"
		for _card in /dev/nvidia*; do
			if [[ -e "${_card}" ]]; then
				bwSwitchableGraphicsArg="${bwSwitchableGraphicsArg} --dev-bind ${_card} ${_card}"
			fi
		done
		export nvExist=1
	fi
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
	addEnv "DRI_PRIME=1"
	if [[ "${nvExist}" = 1 ]]; then
		pecho debug "Specifying environment variables for dGPU utilization: NVIDIA"
		addEnv "__NV_PRIME_RENDER_OFFLOAD=1"
		addEnv "__VK_LAYER_NV_optimus=NVIDIA_only"
		addEnv "__GLX_VENDOR_LIBRARY_NAME=nvidia"
		addEnv "VK_LOADER_DRIVERS_SELECT=nvidia_icd.json"
	else
		pecho debug "Specifying environment variables for dGPU utilization: Mesa"
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

function hybridBind() {
	#local bwSwitchableGraphicsArg
	bwSwitchableGraphicsArg='--setenv portableDiscrete 1'
	if [[ "$(ls -l /sys/class/drm | grep render | wc -l)" -le 1 ]]; then
		pecho debug "Single or no GPU, binding all devices"
		bindNvDevIfExist
	elif [[ "${gameMode}" = "true" ]]; then
		pecho debug "Game Mode enabled on hybrid graphics"
		bindNvDevIfExist
		setNvOffloadEnv
	else
		bwSwitchableGraphicsArg="--tmpfs /dev/dri"
		local activeCardSum=0
		activeCards="placeholder"
		for vCards in $(find /sys/class/drm -name 'card*' -not -name '*-*'); do
			pecho debug "Working on "${vCards}""
			for file in $(find -L "${vCards}" -maxdepth 2 -name status 2>/dev/null); do
				pecho debug "Inspecting ${file}"
				if [[ "$(cat "${file}")" =~ "disconnected" ]]; then
					continue
				else
					pecho debug "Active GPU"
					activeCardSum=$(expr "${activeCardSum}" + 1)
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
			pecho debug "${activeCardSum} card active, identified as ${activeCards}"
			addEnv "VK_LOADER_DRIVERS_DISABLE='nvidia_icd.json'"
			cardToRender "${activeCards}"
			bwSwitchableGraphicsArg="${bwSwitchableGraphicsArg} --dev-bind "/dev/dri/${renderIndex}" "/dev/dri/${renderIndex}""
		else
			pecho debug "${activeCardSum} cards active"
			for vCards in ${activeCards}; do
			# TODO: What happens to non NVIDIA, more than 1 active GPU hybrid configuration?
				if grep -q '0x10de' "/sys/class/drm/${vCards}/device/vendor"; then
					addEnv "VK_LOADER_DRIVERS_DISABLE='nvidia_icd.json'"
					continue
				else
					cardToRender "${vCards}"
					pecho debug "Binding ${renderIndex}"
					bwSwitchableGraphicsArg="${bwSwitchableGraphicsArg} --dev-bind "/dev/dri/${renderIndex}" "/dev/dri/${renderIndex}""
					addEnv 'DRI_PRIME=0'
				fi
			done
		fi
	fi
	pecho debug "Generated GPU bind parameter: ${bwSwitchableGraphicsArg}"
	passDevArgs bwSwitchableGraphicsArg "${bwSwitchableGraphicsArg}"
	readyNotify set hybridBind
}

function cameraBind() {
	bwCamPar=""
	if [[ "${bindCameras}" = "true" ]]; then
		pecho debug "Detecting Camera..."
		for _camera in /dev/video*; do
			if [[ -e "${_camera}" ]]; then
				bwCamPar="${bwCamPar} --dev-bind ${_camera} ${_camera}"
			fi
		done
	fi
	pecho debug "Generated Camera bind parameter: ${bwCamPar}"
	passDevArgs bwCamPar "${bwCamPar}"
	readyNotify set cameraBind
}

function inputBind() {
	if [[ "${bindInputDevices}" = "true" ]]; then
		bwInputArg="--dev-bind-try /sys/class/leds /sys/class/leds --dev-bind-try /sys/class/input /sys/class/input --dev-bind-try /sys/class/hidraw /sys/class/hidraw --dev-bind-try /dev/input /dev/input --dev-bind-try /dev/uinput /dev/uinput"
		for _device in /dev/hidraw*; do
			if [[ -e "${_device}" ]]; then
				bwInputArg="${bwInputArg} --dev-bind ${_device} ${_device}"
			fi
		done
		pecho warn "Detected input preference as expose, setting arg: ${bwInputArg}"
	else
		bwInputArg=""
		pecho debug "Not exposing input devices"
	fi
	passDevArgs bwInputArg "${bwInputArg}"
	readyNotify set inputBind
}

function miscBind() {
	if [[ "${bindNetwork}" = "false" ]]; then
		pecho info "Network access disabled via config"
		sdNetArg="PrivateNetwork=yes"
	else
		sdNetArg="PrivateNetwork=no"
		pecho debug "Network access allowed"
	fi
	passDevArgs sdNetArg "${sdNetArg}"
	if [[ "${bindPipewire}" = "true" ]]; then
		pipewireBinding="--ro-bind-try ${XDG_RUNTIME_DIR}/pipewire-0 ${XDG_RUNTIME_DIR}/pipewire-0"
	fi
	passDevArgs pipewireBinding "${pipewireBinding}"
	readyNotify set miscBind
}

function deviceBinding() {
	mkdir -p "${XDG_RUNTIME_DIR}/portable/${appID}/devstore"
	hybridBind &
	inputBind &
	cameraBind &
	miscBind &
	readyNotify wait cameraBind
	readyNotify wait miscBind
	readyNotify wait inputBind
	readyNotify wait hybridBind
	readyNotify set deviceBinding
}

function appANR() {
	if [[ "${LANG}" =~ "zh_CN" ]]; then
		zenity --title "程序未响应" --icon=utilities-system-monitor-symbolic --default-cancel --question --text="是否结束正在运行的进程?"
	else
		zenity --title "Application is not responding" --icon=utilities-system-monitor-symbolic --default-cancel --question --text="Do you wish to terminate the running session?"
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
	if [[ "$*" =~ "--actions" ]] && [[ "$*" =~ "debug-shell" ]]; then
		export launchTarget="/usr/bin/bash"
		execAppExist
	else
		execAppExistDirect
		exit "$?"
	fi
	appANR
	if [[ $? -eq 0 ]]; then
		stopApp force
	else
		pecho crit "User denied session termination"
		exit "$?"
	fi
}

function genInstanceID() {
	instanceId=$(head -c 4 /dev/urandom | xxd -p | tr -d '\n' | awk '{print strtonum("0x"$1)}')
	while [[ -d "${XDG_RUNTIME_DIR}/.flatpak/${instanceId}" ]]; do
		pecho debug "Instance ID collision detected!"
		instanceId=$(head -c 4 /dev/urandom | xxd -p | tr -d '\n' | awk '{print strtonum("0x"$1)}')
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
		extraDbusArgs="$@"
	else
		extraDbusArgs="${extraDbusArgs} $@"
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
		"${proxyName}*".service \
		"${proxyName}*"-a11y.service \
		"${friendlyName}*"-wayland-proxy.service \
		"${friendlyName}-subprocess*".service
	systemctl --user clean "${friendlyName}*" \
		"${friendlyName}-subprocess*".service \
		"${proxyName}*".service \
		"${proxyName}*"-a11y.service \
		"${friendlyName}*"-wayland-proxy.service
	readyNotify set cleanDUnits
}

function dbusArg() {
	mkdir -p "${XDG_RUNTIME_DIR}/portable/${appID}/busstore"
	if [[ "${PORTABLE_LOGGING}" = "debug" ]]; then
		proxyArg="--log"
	fi
	if [[ "${XDG_CURRENT_DESKTOP}" = "GNOME" ]]; then
		local featureSet="GlobalShortcuts"
		pecho info "Enabling GNOME exclusive features: ${featureSet}"
		addDbusArg \
			"--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.GlobalShortcuts --call=org.freedesktop.portal.Desktop=org.freedesktop.portal.GlobalShortcuts.*"
	fi
	mkdir \
		--parents \
		--mode=0700 \
		"${XDG_RUNTIME_DIR}/doc/by-app/${appID}"
	if [[ -n "${mprisName}" ]]; then
		local mprisBus="org.mpris.MediaPlayer2.${mprisName}"
		addDbusArg \
			"--own=${mprisBus} --own=${mprisBus}.*"
	else
		local mprisBus="org.mpris.MediaPlayer2.${appID##*.}"
		addDbusArg \
			"--own=${mprisBus} --own=${mprisBus}.*"
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
	getBusArgs extraDbusArgs
	getBusArgs proxyArg
	systemd-run \
		--user \
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
			--own=com.belmoussaoui.ashpd.demo \
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
			-p Slice="portable-${friendlyName}.slice" \
			-u "${friendlyName}"-wayland-proxy \
			-p BindsTo="${proxyName}.service" \
			-p Environment=WAYLAND_DISPLAY="${WAYLAND_DISPLAY}" \
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
		"${launchTarget}"
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
	sdOption="-P"
	if systemctl --user --quiet is-failed "${unitName}.service"; then
		pecho warn "${appID} failed last time"
		systemctl --user reset-failed "${unitName}.service" &
	fi
	deviceBinding &
	if systemctl --user --quiet is-active "${unitName}.service"; then
		warnMulRunning "$@"
	fi
	sanityCheck &
	if [[ "$*" =~ "--actions" && "$*" =~ "debug-shell" ]]; then
		launchTarget="/usr/bin/bash"
	fi
	if [[ ${trashAppUnsafe} -eq 1 ]]; then
		pecho warn "Launching ${appID} (unsafe)..."
		execAppUnsafe
	else
		dbusProxy
		pecho info "Launching ${appID}..."
		execApp
	fi
}

function stopApp() {
	if [[ "$*" =~ "external" ]]; then
		systemctl \
			--user stop \
			"${friendlyName}.service"
		exit 0
	elif [[ "$*" =~ "force" ]]; then
		pecho info "Force stop is called, killing service" &
		systemctl \
			--user kill \
			-sSIGKILL \
			"${friendlyName}.service"
	else
		sleep 1s
		local sdOut
		sdOut=$(systemctl --user list-units --state active --no-pager "${friendlyName}"*)
		if [[ "${sdOut}" =~ "${friendlyName}.service" ]] && [[ "${sdOut}" =~ "subprocess" ]]; then
			pecho crit "Not stopping the slice because one or more instance are still running"
			exit 1
		#elif [[ $(systemctl --user list-units --state active --no-pager "${friendlyName}"* | grep subprocess | wc -l) -ge 2 ]]; then
			#pecho crit "Not stopping the slice because two or more subprocesses are still running"
			#exit 1
		fi
	fi
	if [[ "$(systemctl --user list-units --state active --no-pager "${friendlyName}"* | grep service | wc -l)" -eq 0 ]]; then
		pecho debug "Application already stopped!"
		alreadyStopped=1
	else
		pecho info "Stopping application..." &
		systemctl \
			--user stop \
			"portable-${friendlyName}.slice"
	fi
	source "${XDG_RUNTIME_DIR}/portable/${appID}/control" 2>/dev/null
	if [[ $? -eq 0 ]]; then
		pecho debug "Sourced configuration..." &
	else
		if [[ ${alreadyStopped} -eq 1 ]]; then
			pecho debug "Not showing warning because application is already stopped" &
		else
			pecho warn "Control file missing! Did you upgrade portable?" &
		fi
	fi
	if [[ -n "${appID}" ]] && [[ -n "${instanceId}" ]] && [[ -n "${busDir}" ]]; then
		pecho debug "Cleaning leftovers..." &
		rm -rf "${XDG_RUNTIME_DIR}/.flatpak/${instanceId}"
		rm -rf "${XDG_RUNTIME_DIR}/.flatpak/${appID}"
		rm -rf "${busDir}"
		rm -rf "${XDG_RUNTIME_DIR}/portable/${appID}"
		if [[ -n "${busDirAy}" ]]; then
			rm -rf "${busDirAy}"
		fi
		rm -rf \
			"${XDG_DATA_HOME}/applications/${appID}.desktop"\
			2>/dev/null
		exit 0
	else
		pecho warn "Clean shutdown not possible due to missing information."
		exit 1
	fi
}

function resetDocuments() {
	flatpak permission-reset "${appID}"
}

function cmdlineDispatcher() {
	local cmdlineArgs=("$@")
	local indexSep=-1

	for i in "${!cmdlineArgs[@]}"; do
		if [[ "${cmdlineArgs[${i}]}" = "--" ]]; then
			indexSep=${i}
			break # break the loop at separator
		fi
		continue
	done

	local appArgs=()
	if [[ ${indexSep} -ge 0 ]]; then
		appArgs=("${cmdlineArgs[@]:$((indexSep + 1))}")
	fi
	targetArgs=("${appArgs[@]}")
	pecho info "Application argument interpreted as: ${targetArgs[*]}"

	if [[ "$*" =~ "f5aaebc6-0014-4d30-beba-72bce57e0650" ]] && [[ "$*" =~ "--actions" ]]; then
		rm -f "${XDG_DATA_HOME}/${stateDirectory}/options/sandbox"
		questionFirstLaunch
	fi
	if [[ "$*" =~ "--actions" ]] && [[ "$*" =~ "opendir" ]]; then
		/usr/lib/flatpak-xdg-utils/xdg-open "${XDG_DATA_HOME}/${stateDirectory}"
		exit "$?"
	fi
	if [[ "$*" =~ "--actions" ]] && [[ "$*" =~ "share-files" ]]; then
		shareFile
	fi
	if [[ "$*" =~ "--actions" ]] && [[ "$*" =~ "reset-documents" ]]; then
		resetDocuments
	fi
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
