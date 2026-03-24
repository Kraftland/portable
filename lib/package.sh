#!/usr/bin/bash

if [[ ${pkgdir} ]]; then
	installPrefix="${pkgdir}"
fi

install -d "${installPrefix}/usr/lib/"
cp -r "lib" "${installPrefix}/usr/lib/portable"
install -t "${installPrefix}/usr/share/portable" -Dm755 "share"/*
install -vDm755 portable-pools "${installPrefix}/usr/bin/portable-pools"
cp -r lib/modules-load.d "${installPrefix}/usr/lib"
ln -sf /usr/lib/portable/daemon/portable-daemon "${installPrefix}/usr/bin/portable"