#!/usr/bin/bash

echo "Running hyperfine..."

env _portableConfig=./doc/dev/conf-perf PORTABLE_LOGGING=debug hyperfine --conclude 'sleep 1s' --warmup 10 --runs=100 --shell=none /usr/bin/portable