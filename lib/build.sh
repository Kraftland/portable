#!/usr/bin/bash

function buildGo() {
	currDir="$(pwd)"
	cd "$1"
	go mod download -modcacherw
	export CGO_CPPFLAGS="${CPPFLAGS}"
	export CGO_CFLAGS="${CFLAGS}"
	export CGO_CXXFLAGS="${CXXFLAGS}"
	export CGO_LDFLAGS="${LDFLAGS}"
	export GOFLAGS="-buildmode=pie -trimpath -ldflags=-linkmode=external -mod=readonly -modcacherw"

	go build \
		-trimpath \
		-buildmode=pie \
		-modcacherw \
		-mod=readonly \
		-ldflags "-linkmode external -extldflags \"${LDFLAGS}\"" \
		.
	cd "${currDir}"
}

buildGo ./lib/daemon

buildGo ./lib/helper
buildGo ./lib/open-ng