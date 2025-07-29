# Environment Variables

Environment variables are sourced from `XDG_DATA_HOME/stateDirectory/portable.env`.

You can also specify environment variables in the config file. Though, some environment variables, like `GNOME_SETUP_DISPLAY` will be discarded.

It is worth noting that setting environment variable for portable doesn't work for the underlying application sandbox. That of the environment should be set in a global manner, i.e. at least for the user service manager.

Some environments, listed below, will have special effects for portable or the sandbox environment:

- XMODIFIERS
	- Controls the Input Method detection of portable, which controls `QT_IM_MODULE` and `GTK_IM_MODULE` in the sandbox
- bwBindPar
	- Bind a specified directory into sandbox. Notice that it requires user consent every time.

# Input Method
Portable automatically determines which environment variable are required for the user, based on the aforementioned `XMODIFIERS` variable and the host process tree.

It currently supports Fcitx5 and iBus, with experimental support for SCIM.