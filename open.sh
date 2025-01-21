#!/bin/bash

if [[ "$@" =~ "https://" ]] || [[ "$@" =~ "http://" ]]; then
	echo "[Info] Received a request: $@, interpreting as link"
	/usr/lib/flatpak-xdg-utils/xdg-open "$@"
	exit $?
fi

if [[ $1 =~ "/" ]]; then
	origReq="$1"
fi

if [[ $2 =~ "/" ]]; then
	origReq="$2"
fi

if [ ${trashAppUnsafe} ]; then
	link="${origReq}"
	xdg-open "${origReq}"
	exit $?
# elif [[ ${XDG_CURRENT_DESKTOP} = KDE ]]; then
# 	if [[ "$(echo $origReq | cut -c '1-8' )" =~ 'file://' ]]; then
# 		if [[ "$origReq" =~ 'file:///tmp' ]]; then
# 			echo "file:// link and /tmp path detected!"
# 			export link="/proc/$(cat ~/mainPid)/root/$(echo $origReq | sed 's|file:///||g')"
# 		else
# 			fakeDirBase="${HOME}"
# 			realDirBase="${XDG_DATA_HOME}/${stateDirectory}"
# 			export link=$(echo "$origReq" | sed "s|${fakeDirBase}|${realDirBase}|g" | sed 's|file://||g')
# 		fi
# 	else
# 		if [[ $(echo "${origReq}" | cut -c '-4') = '/tmp' ]]; then
# 			echo "/tmp path detected!"
# 			export link="/proc/$(cat ~/mainPid)/root${origReq}"
# 		else
# 			fakeDirBase="${HOME}"
# 			realDirBase="${XDG_DATA_HOME}/${stateDirectory}"
# 			link=$(echo "$origReq" | sed "s|${fakeDirBase}|${realDirBase}|g")
# 		fi
# 	fi
# else
# 	if [[ "$(echo $origReq | cut -c '1-8' )" =~ 'file://' ]]; then
# 		echo "file:// link detected!"
# 		export link="/proc/$(cat ~/mainPid)/root/$(echo $origReq | sed 's|file:///||g')"
# 	else
# 		export link="/proc/$(cat ~/mainPid)/root${origReq}"
# 	fi
fi

if [[ "${origReq}" =~ "${bwBindPar}" ]]; then
	echo "[Warn] Request is in bwBindPar!"
	if [[ "$(echo ${origReq} | cut -c '1-8' )" =~ 'file://' ]]; then
		export link="/proc/$(cat ~/mainPid)/root/$(echo $origReq | sed 's|file:///||g')"
	else
		export link="/proc/$(cat ~/mainPid)/root${origReq}"
	fi
else
	if [[ "$(echo ${origReq} | cut -c '1-8' )" =~ 'file://' ]]; then
		ln \
			-sfr \
			"$(echo ${origReq} | sed 's|file://||g')" ~/Shared
	else
		ln \
			-sfr \
			${origReq} ~/Shared
	fi
fi

link="${XDG_DATA_HOME}/${stateDirectory}/Shared/$(basename ${origReq})"

echo "[Info] received a request: $@, translated to ${link}"

# if [[ ${portableUsePortal} = 1 ]]; then
# 	/usr/lib/flatpak-xdg-utils/xdg-open $(dirname "${link}")
# 	if [[ $? = 0 ]]; then
# 		exit 0
# 	fi
# fi
echo "[Info] Initiating D-Bus call..."
dbus-send --print-reply --dest=org.freedesktop.FileManager1 \
	/org/freedesktop/FileManager1 \
	org.freedesktop.FileManager1.ShowItems \
	array:string:"file://${link}" \
	string:fake-dde-show-items

if [[ $? = 0 ]]; then
	exit 0
fi

/usr/lib/flatpak-xdg-utils/xdg-open $(dirname "${link}")

if [[ $? = 0 ]]; then
	exit 0
fi


if [ -f /usr/bin/dolphin ] && [ ${XDG_CURRENT_DESKTOP} = KDE ]; then
	/usr/bin/dolphin --select "${link}"
elif [ -f /usr/bin/nautilus ] && [ ${XDG_CURRENT_DESKTOP} = GNOME ]; then
	/usr/bin/nautilus $(dirname "${link}")
else
	xdg-open $(dirname "${link}")
fi
fi
