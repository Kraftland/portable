#!/usr/bin/bash

function waitForStart() {
	inotifywait \
		-e modify \
		--quiet \
		~/startSignal 1>/dev/null
}

echo "yes" >~/startSignal

waitForStart

$@

