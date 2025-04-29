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
		echo "Starting application"
		$(cat ~/startSignal) &
	done
}

echo "app-started" >~/startSignal

waitForStart

startLoop &

$@

