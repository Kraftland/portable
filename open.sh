#!/bin/bash

function procOpen() {
	export link="/proc/$(cat ~/mainPid)/root${origReq}"
}

if [[ "$@" =~ "https://" ]] || [[ "$@" =~ "http://" ]]; then
	echo "[Info] Received a request: $@, interpreting as link"
	/usr/lib/flatpak-xdg-utils/xdg-open "$@"
	exit $?
fi

if [[ -e "$1" ]] || [[ -e "$(echo "$1" | sed 's|file:///|/|g')" ]]; then
	echo "Arg1: $1"
	export origReq="$1"
fi

if [[ -e "$2" ]] || [[ -e "$(echo "$2" | sed 's|file:///|/|g')" ]]; then
	echo "Arg2: $2"
	export origReq="$2"
fi

if [ ${trashAppUnsafe} ]; then
	link="${origReq}"
	xdg-open "${origReq}"
	exit $?
fi

if [[ $(echo ${origReq} | cut -c '1-8') =~ 'file://'  ]]; then
	echo "Received a request with file://: ${origReq}"
	export origReq="$(echo ${origReq} | sed 's|file:///|/|g')"
	echo "Decoding path as: ${origReq}"
else
	export origReq=$(realpath "${origReq}")
	echo "Interpreting origReq as ${origReq}"
fi

if [[ "${origReq}" =~ "/tmp" ]]; then
	echo "[Info] Detected /tmp!"
	procOpen
elif [[ "${origReq}" =~ "/run/user" ]]; then
	echo "[Info] Detected run path!"
	procOpen
elif [[ "$(dirname "${origReq}")" = "${HOME}" ]]; then
	echo "[Info] Detected sandbox home"
	link="${origReq}"
else
	link="${HOME}/Shared/$(basename "${origReq}")"
	ln \
		-sfr \
		"${origReq}" ~/Shared/
fi

echo "[Info] received a request: $@, translated to ${link}"
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
