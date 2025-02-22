#!/usr/bin/bash

#function waitForStart() {
	#echo 1 >~/startSignal
	#inotifywait \
	#	-e modify \
	#	--quiet \
	#	~/startSignal
#}

#waitForStart

$@

#rm ~/startSignal