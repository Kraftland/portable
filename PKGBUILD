# Maintainer: Kimiblock Moe
pkgname=portable-git
pkgver=1
pkgrel=1
epoch=
pkgdesc="Sandboxing framework"
arch=('any')
url="https://github.com/Kraftland/portable"
license=()
groups=()
options=(!debug !strip)

makedepends+=()

depends=(
	"xdg-user-dirs"
	"xorg-xhost"
	"findutils"
	"zenity"
	"xdg-dbus-proxy"
	"nss"
	"bubblewrap"
	"xcb-util-renderutil"
	"xcb-util-keysyms"
	"xcb-util-image"
	"xcb-util-wm"
	"libxkbcommon-x11"
	"libxkbcommon"
	"libxcb"
	"util-linux"
	"openssl-1.1"
	"libxcb"
	"gcc-libs"
	"nspr"
	"bzip2"
	"glibc"
	"zlib"
	"libxcomposite"
	"glib2"
	"wayland"
	"libxrender"
	"libxext"
	"alsa-lib"
	"dbus"
	"libxrandr"
	"fontconfig"
	"pango"
	"freetype2"
	"libxfixes"
	"cairo"
	"libx11"
	"expat"
	"at-spi2-core"
	"libxdamage"
	"libdrm"
	"mesa"
	"hicolor-icon-theme"
	"bash"
	"lsb-release"
	"psmisc"
	"wmctrl"
	"flatpak-xdg-utils"
	"xdg-desktop-portal"
	"xdg-desktop-portal-gtk"
)

optdepends=(
	'at-spi2-core: accessibility'
	'orca: screen reader'
)

makedepends+=(
	"libarchive"
)

checkdepends=()

source=(
	"git+https://github.com/Kraftland/portable.git"
)

function package() {
	cd portable
	install -Dm755 portable.sh /usr/bin/portable
	install -Dm755 open.sh /usr/lib/portable/open
	install -Dm755 user-dirs.dirs /usr/lib/portable/user-dirs.dirs
	install -Dm755 mimeapps.list /usr/lib/portable/mimeapps.list
	install -Dm755 flatpak-info /usr/lib/portable/flatpak-info
}
