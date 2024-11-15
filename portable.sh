#!/bin/bash

if [ ${_portalConfig} ]; then
	source "${_portalConfig}"
else
	echo "[Critical] No portable config specified!"
	exit 1
fi

busName="${appID}"
busDir="${XDG_RUNTIME_DIR}/app/${busName}"
unitName="${friendlyName}"
proxyName="${friendlyName}-dbus"

function moeDect() {
	if [[ -f /usr/share/moeOS-Docs/os-release ]]; then
		osRel="/usr/share/moeOS-Docs/os-release"
	else
		osRel="/usr/lib/os-release"
	fi
}

function sourceXDG() {
	if [[ ! ${XDG_CONFIG_HOME} ]]; then
		export XDG_CONFIG_HOME="${HOME}"/.config
		echo "[Info] Guessing XDG Config Home @ ${XDG_CONFIG_HOME}"
	else
		source "${XDG_CONFIG_HOME}"/user-dirs.dirs
		echo "[Info] XDG Config Home defined @ ${XDG_CONFIG_HOME}"
	fi
	if [[ ! ${XDG_DATA_HOME} ]]; then
		export XDG_DATA_HOME="${HOME}"/.local/share
	fi
	export XDG_DOCUMENTS_DIR="$(xdg-user-dir DOCUMENTS)"
}

function manageDirs() {
	createWrapIfNotExist "${XDG_DATA_HOME}"/${stateDirectory}
}

function detectXauth() {
	if [ ! ${XAUTHORITY} ]; then
		echo '[Warn] No ${XAUTHORITY} detected! Do you have any X server running?'
		export XAUTHORITYpath="/$(uuidgen)/$(uuidgen)"
		xhost +
	else
		export XAUTHORITYpath="${XAUTHORITY}"
	fi
	if [[ ! ${DISPLAY} ]]; then
		echo '[Warn] No ${DISPLAY} detected! Do you have any X server running?'
	fi
}

function createWrapIfNotExist() {
	if [ -d "$@" ]; then
		return 0
	else
		mkdir -p "$@"
	fi
}

function inputMethod() {
	if [[ ${XMODIFIERS} =~ fcitx ]] || [[ ${QT_IM_MODULE} =~ fcitx ]] || [[ ${GTK_IM_MODULE} =~ fcitx ]]; then
		export QT_IM_MODULE=fcitx
		export GTK_IM_MODULE=fcitx
	elif [[ ${XMODIFIERS} =~ ibus ]] || [[ ${QT_IM_MODULE} =~ ibus ]] || [[ ${GTK_IM_MODULE} =~ ibus ]]; then
		export QT_IM_MODULE=ibus
		export GTK_IM_MODULE=ibus
		IBUS_USE_PORTAL=1
	elif [[ ${XMODIFIERS} =~ gcin ]]; then
		export QT_IM_MODULE=ibus
		export GTK_IM_MODULE=gcin
		export LC_CTYPE=zh_TW.UTF-8
	else
		echo '[Warn] Input Method potentially broken! Please set $XMODIFIERS properly'
	fi
}

function importEnv() {
	if [ -e "${XDG_DATA_HOME}"/${stateDirectory}/portable.env ]; then
		echo "[Info] ${XDG_DATA_HOME}/${stateDirectory}/portable.env exists"
	else
		touch "${XDG_DATA_HOME}"/${stateDirectory}/portable.env
	fi
	if [[ $(cat "${XDG_DATA_HOME}"/${stateDirectory}/portable.env) ]]; then
		return 0
	else
		echo "# Envs" >>"${XDG_DATA_HOME}"/${stateDirectory}/portable.env
		echo "isPortableEnvPresent=1" >>"${XDG_DATA_HOME}"/${stateDirectory}/portable.env
	fi
}

function cameraDect() {
	bwCamPar=""
	for camera in $(ls /dev/video*); do
		if [ -e ${camera} ]; then
			bwCamPar="${bwCamPar} --dev-bind ${camera} ${camera}"
		fi
	done
}

