#!/usr/bin/bash

set -e

# $1 is po dir, $2 is DOMAIN name. Outputs to build/locale
function buildLocales() {
	for po in "$1"/*.po; do
		[ -f "$po" ] || continue
		lang=${po##*/}
		lang=${lang%.po}

		output="build/locale/$lang/LC_MESSAGES/$2.mo"
		mkdir -p "$(dirname "$output")"
		msgfmt -o "$output" "$po"
	done
}

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
	cargo build --frozen --release
	cd "${currDir}"
}

git submodule update --init --recursive

# Build locales for portable-config
mkdir -p locales

buildLocales lib/config/po "portable-config"

buildRust .

buildRust ./lib/init

buildGo ./lib/open-ng
buildGo ./lib/flatpak-spawn-stub
buildGo ./lib/prlimit-stub