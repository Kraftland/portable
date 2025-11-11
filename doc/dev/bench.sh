#!/usr/bin/bash

echo "Running hyperfine..."

hyperfine '_portableConfig=./doc/dev/conf-perf PORTABLE_LOGGING=debug ./portable.sh'

echo "Running ts..."

_portableConfig=./doc/dev/conf-perf PORTABLE_LOGGING=debug ./portable.sh | ts "%.S"