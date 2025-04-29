#!/usr/bin/bash

function waitForStart() {
	inotifywait \
		-e modify \
		--quiet \
		~/startSignal 1>/dev/null
}

function startLoop() {
	while true; do
		inotifywait \
			-e modify \
			--quiet \
			~/startSignal 1>/dev/null
		_launch="$(cat ~/startSignal)"
		if [[ ${_launch} = terminate ]]; then
			break
		fi
		echo "Starting application"
		$(cat ~/startSignal) &
	done
}

echo "app-started" >~/startSignal

startLoop &

waitForStart

$@

if [ $(ps aux | wc -l) = "7" ]; then
	echo "No more application running, terminating..."
	#kill %1
	echo terminate >~/startSignal
	exit 0
fi