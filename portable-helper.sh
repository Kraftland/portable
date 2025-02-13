#!/usr/bin/bash

function waitForStart() {
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