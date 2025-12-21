# Possible escapes

Work-in-progress possible holes in packaging. Below is a checklist for ruling it out:

- [ ] Always, go through all of the files in a package
- [ ] Inspect the dependencies
- [ ] Clear the following directories:
	- [ ] /etc/xdg 			for auto start
	- [ ] /usr/share/dbus-1		for D-Bus services
	- [ ] /usr/share/menu		for XDG menus
	- [ ] /usr/share/applications	for extra .desktop files
	- [ ] /usr/bin			for surplus binary files
	- [ ] /usr/share/gnome-shell	for search providers