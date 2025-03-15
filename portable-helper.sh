#!/usr/bin/bash

function waitForStart() {
	inotifywait \
		-e modify \
		--quiet \
		~/startSignal
}

#waitForStart

function _sleep() {
	while true; do
		sleep 3650d
	done
}

echo "yes" >~/startSignal

_sleep &

$@