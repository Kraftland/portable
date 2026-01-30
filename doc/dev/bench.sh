#!/usr/bin/bash

echo "Running hyperfine..."

hyperfine --warmup 5 --show-output --runs=500 --shell=none 'env _portableConfig=./doc/dev/conf-perf PORTABLE_LOGGING=debug /usr/bin/portable'