#!/usr/bin/bash

set -e

if [[ ${pkgdir} ]]; then
	installPrefix="${pkgdir}"
fi

install -vDm755 \
	"lib/init/target/release/init" \
	"${installPrefix}/usr/lib/portable/helper/helper"

install -vDm755 \
	"lib/daemon/portable-daemon" \
	-t \
	"${installPrefix}/usr/lib/portable/daemon/"

install -vDm755 \
	"lib/daemon/portable-daemon" \
	"${installPrefix}/usr/bin/portable"

install -vDm644 \
	"lib/flatpak-info" \
	-t \
	"${installPrefix}/usr/lib/portable/"

install -vDm755 \
	"lib/flatpak-spawn-stub/top.kimiblock.flatpak-spawn" \
	-t \
	"${installPrefix}/usr/lib/portable/flatpak-spawn-stub/"

install -vDm644 \
	"lib/modules-load.d"/* \
	-t \
	"${installPrefix}/usr/lib/portable/modules-load.d"

install -vDm755 \
	"lib/open-ng/top.kimiblock.sandboxopen" \
	-t \
	"${installPrefix}/usr/lib/portable/open-ng/"

install -d "${installPrefix}/usr/lib/portable"

cp -r \
	"lib/overlay-usr" \
	"${installPrefix}/usr/lib/portable/"

install -vDm755 \
	"lib/portable-pools" \
	-t \
	"${installPrefix}/usr/bin"

install -vDm755 \
	"lib/prlimit-stub/top.kimiblock.prlimit" \
	-t \
	"${installPrefix}/usr/lib/portable/prlimit-stub/"


install -t "${installPrefix}/usr/share/portable" -Dm755 "share"/*

install -d "${installPrefix}/usr/share/doc"
cp -r "doc" "${installPrefix}/usr/share/doc/portable"