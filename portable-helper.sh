#!/usr/bin/bash

function waitForStart() {
	touch ~/startSignal
	inotifywait \
		-e modify \
		--quiet \
		~/startSignal
}

waitForStart

$@