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
else
	echo "Warning! There're still processes running in the background."

	_state=$(notify-send --wait --action="kill"="Gracefully Terminate" --action="ignore"="Ignore" "Application running in background!" "Terminate as required")
	if [[ ${_state} = "kill" ]]; then
		echo "User opted to kill processes"
		kill %1
		for pid in /proc/[0-9]*; do
			pid="${pid#/proc/}"
			echo "Terminating process ${pid}" &
			if [[ $(cat /proc/${pid}/cmdline | tr '\000' ' ') =~ "/usr/lib/portable/helper" ]] || [[ ${pid} = 1 ]]; then
				echo "Skipping self..."
				continue
			fi
			kill "${pid}" &
		done
		sleep 1s
		exit 0
	else
		echo "User denied termination"
	fi
fi