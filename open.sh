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
	link="$2"
	xdg-open "$2"
	exit $?
else
	if [[ "$(echo $origReq | cut -c '1-8' )" =~ 'file://' ]]; then
		echo "file:// link detected!"
		export link="/proc/$(cat ~/mainPid)/root/$(echo $origReq | sed 's|file:///||g')"
		echo "[Info] received a file open request: $origReq, translated to ${link}"
	else
		export link="/proc/$(cat ~/mainPid)/root${origReq}"
	fi
fi



echo "[Info] received a request: $@, translated to ${link}"

if [[ ${portableUsePortal} = 1 ]]; then
	/usr/lib/flatpak-xdg-utils/xdg-open $(dirname "${link}")
	if [[ $? = 0 ]]; then
		exit 0
	fi
fi
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
