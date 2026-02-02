#!/usr/bin/bash

echo "Running hyperfine..."

systemd-run \
	--user \
	--tty \
	--same-dir \
	-p CPUWeight=10000 \
	-p MemoryLow=1G \
	-p CPUAffinity=0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15 \
	-- \
	hyperfine --warmup 10 --runs=500 --shell=none 'env _portableConfig=./doc/dev/conf-perf PORTABLE_LOGGING=debug /usr/bin/portable'