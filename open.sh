#!/bin/bash

function procOpen() {
	export link="/proc/$(cat ~/mainPid)/root${origReq}"
}

if [[ "$@" =~ "https://" ]] || [[ "$@" =~ "http://" ]]; then
	echo "[Info] Received a request: $@, interpreting as link"
	/usr/lib/flatpak-xdg-utils/xdg-open "$@"
	exit $?
fi

if [[ $1 =~ "/" ]]; then
	export origReq="$1"
fi

if [[ $2 =~ "/" ]]; then
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

if [ ! -z ${bwBindPar} ]; then
	export bwBindPar="$(realpath ${bwBindPar})"
	echo "bwBindPar set to ${bwBindPar}"
fi

if [[ "${origReq}" =~ "${bwBindPar}" ]] && [ ! -z ${bwBindPar} ]; then
	echo "[Info] Detected bwBindPar!"
	procOpen
elif [[ "${origReq}" =~ "/tmp" ]]; then
	echo "[Info] Detected /tmp!"
	procOpen
else
	ln \
		-sfr \
		"${origReq}" ~/Shared/$(basename ${origReq})
	link="${XDG_DATA_HOME}"/"${stateDirectory}"/Shared/$(basename "${origReq}")
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
