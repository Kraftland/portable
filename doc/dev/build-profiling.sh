#!/usr/bin/bash

RUSTFLAGS="-C force-frame-pointers=yes -C symbol-mangling-version=v0" cargo build