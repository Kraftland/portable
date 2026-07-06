#!/usr/bin/bash

set -e

if [[ ${pkgdir} ]]; then
	installPrefix="${pkgdir}"
fi

install -vDm755 \
	"lib/init/target/release/init" \
	"${installPrefix}/usr/lib/portable/helper/helper"

install -d "${installPrefix}/usr/bin/"
install -d "${installPrefix}/usr/lib/"
cp -r "lib" "${installPrefix}/usr/lib/portable"
install -t "${installPrefix}/usr/share/portable" -Dm755 "share"/*
ln -sf \
	/usr/lib/portable/portable-pools \
	"${installPrefix}/usr/bin/portable-pools"
cp -r lib/modules-load.d "${installPrefix}/usr/lib"
ln -sf /usr/lib/portable/daemon/portable-daemon "${installPrefix}/usr/bin/portable"
install -d "${installPrefix}/usr/share/doc"
cp -r "doc" "${installPrefix}/usr/share/doc/portable"