function execApp() {
	if [ ! -S "${busDir}/bus" ]; then
		echo "[Info] Waiting for D-Bus proxy..."
		counter=0
		while [ ! -S "${busDir}/bus" ]; do
			counter=$(expr ${counter} + 1)
			sleep 0.1s
		done
		echo "[Info] D-Bus proxy took $(expr ${counter} / 10)s to launch"
	fi
	cameraDect
	importEnv
	if [ ${XDG_SESSION_TYPE} = wayland ]; then
		echo "[Info] Skipping Xhost operation"
	else
		xhost + #Unlock the XServer for X11 users
	fi
	mkdir -p "${XDG_DATA_HOME}"/"${stateDirectory}"/.config
	echo "GTK_IM_MODULE is ${GTK_IM_MODULE}"
	echo "QT_IM_MODULE is ${QT_IM_MODULE}"
	if [ ! ${bwBindPar} ]; then
		bwBindPar="/$(uuidgen)"
	else
		echo "bwBindPar is ${bwBindPar}"
	fi
	systemd-run \
	--user \
	${sdOption} \
	-u "${unitName}" \
	-p Description="Portable Sandbox" \
	-p Documentation="https://github.com/Kraftland/portable" \
	-p ExitType=cgroup \
	-p OOMPolicy=stop \
	-p KillMode=control-group \
	-p CPUAccounting=yes \
	-p StartupCPUWeight=idle \
	-p StartupIOWeight=1 \
	-p MemoryMax=90% \
	-p MemoryHigh=80% \
	-p LimitCORE=0 \
	-p CPUWeight=20 \
	-p IOWeight=20 \
	-p ManagedOOMSwap=kill \
	-p ManagedOOMMemoryPressure=kill \
	-p IPAccounting=yes \
	-p PrivateIPC=yes \
	-p DevicePolicy=strict \
	-p EnvironmentFile="${_portalConfig}" \
	-p EnvironmentFile="${XDG_DATA_HOME}/${stateDirectory}/portable.env" \
	-p Environment=GTK_IM_MODULE="${GTK_IM_MODULE}" \
	-p Environment=QT_IM_MODULE="${QT_IM_MODULE}" \
	-p IPAddressDeny=localhost \
	-p IPAddressDeny=link-local \
	-p IPAddressDeny=multicast \
	-p SystemCallFilter=~@clock \
	-p SystemCallFilter=~@cpu-emulation \
	-p SystemCallFilter=~@debug \
	-p SystemCallFilter=~@module \
	-p SystemCallFilter=~@obsolete \
	-p SystemCallFilter=~@raw-io \
	-p SystemCallFilter=~@reboot \
	-p SystemCallFilter=~@swap \
	-p SystemCallErrorNumber=EPERM \
	-p ProcSubset=pid \
	-p RestrictAddressFamilies=AF_UNIX \
	-p RestrictAddressFamilies=AF_INET \
	-p RestrictAddressFamilies=AF_INET6 \
	-p NoNewPrivileges=yes \
	-p RestrictNamespaces=~net \
	-p RestrictNamespaces=~pid \
	-p RestrictNamespaces=~uts \
	-p RestrictNamespaces=~ipc \
	-p ProtectControlGroups=yes \
	-p KeyringMode=private \
	-p ProtectClock=yes \
	-p CapabilityBoundingSet= \
	-p ProtectKernelModules=yes \
	-p SystemCallArchitectures=native \
	-p RestrictNamespaces=no \
	-p RestrictSUIDSGID=yes \
	-p LockPersonality=yes \
	-p RestrictRealtime=yes \
	-p ProtectSystem=strict \
	-p ProtectProc=invisible \
	-p ProtectHome=no \
	-p PrivateUsers=yes \
	-p UMask=077 \
	-p TimeoutStopSec=20s \
	-p RestrictAddressFamilies=~AF_PACKET \
	-p PrivateTmp=yes \
	-p BindReadOnlyPaths=/usr/bin/true:/usr/bin/lsblk \
	-p BindReadOnlyPaths=/dev/null:/proc/cpuinfo \
	-p BindReadOnlyPaths=/dev/null:/proc/meminfo \
	-p BindReadOnlyPaths=-/run/systemd/resolve/stub-resolv.conf \
	-p BindReadOnlyPaths=/usr/lib/portable/flatpak-info:"${XDG_RUNTIME_DIR}/.flatpak-info" \
	-p Environment=PATH=/sandbox:"${PATH}" \
	-- \
	bwrap \
		--tmpfs /tmp \
		--ro-bind-try /tmp/.X11-unix /tmp/.X11-unix \
		--dev /dev \
		--dev-bind /dev/dri /dev/dri \
		--dev-bind-try /dev/nvidia0 /dev/nvidia0 \
		--dev-bind-try /dev/nvidiactl /dev/nvidiactl \
		--dev-bind-try /dev/nvidia-modeset /dev/nvidia-modeset \
		--dev-bind-try /dev/nvidia-uvm /dev/nvidia-uvm \
		--tmpfs /sys \
		--bind /sys/module/ /sys/module/ \
		--ro-bind /sys/dev/char /sys/dev/char \
		--ro-bind /sys/devices /sys/devices \
		--dir /sandbox \
		--ro-bind /usr/lib/flatpak-xdg-utils/xdg-open \
			/sandbox/chromium \
		--ro-bind /usr/lib/flatpak-xdg-utils/xdg-open \
			/sandbox/firefox \
		--ro-bind /usr/lib/portable/mimeapps.list \
			"${XDG_DATA_HOME}/${stateDirectory}/.config/mimeapps.list" \
		--proc /proc \
		--bind /usr /usr \
		--ro-bind /etc /etc \
		--ro-bind-try /lib /lib \
		--ro-bind-try /lib64 /lib64 \
		--ro-bind-try /bin /bin \
		--ro-bind-try /sbin /sbin \
		--ro-bind-try /opt /opt \
		--bind "${busDir}/bus" "${XDG_RUNTIME_DIR}/bus" \
		--ro-bind "${XDG_RUNTIME_DIR}/pulse" \
			"${XDG_RUNTIME_DIR}/pulse" \
		--bind "${XDG_DATA_HOME}/${stateDirectory}" "${HOME}" \
		--ro-bind-try "${XDG_DATA_HOME}"/icons "${XDG_DATA_HOME}"/icons \
		--ro-bind-try "${XAUTHORITYpath}" "${XAUTHORITYpath}" \
		--ro-bind-try "${XDG_RUNTIME_DIR}/${WAYLAND_DISPLAY}" \
				"${XDG_RUNTIME_DIR}/${WAYLAND_DISPLAY}" \
		--ro-bind-try "${XDG_RUNTIME_DIR}/${WAYLAND_DISPLAY}.lock" \
				"${XDG_RUNTIME_DIR}/${WAYLAND_DISPLAY}.lock" \
		--ro-bind /usr/lib/portable/open \
			/sandbox/dde-file-manager \
		--ro-bind /usr/lib/portable/open \
			/sandbox/xdg-open \
		--ro-bind /usr/lib/portable/open \
			/sandbox/open \
		--ro-bind /usr/lib/portable/user-dirs.dirs \
			"${XDG_CONFIG_HOME}"/user-dirs.dirs \
		--ro-bind-try "${XDG_CONFIG_HOME}"/fontconfig \
			"${XDG_CONFIG_HOME}"/fontconfig \
		--ro-bind-try "${XDG_DATA_HOME}/fonts" \
			"${XDG_DATA_HOME}/fonts" \
		--ro-bind-try "/run/systemd/resolve/stub-resolv.conf" \
			"/run/systemd/resolve/stub-resolv.conf" \
		--dir "${XDG_DATA_HOME}/${stateDirectory}/Documents" \
		--bind "${XDG_DATA_HOME}/${stateDirectory}" \
			"${XDG_DATA_HOME}/${stateDirectory}" \
		--tmpfs "${XDG_DATA_HOME}/${stateDirectory}"/options \
		--ro-bind-try /dev/null \
			"${XDG_DATA_HOME}/${stateDirectory}"/portable.env \
		--bind-try "${bwBindPar}" "${bwBindPar}" \
		${bwCamPar} \
		--setenv XDG_DOCUMENTS_DIR "$HOME/Documents" \
		--setenv XDG_DATA_HOME "${XDG_DATA_HOME}" \
		--unshare-cgroup-try \
		--unshare-ipc \
		--unshare-uts \
		--unshare-user \
		--disable-userns \
		-- \
			"${launchTarget}"
}

