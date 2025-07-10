#!/bin/bash

function pecho() {
	if [[ "$1" = "debug" ]] && [[ "${PORTABLE_LOGGING}" = "debug" ]]; then
		echo "[Debug] $2" &
	elif [[ "$1" = "info" ]] && [[ "${PORTABLE_LOGGING}" = "info" || "${PORTABLE_LOGGING}" = "debug" ]]; then
		echo "[Info] $2" &
	elif [[ "$1" = "warn" ]]; then
		echo "[Warn] $2" &
	elif [[ "$1" = "crit" ]]; then
		echo "[Critical] $2" &
	fi
}

if [[ "${_portalConfig}" && "${_portableConfig}" ]]; then
	pecho crit "No portable config specified!"
	exit 1
fi

if [[ "${_portalConfig}" ]]; then
	export _portableConfig="${_portalConfig}"
	pecho warn "Using legacy configuration variable!"
fi

if [[ -f "${_portableConfig}" ]]; then
	pecho info "Configuration specified as absolute path: ${_portableConfig}"
	source "${_portableConfig}"
else
	if [[ -f "/usr/lib/portable/info/${_portableConfig}/config" ]]; then
		pecho info \
			"Configuration specified as global name /usr/lib/portable/info/${_portableConfig}/config"
		source "/usr/lib/portable/info/${_portableConfig}/config"
		export _portableConfig="/usr/lib/portable/info/${_portableConfig}/config"
	elif [[ -f "$(pwd)/${_portableConfig}" ]]; then
		pecho info \
			"Configuration specified as relative path ${_portableConfig}"
		source "$(pwd)/${_portableConfig}"
		export _portableConfig="$(pwd)/${_portableConfig}"
	else
		pecho crit "Specified config cannot be found!"
		exit 1
	fi
fi

busName="${appID}"
busDir="${XDG_RUNTIME_DIR}/app/${busName}"
busDirAy="${XDG_RUNTIME_DIR}/app/${busName}-a11y"
unitName="${friendlyName}"
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
	export XDG_DOCUMENTS_DIR="$(xdg-user-dir DOCUMENTS)"
}

function manageDirs() {
	createWrapIfNotExist "${XDG_DATA_HOME}/${stateDirectory}"
	rm -r "${XDG_DATA_HOME}/${stateDirectory}/Shared"
	mkdir -p "${XDG_DATA_HOME}/${stateDirectory}/Shared" &
	ln -sfr \
		"${XDG_DATA_HOME}/${stateDirectory}/Shared" \
		"${XDG_DATA_HOME}/${stateDirectory}/共享文件" &
}

