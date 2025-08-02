# Theming in Portable

Currently we support GTK theming via the following means:

- All globally installed themes are exposed and available
- `XDG_CONFIG_HOME/gtk-3.0/gtk.css` and `XDG_CONFIG_HOME/gtk-4.0/gtk.css` are exposed to the sandbox.
- Applications can get the active GTK theme via the Settings Portal.

Currently we support Qt theming via the following means:
- All globally installed themes are exposed and available
- `XDG_CONFIG_HOME/qt6ct` is exposed to the sandbox
- When qt5Compat is disabled, Qt platform theme variable is preserved.