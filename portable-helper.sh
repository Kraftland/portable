#!/usr/bin/bash

function waitForStart() {
	inotifywait \
		-e modify \
		--quiet \
		~/startSignal
}

#waitForStart

echo "yes" >~/startSignal

#waitForStart

$@

#rm ~/startSignal