#!/usr/bin/bash

set -e

function buildGo() {
	currDir="$(pwd)"
	cd "$1"
	go mod download -modcacherw
	export CGO_CPPFLAGS="${CPPFLAGS}"
	export CGO_CFLAGS="${CFLAGS}"
	export CGO_CXXFLAGS="${CXXFLAGS}"
	export CGO_LDFLAGS="${LDFLAGS}"
	export GOFLAGS="-buildmode=pie -trimpath -ldflags=-linkmode=external -modcacherw"

	go build \
		-trimpath \
		-buildmode=pie \
		-modcacherw \
		-ldflags "-linkmode external -extldflags \"${LDFLAGS}\"" \
		.
	cd "${currDir}"
}

function buildRust() {
	currDir="$(pwd)"
	cd "$1"
	export RUSTUP_TOOLCHAIN=stable
	export CARGO_TARGET_DIR=target
	cargo fetch --locked --target host-tuple
	cargo build --frozen --release --all-features
	cd "${currDir}"
}

git submodule update --init --recursive

buildRust .

buildGo ./lib/daemon

buildRust ./lib/init

buildGo ./lib/open-ng
buildGo ./lib/flatpak-spawn-stub
buildGo ./lib/prlimit-stub