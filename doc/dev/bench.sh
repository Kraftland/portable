#!/usr/bin/bash

echo "Running hyperfine..."

hyperfine '_portalConfig=./doc/dev/conf-perf PORTABLE_LOGGING=debug ./portable.sh'

echo "Running ts..."

_portalConfig=./doc/dev/conf-perf PORTABLE_LOGGING=debug ./portable.sh | ts "%.S"