function genXAuth() {
	rm "${XDG_DATA_HOME}/${stateDirectory}/.Xauthority" 2>/dev/null
	rm -rf "${XDG_DATA_HOME}/${stateDirectory}/.XAuthority" 2>/dev/null
	ln -srf \
		"${XDG_DATA_HOME}/${stateDirectory}/.Xauthority" \
		"${XDG_DATA_HOME}/${stateDirectory}/.XAuthority" &
	if [[ "${waylandOnly}" = "true" ]]; then
		touch "${XDG_DATA_HOME}/${stateDirectory}/.Xauthority"
		return "$?"
	elif [[ "${waylandOnly}" = "adaptive" && "${XDG_SESSION_TYPE}" = "wayland" ]]; then
		touch "${XDG_DATA_HOME}/${stateDirectory}/.Xauthority"
		return "$?"
	fi
	pecho debug "Processing X Server security restriction..."
	touch "${XDG_DATA_HOME}/${stateDirectory}/.Xauthority"
	pecho debug "Detecting display as ${DISPLAY}"
	if [[ $(xauth list "${DISPLAY}" | head -n 1) =~ "$(hostname)/unix: " ]]; then
		pecho warn "Adding new display..."
		authHash="$(xxd -p -l 16 /dev/urandom)"
		xauth add "${DISPLAY}" . "${authHash}"
		xauth \
			add \
			"${DISPLAY}" \
			. \
			"${authHash}"
	else
		xauth -f \
			"${XDG_DATA_HOME}/${stateDirectory}/.Xauthority" \
			add $(xauth list ${DISPLAY} | head -n 1)
	fi
	if [[ ! -f "${HOME}/.Xauthority" && -z "${XAUTHORITY}" ]]; then
		pecho warn "Could not determine Xauthority file path"
		xhost +localhost
	fi
	if xauth -f \
			"${XDG_DATA_HOME}/${stateDirectory}/.Xauthority" list >/dev/null; then
		return 0
	else
		pecho warn "Turning off X access control for localhost"
		xauth +localhost
	fi
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
		mkdir -p "$@"
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

function importEnv() {
	inputMethod
	genXAuth
	cat "${_portableConfig}" > "${XDG_RUNTIME_DIR}/portable/${appID}/portable-generated.env"
	ln -srf \
		"${XDG_RUNTIME_DIR}/portable/${appID}/portable-generated.env" \
		"${XDG_DATA_HOME}/${stateDirectory}/portable-generated.env" &
	printf "\n\n" >> "${XDG_RUNTIME_DIR}/portable/${appID}/portable-generated.env"
	addEnv "XDG_CONFIG_HOME=$(echo "${XDG_CONFIG_HOME}" | pathTranslation)" &
	addEnv "XDG_DOCUMENTS_DIR=${XDG_DATA_HOME}/${stateDirectory}/Documents" &
	addEnv "XDG_DATA_HOME=${XDG_DATA_HOME}/${stateDirectory}/.local/share" &
	addEnv "XDG_STATE_HOME=${XDG_DATA_HOME}/${stateDirectory}/.local/state" &
	addEnv "XDG_CACHE_HOME=${XDG_DATA_HOME}/${stateDirectory}/cache" &
	addEnv "XDG_DESKTOP_DIR=${XDG_DATA_HOME}/${stateDirectory}/Desktop" &
	addEnv "XDG_DOWNLOAD_DIR=${XDG_DATA_HOME}/${stateDirectory}/Downloads" &
	addEnv "XDG_TEMPLATES_DIR=${XDG_DATA_HOME}/${stateDirectory}/Templates" &
	addEnv "XDG_PUBLICSHARE_DIR=${XDG_DATA_HOME}/${stateDirectory}/Public" &
	addEnv "XDG_MUSIC_DIR=${XDG_DATA_HOME}/${stateDirectory}/Music" &
	addEnv "XDG_PICTURES_DIR=${XDG_DATA_HOME}/${stateDirectory}/Pictures" &
	addEnv "XDG_VIDEOS_DIR=${XDG_DATA_HOME}/${stateDirectory}/Videos" &
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
		addEnv "__GLX_VENDOR_LIBRARY_NAME=mesa"
		addEnv "MESA_LOADER_DRIVER_OVERRIDE=zink"
		addEnv "GALLIUM_DRIVER=zink"
		addEnv "LIBGL_KOPPER_DRI2=1"
		addEnv "__EGL_VENDOR_LIBRARY_FILENAMES=/usr/share/glvnd/egl_vendor.d/50_mesa.json"
	fi
	addEnv "GDK_DEBUG=portals"
	addEnv "GTK_USE_PORTAL=1"
	addEnv "QT_AUTO_SCREEN_SCALE_FACTOR=1"
	addEnv "GTK_IM_MODULE=${GTK_IM_MODULE}"
	addEnv "QT_IM_MODULE=${QT_IM_MODULE}"
	addEnv "QT_ENABLE_HIGHDPI_SCALING=1"
	addEnv "PATH=/sandbox:${PATH}"
	addEnv "DISPLAY=${DISPLAY}"
	addEnv "QT_SCALE_FACTOR=${QT_SCALE_FACTOR}"
	addEnv "PS1='╰─>Portable Sandbox·${appID}·🧐⤔ '"
	printf "\n\n" >> "${XDG_RUNTIME_DIR}/portable/${appID}/portable-generated.env"
	if [[ -e "${XDG_DATA_HOME}/${stateDirectory}/portable.env" ]]; then
		pecho info "${XDG_DATA_HOME}/${stateDirectory}/portable.env exists"
	else
		touch "${XDG_DATA_HOME}/${stateDirectory}/portable.env"
	fi
	if [[ -s "${XDG_DATA_HOME}/${stateDirectory}/portable.env" ]]; then
		cat "${XDG_DATA_HOME}/${stateDirectory}/portable.env" >> "${XDG_RUNTIME_DIR}/portable/${appID}/portable-generated.env"
	else
		echo "# Envs" >> "${XDG_DATA_HOME}/${stateDirectory}/portable.env"
		echo "isPortableEnvPresent=1" >> "${XDG_DATA_HOME}/${stateDirectory}/portable.env"
	fi
	echo "source /run/portable-generated.env" > "${XDG_DATA_HOME}/${stateDirectory}/.bashrc"
}

function getChildPid() {
	local cGroup cmdlineArg
	cGroup=$(systemctl --user show "${unitName}" -p ControlGroup | cut -c '14-')
	pecho debug "Getting PID from unit ${unitName}'s control group $(systemctl --user show "${unitName}" -p ControlGroup | cut -c '14-')"
	for childPid in $(pgrep --cgroup "${cGroup}"); do
		pecho debug "Trying PID ${childPid}"
		cmdlineArg=$(tr '\000' ' ' < "/proc/${childPid}/cmdline")
		if [[ "${cmdlineArg}" =~ "/usr/lib/portable/helper" ]]; then
			if [[ $(echo "${cmdlineArg}" | cut -c '-14') =~ "bwrap" ]]; then
				pecho debug "Detected bwrap"
			else
				pecho debug "Detected helper"
				return 0
			fi
		fi
	done
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
	if [[ ! -d "${XDG_RUNTIME_DIR}/portable/${appID}" ]]; then
		mkdir -p "${XDG_RUNTIME_DIR}/portable/${appID}"
	fi
}

function execApp() {
	desktopWorkaround &
	importEnv
	deviceBinding
	mkdir -p "${XDG_DATA_HOME}/${stateDirectory}/.config"
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
	termExec
	terminateOnRequest &
	passPid &
	systemd-run \
	--user \
	${sdOption} \
	--send-sighup \
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
	-p ProtectKernelModules=yes \
	-p RestrictSUIDSGID=yes \
	-p LockPersonality=yes \
	-p RestrictRealtime=yes \
	-p ProtectSystem=full \
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
	-p BindReadOnlyPaths=/usr/bin/true:/usr/bin/lsblk \
	-p Environment=XAUTHORITY="${XDG_DATA_HOME}/${stateDirectory}/.Xauthority" \
	-p Environment=instanceId="${instanceId}" \
	-p Environment=busDir="${busDir}" \
	-p "${sdNetArg}" \
	-p Environment=HOME="${XDG_DATA_HOME}/${stateDirectory}" \
	-p WorkingDirectory="${XDG_DATA_HOME}/${stateDirectory}" \
	-p Environment=WAYLAND_DISPLAY="${wayDisplayBind}" \
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
		--dir /sandbox \
		--ro-bind /usr/bin/true \
			/sandbox/sudo \
		--ro-bind /usr/lib/portable/open \
			/sandbox/chromium \
		--ro-bind /usr/lib/portable/open \
			/sandbox/firefox \
		--ro-bind /usr/lib/portable/open \
			/sandbox/dde-file-manager \
		--ro-bind /usr/lib/portable/open \
			/sandbox/xdg-open \
		--ro-bind /usr/lib/portable/open \
			/sandbox/open \
		--ro-bind /usr/lib/portable/open \
			/sandbox/nautilus \
		--ro-bind /usr/lib/portable/open \
			/sandbox/dolphin \
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
		${procDriverBind} \
		--tmpfs /proc/1 \
		--bind-try /dev/null /proc/cpuinfo \
		--bind /usr /usr \
		--tmpfs /usr/share/applications \
		--ro-bind /etc /etc \
		--tmpfs /etc/kernel \
		--symlink /usr/lib /lib \
		--symlink /usr/lib /lib64 \
		--ro-bind-try /bin /bin \
		--ro-bind-try /sbin /sbin \
		--ro-bind-try /opt /opt \
		--bind "${XDG_RUNTIME_DIR}/portable/${appID}" \
			/run \
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
		--tmpfs "${HOME}/options" \
		${bwBindPar:+--dev-bind "${bwBindPar}" "${bwBindPar}"} \
		--tmpfs "${XDG_DATA_HOME}/${stateDirectory}/options" \
		--bind "${XDG_RUNTIME_DIR}/systemd/notify" \
			"${XDG_RUNTIME_DIR}/systemd/notify" \
		${bwCamPar} \
		-- \
			/usr/lib/portable/helper ${launchTarget} "${targetArgs[@]}"

		stopApp
}

function terminateOnRequest() {
	while true; do
		if [[ ! -r "${XDG_RUNTIME_DIR}/portable/${appID}/startSignal" ]]; then
			pecho warn "startSignal is missing! Stopping application"
			stopApp
		fi
		inotifywait \
			-e modify \
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
	trap "stopApp" SIGTERM SIGINT SIGHUP SIGQUIT
}

function execAppExist() {
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
	echo "$@" >> "${XDG_RUNTIME_DIR}/portable/${appID}/portable-generated.env"
}

function desktopWorkaround() {
	dbus-send --session \
		--dest=org.freedesktop.impl.portal.PermissionStore \
		/org/freedesktop/impl/portal/PermissionStore \
		org.freedesktop.impl.portal.PermissionStore.SetPermission \
		string:"background" boolean:true string:"background" string:"${appID}" array:string:"yes" &
}

function deviceBinding() {
	if [[ "${gameMode}" = "true" ]]; then
		bwSwitchableGraphicsArg=""
		pecho debug "Binding all GPUs in Game Mode"
		if ls /dev/nvidia* &> /dev/null; then
			pecho debug "Binding NVIDIA GPUs"
			for _card in /dev/nvidia*; do
				if [[ -e "${_card}" ]]; then
					bwSwitchableGraphicsArg="${bwSwitchableGraphicsArg} --dev-bind ${_card} ${_card}"
				fi
			done
			pecho debug "Specifying environment variables for dGPU utilization"
			addEnv "__NV_PRIME_RENDER_OFFLOAD=1"
			addEnv "__VK_LAYER_NV_optimus=NVIDIA_only"
			addEnv "__GLX_VENDOR_LIBRARY_NAME=nvidia"
			addEnv "VK_LOADER_DRIVERS_SELECT=nvidia_icd.json"
			addEnv "DRI_PRIME=1"
		else
			pecho info "No NVIDIA GPU could be found!"
			pecho info "Using mesa feature... unsetting all related environment variables"
			addEnv "VK_LOADER_DRIVERS_DISABLE="
			addEnv "DRI_PRIME=1"
		fi
	else
		pecho debug "Detecting GPU..."
		bwSwitchableGraphicsArg=""
		videoMod=$(lsmod)
		if [[ $(find /dev/dri/renderD* -maxdepth 0 -type c 2>/dev/null | wc -l) -eq 1 && "${videoMod}" =~ "nvidia" ]]; then
			pecho info "Using single NVIDIA GPU"
			addEnv "GSK_RENDERER=ngl"
			for _card in /dev/nvidia*; do
				if [[ -e "${_card}" ]]; then
					bwSwitchableGraphicsArg="${bwSwitchableGraphicsArg} --dev-bind ${_card} ${_card}"
				fi
			done
		elif [[ "${videoMod}" =~ "i915" ]] || [[ "${videoMod}" =~ "xe" ]] || [[ "${videoMod}" =~ "amdgpu" ]]; then
			if [[ "${videoMod}" =~ "nvidia" ]]; then
				pecho debug "Activating hybrid GPU detection"
				bwSwitchableGraphicsArg="--tmpfs /dev/dri"
				for device in /sys/class/drm/renderD12*; do
					if [[ -e "${device}" ]]; then
						if [[ $(cat "${device}/device/vendor") = 0x10de ]]; then
							pecho debug "Device $(basename "${device}") detected as NVIDIA GPU"
						else
							bwSwitchableGraphicsArg="${bwSwitchableGraphicsArg} --dev-bind /dev/dri/$(basename "${device}") /dev/dri/$(basename "${device}")"
							pecho debug "Device $(basename "${device}") binded"
						fi
					fi
				done
			else
				pecho debug "Not using NVIDIA GPU"
			fi
			addEnv "VK_LOADER_DRIVERS_DISABLE='nvidia_icd.json'"
		elif [[ "${videoMod}" =~ "nvidia" ]]; then
			pecho debug "Using NVIDIA GPU"
			addEnv "GSK_RENDERER=ngl"
			for _card in /dev/nvidia*; do
				if [[ -e "${_card}" ]]; then
					bwSwitchableGraphicsArg="${bwSwitchableGraphicsArg} --dev-bind ${_card} ${_card}"
				fi
			done
		fi
	fi
	pecho debug "Generated GPU bind parameter: ${bwSwitchableGraphicsArg}"
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
	if [[ "${bindNetwork}" = "false" ]]; then
		pecho info "Network access disabled via config"
		sdNetArg="PrivateNetwork=yes"
	else
		sdNetArg="PrivateNetwork=no"
		pecho debug "Network access allowed"
	fi
	if [[ "${bindPipewire}" = "true" ]]; then
		pipewireBinding="--ro-bind-try ${XDG_RUNTIME_DIR}/pipewire-0 ${XDG_RUNTIME_DIR}/pipewire-0"
	fi
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

function generateFlatpakInfo() {
	pecho debug "Installing flatpak-info..."
	install /usr/lib/portable/flatpak-info \
		"${XDG_RUNTIME_DIR}/portable/${appID}/flatpak-info"
	pecho debug "Generating flatpak-info..."
	instanceId=$(head -c 4 /dev/urandom | xxd -p | tr -d '\n' | awk '{print strtonum("0x"$1)}')
	sed -i "s|placeHolderAppName|${appID}|g" \
		"${XDG_RUNTIME_DIR}/portable/${appID}/flatpak-info"
	sed -i "s|placeholderInstanceId|${instanceId}|g" \
		"${XDG_RUNTIME_DIR}/portable/${appID}/flatpak-info"
	sed -i "s|placeholderPath|${XDG_DATA_HOME}/${stateDirectory}|g" \
		"${XDG_RUNTIME_DIR}/portable/${appID}/flatpak-info"

	mkdir -p "${XDG_RUNTIME_DIR}/.flatpak/${instanceId}"
	install /usr/lib/portable/bwrapinfo.json \
		"${XDG_RUNTIME_DIR}/.flatpak/${instanceId}/bwrapinfo.json"
	install "${XDG_RUNTIME_DIR}/portable/${appID}/flatpak-info" \
		"${XDG_RUNTIME_DIR}/.flatpak/${instanceId}/info"
	pecho debug "Successfully installed bwrapinfo @${XDG_RUNTIME_DIR}/.flatpak/${instanceId}/bwrapinfo.json"
	mkdir -p "${XDG_RUNTIME_DIR}/.flatpak/${appID}/xdg-run"
	mkdir -p "${XDG_RUNTIME_DIR}/.flatpak/${appID}/tmp"
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
}

function resetUnit() {
	if [[ $(systemctl --user is-failed "${1}".service) = "failed" ]]; then
		pecho warn "${1} failed last time"
		systemctl --user reset-failed "${1}".service
	fi
}

function dbusProxy() {
	defineRunPath
	generateFlatpakInfo
	waylandDisplay
	systemctl --user clean "${friendlyName}*" &
	systemctl --user clean "${proxyName}*".service &
	systemctl --user clean "${proxyName}*"-a11y.service &
	systemctl --user clean "${friendlyName}*"-wayland-proxy.service &
	systemctl --user clean "${friendlyName}-subprocess*".service &
	mkdir -p "${busDir}"
	mkdir -p "${busDirAy}"
	pecho info "Starting D-Bus Proxy @ ${busDir}..."
	if [[ "${PORTABLE_LOGGING}" = "debug" ]]; then
		proxyArg="--log"
	fi
	if [[ "${XDG_CURRENT_DESKTOP}" = "GNOME" ]]; then
		local featureSet="GlobalShortcuts ScreenShot"
		pecho info "Enabling GNOME exclusive features: ${featureSet}"
		extraDbusArgs="--call=org.freedesktop.portal.Desktop=org.freedesktop.portal.GlobalShortcuts --call=org.freedesktop.portal.Desktop=org.freedesktop.portal.GlobalShortcuts.*"
	else
		pecho info "Disabling GNOME exclusive features"
		extraDbusArgs=""
	fi
	mkdir -p "${XDG_RUNTIME_DIR}/doc/by-app/${appID}"
	# Unused Policy: --call=org.freedesktop.portal.Flatpak=*@/org/freedesktop/portal/Flatpak
	# Unused: --call=org.freedesktop.portal.Flatpak=org.freedesktop.DBus.Peer.Ping
	systemd-run \
		--user \
		-p Slice="portable-${friendlyName}.slice" \
		-u "${proxyName}" \
		-p KillMode=control-group \
		-p Wants="xdg-document-portal.service xdg-desktop-portal.service" \
		-p After="xdg-document-portal.service xdg-desktop-portal.service" \
		-p SuccessExitStatus=SIGKILL \
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
			--own="org.mpris.MediaPlayer2.${appID##*.}" \
			--own="org.mpris.MediaPlayer2.${appID##*.}".* \
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
	mkdir -p "${XDG_DATA_HOME}/${stateDirectory}/options"
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
				--text="对应用程序 ${appID} 启用沙盒?"
		else
			/usr/bin/zenity \
				--title "${friendlyName}" \
				--icon=security-medium-symbolic \
				--question \
				--text="Enable sandbox for: ${appID}?"
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
				mkdir -p "${XDG_DATA_HOME}/${stateDirectory}/options"
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
	if systemctl --user --quiet is-active "${unitName}.service"; then
		warnMulRunning "$@"
	fi
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

function passPid() {
	if [[ $(cat "${XDG_RUNTIME_DIR}/portable/${appID}/startSignal") = "app-started" ]]; then
		pecho warn "Application started before passPid()"
	else
		inotifywait \
			-e modify \
			--quiet \
			"${XDG_RUNTIME_DIR}/portable/${appID}/startSignal" 1>/dev/null
	fi
	getChildPid
	echo "${childPid}" > "${XDG_DATA_HOME}/${stateDirectory}/mainPid"
	unset childPid
	local childPid=$(systemctl --user show "${friendlyName}-dbus" -p MainPID | cut -c '9-')
	sed -i \
		"s|placeholderChildPid|${childPid}|g" \
		"${XDG_RUNTIME_DIR}/.flatpak/${instanceId}/bwrapinfo.json"

	sed -i \
		"s|placeholderMntId|$(readlink "/proc/${childPid}/ns/mnt" | sed 's/[^0-9]//g')|g" \
		"${XDG_RUNTIME_DIR}/.flatpak/${instanceId}/bwrapinfo.json"
	sed -i \
		"s|placeholderPidId|$(readlink "/proc/${childPid}/ns/pid" | sed 's/[^0-9]//g')|g" \
		"${XDG_RUNTIME_DIR}/.flatpak/${instanceId}/bwrapinfo.json"
	echo "finish" > "${XDG_RUNTIME_DIR}/portable/${appID}/startSignal"
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
		elif [[ $(systemctl --user list-units --state active --no-pager "${friendlyName}"* | grep subprocess | wc -l) -ge 2 ]]; then
			pecho crit "Not stopping the slice because two or more subprocesses are still running"
			exit 1
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
sourceXDG
if [[ "$*" = "--actions quit" ]]; then
	stopApp external
fi
questionFirstLaunch
manageDirs
cmdlineDispatcher "$@"
launch "$@"
