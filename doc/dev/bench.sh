#!/usr/bin/bash

echo "Running hyperfine..."

env PORTABLE_CONF=./doc/dev/perf.toml PORTABLE_LOGGING=debug hyperfine --warmup 10 --runs=100 --shell=none /usr/bin/portable

PORTABLE_CONF=./doc/dev/perf.toml PORTABLE_LOGGING=debug /usr/bin/portable  | ts "%.S"