function warnMulRunning() {
	wmctrl -a "微信"
	if [[ $? = 0 ]]; then
		exit 0
	else
		id=$(dbus-send \
			--bus=unix:path="${busDir}/bus" \
			--dest=org.kde.StatusNotifierWatcher \
			--type=method_call \
			--print-reply=literal /StatusNotifierWatcher \
			org.freedesktop.DBus.Properties.Get \
			string:org.kde.StatusNotifierWatcher \
			string:RegisteredStatusNotifierItems | grep -oP 'org.kde.StatusNotifierItem-\d+-\d+')
		echo "[Info] Unique ID: ${id}"
		dbus-send \
			--print-reply \
			--session \
			--dest=${id} \
			--type=method_call \
			/StatusNotifierItem \
			org.kde.StatusNotifierItem.Activate \
			int32:114514 \
			int32:1919810
		if [[ $? = 0 ]]; then
			exit 0
		fi
	fi
	if [[ "${LANG}" =~ 'zh_CN' ]]; then
		zenity --title "程序未响应" --icon=utilities-system-monitor-symbolic --default-cancel --question --text="是否结束正在运行的进程?"
	else
		zenity --title "Application is not responding" --icon=utilities-system-monitor-symbolic --default-cancel --question --text="Do you wish to terminate the running session?"
	fi
	if [ $? = 0 ]; then
		systemctl --user stop $@
	else
		echo "[Critical] User denied session termination"
		exit $?
	fi
}

