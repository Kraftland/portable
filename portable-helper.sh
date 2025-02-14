#!/usr/bin/bash

function waitForStart() {
	echo 1 >"${XDG_RUNTIME_DIR}"/startSignal
	while true; do
		grep placeholderPidId "${XDG_RUNTIME_DIR}/.flatpak/${instanceId}/bwrapinfo.json"
		if [ $? = 1 ] && [ -S "${XDG_RUNTIME_DIR}"/app/"${appID}"/bus ]; then
			echo "Starting Application..."
			return 0
		fi
	done
}

waitForStart

$@