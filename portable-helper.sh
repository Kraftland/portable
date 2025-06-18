#!/usr/bin/bash

function waitForStart() {
	inotifywait \
		-e modify \
		--quiet \
		/run/startSignal 1>/dev/null
}

function startLoop() {
	while true; do
		inotifywait \
			-e modify \
			--quiet \
			/run/startSignal 1>/dev/null
		_launch="$(cat /run/startSignal)"
		if [[ ${_launch} =~ terminate ]]; then
			break
		elif [[ ${_launch} = finish ]]; then
			return 0
		fi
		echo "Starting application"
		$(cat /run/startSignal) &
	done
}

function stopApp() {
	echo "terminate-now" >/run/startSignal
	return $?
}

echo "app-started" >/run/startSignal

startLoop &
trap "stopApp" SIGTERM SIGINT SIGHUP SIGQUIT
waitForStart

cmd=$1
shift
"$cmd" "$@"

if [[ $(ps aux | wc -l) -le 5 ]]; then
	echo "No more application running, terminating..."
	#kill %1
	echo terminate-now >/run/startSignal
	exit 0
else
	echo "Warning! There're still processes running in the background."

	_state=$(notify-send --expire-time=10000 --wait --action="kill"="Gracefully Terminate" --action="ignore"="Ignore" "Application running in background!" "Terminate as required")
	if [[ ${_state} = "kill" ]]; then
		echo "User opted to kill processes"
		stopApp
	else
		echo "User denied termination, staying in background indefinitely..."
		while true; do
			sleep 3650d
		done
	fi
fi