function dbusProxy() {
	if [[ $(systemctl --user is-failed ${proxyName}.service) = failed ]]; then
		echo "[Warning] D-Bus proxy failed last time"
		systemctl --user reset-failed ${proxyName}.service
	fi
	if [[ $(systemctl --user is-active ${proxyName}.service) = active ]]; then
		echo "[Warning] Existing D-Bus proxy detected! Terminating..."
		systemctl --user kill ${proxyName}.service
	fi
	if [ -d "${busDir}" ]; then
		rm "${busDir}" -r
	fi
	mkdir "${busDir}" -p
	echo "Starting D-Bus Proxy @ ${busDir}..."
	systemd-run \
		--user \
		-u ${proxyName} \
		-- bwrap \
			--symlink /usr/lib64 /lib64 \
			--ro-bind /usr/lib /usr/lib \
			--ro-bind /usr/lib64 /usr/lib64 \
			--ro-bind /usr/bin /usr/bin \
			--ro-bind-try /usr/share /usr/share \
			--bind "${XDG_RUNTIME_DIR}" "${XDG_RUNTIME_DIR}" \
			--ro-bind /usr/lib/portable/flatpak-info \
				/.flatpak-info \
			-- /usr/bin/xdg-dbus-proxy \
			"${DBUS_SESSION_BUS_ADDRESS}" \
			"${busDir}/bus" \
			--filter \
			--own=org.kde.* \
			--talk=org.freedesktop.portal.Flatpak \
			--talk=org.freedesktop.portal.Desktop \
			--talk=org.freedesktop.portal.* \
			--talk=org.freedesktop.Notifications \
			--talk=org.freedesktop.FileManager1 \
			--talk=org.kde.StatusNotifierWatcher \
			--talk=org.freedesktop.portal.OpenURI \
			--talk=org.freedesktop.portal.OpenURI.* \
			--call=org.freedesktop.portal.*=* \
			--own="${busName}" \
			--broadcast=org.freedesktop.portal.*=@/org/freedesktop/portal/* \
			--call=org.a11y.atspi.Registry=org.a11y.atspi.DeviceEventController.NotifyListenersAsync@/org/a11y/atspi/registry/deviceeventcontroller \
			--call=org.a11y.atspi.Registry=org.a11y.atspi.DeviceEventController.NotifyListenersSync@/org/a11y/atspi/registry/deviceeventcontroller \
			--call=org.a11y.atspi.Registry=org.a11y.atspi.DeviceEventController.GetDeviceEventListeners@/org/a11y/atspi/registry/deviceeventcontroller \
			--call=org.a11y.atspi.Registry=org.a11y.atspi.DeviceEventController.GetKeystrokeListeners@/org/a11y/atspi/registry/deviceeventcontroller \
			--call=org.a11y.atspi.Registry=org.a11y.atspi.Registry.GetRegisteredEvents@/org/a11y/atspi/registry \
			--call=org.a11y.atspi.Registry=org.a11y.atspi.Socket.Unembed@/org/a11y/atspi/accessible/root \
			--call=org.a11y.atspi.Registry=org.a11y.atspi.Socket.Embed@/org/a11y/atspi/accessible/root
}

function execAppUnsafe() {
	source "${XDG_DATA_HOME}/${stateDirectory}/portable.env"
	echo "GTK_IM_MODULE is ${GTK_IM_MODULE}"
	echo "QT_IM_MODULE is ${QT_IM_MODULE}"
	systemd-run --user \
		-u ${unitName} \
		--tty \
		"${launchTarget}"
}

function questionFirstLaunch() {
	if [ ! -f "${XDG_DATA_HOME}"/${stateDirectory}/options/sandbox ]; then
		if [[ "${LANG}" =~ 'zh_CN' ]]; then
			zenity --title "初次启动" --icon=security-medium-symbolic --default-cancel --question --text="允许程序读取 / 修改所有个人数据?"
		else
			zenity --title "Welcome" --icon=security-medium-symbolic --default-cancel --question --text="Do you wish this Application to access and modify all of your data?"
		fi
		if [[ $? = 0 ]]; then
			export trashAppUnsafe=1
			if [[ "${LANG}" =~ 'zh_CN' ]]; then
				zenity --error --title "沙盒已禁用" --icon=security-low-symbolic --text "用户数据不再被保护"
			else
				zenity --error --title "Sandbox disabled" --icon=security-low-symbolic --text "User data is potentially compromised"
			fi
		else
			echo "Request canceled by user"
			mkdir -p "${XDG_DATA_HOME}"/${stateDirectory}/options
			touch "${XDG_DATA_HOME}"/${stateDirectory}/options/sandbox
			return 0
		fi
		mkdir -p "${XDG_DATA_HOME}"/${stateDirectory}/options
		echo disableSandbox >>"${XDG_DATA_HOME}"/${stateDirectory}/options/sandbox
	fi
	if [[ $(cat "${XDG_DATA_HOME}"/${stateDirectory}/options/sandbox) =~ "disableSandbox" ]]; then
		export trashAppUnsafe=1
	fi
}

function disableSandbox() {
	if [[ $@ =~ "f5aaebc6-0014-4d30-beba-72bce57e0650" ]] && [[ $@ =~ "--actions" ]]; then
		rm "${XDG_DATA_HOME}"/${stateDirectory}/options/sandbox
		questionFirstLaunch
	fi
}

function openDataDir() {
	if [[ $@ =~ "--actions" ]] && [[ $@ =~ "opendir" ]]; then
		xdg-open "${XDG_DATA_HOME}"/${stateDirectory}
		exit $?
	fi
}

function launch() {
	detectXauth
	inputMethod
	moeDect
	if [[ $(systemctl --user is-failed ${unitName}.service) = failed ]]; then
		echo "[Warning] ${appID} failed last time"
		systemctl --user reset-failed ${unitName}.service
	fi
	if [[ $(systemctl --user is-active ${unitName}.service) = active ]]; then
		warnMulRunning ${unitName}.service
	fi
	if [[ $@ =~ "--actions" ]] && [[ $@ =~ "debug-shell" ]]; then
		launchTarget="/usr/bin/bash"
	fi
	if [[ $@ =~ "--actions" ]] && [[ $@ =~ "connect-tty" ]]; then
		sdOption="-t"
	elif [[ $@ =~ "--actions" ]] && [[ $@ =~ "pipe-tty" ]]; then
		sdOption="-P"
	else
		sdOption=""
	fi
	if [[ ${trashAppUnsafe} = 1 ]]; then
		echo "Launching ${appID} (unsafe)..."
		execAppUnsafe
	else
		dbusProxy
		echo "Launching ${appID}..."
		execApp
	fi
}

function stopApp() {
	stopCmd="systemctl --user stop ${proxyName} ${unitName}"
}

if [[ $@ = "--actions quit" ]]; then
	stopApp $@
	exit $?
fi

sourceXDG
disableSandbox $@
questionFirstLaunch
openDataDir $@
manageDirs
launch $@

