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

$@ &

while true; do
	sleep 3650d
done

#rm ~/startSignal