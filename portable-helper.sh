#!/usr/bin/bash

function waitForStart() {
	inotifywait \
		-e modify \
		--quiet \
		~/startSignal 1>/dev/null
}

function startLoop() {
	inotifywait \
		-e modify \
		--quiet \
		~/startSignal 1>/dev/null
	$(cat ~/startSignal)
}

echo "app-started" >~/startSignal

waitForStart

$@

