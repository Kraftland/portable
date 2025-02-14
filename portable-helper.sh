#!/usr/bin/bash

function waitForStart() {
	touch ~/startSignal
	while true; do
		sleep 0.05s
		grep --silent placeholderPidId "${XDG_RUNTIME_DIR}/flatpak-runtime-directory/bwrapinfo.json"
		if [ $? = 1 ]; then
			echo "Starting Application..."
			return 0
		fi
	done
}

waitForStart

$@