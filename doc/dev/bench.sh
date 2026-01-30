#!/usr/bin/bash

echo "Running hyperfine..."

hyperfine --warmup 10 --runs=500 --shell=none 'env _portableConfig=./doc/dev/conf-perf PORTABLE_LOGGING=debug /usr/bin/portable'