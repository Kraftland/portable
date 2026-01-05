## Installing Portable

For Arch Linux and derivatives, install from Arch User Repository or [archlinuxcn]:
	- portable
		- Tagged, so-called stable version.
	- portable-git
		- Latest, development version.

For others, please find the respective distro package.

For packagers of portable:

Install the following files directly. Dependencies can be retrived through those official AUR packages.

```bash
	cd portable
	install -Dm755 portable.sh /usr/bin/portable
	install -d "/usr/lib/"
	cp -r "lib" "/usr/lib/portable"
	install -t "/usr/share/portable" -Dm755 "share"/*
	install -Dm755 portable-pools /usr/bin/portable-pools
```