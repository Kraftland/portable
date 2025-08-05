# Multiple instances in Portable

---

Portable starts the `launchTarget` by default, the executable path defined by package maintainers. It is considered the main application.

When it terminates for any reason, portable checks for any left processes. If true, it then notifies you about this and let you choose between stopping the whole sandbox and let it run freely.

If you choose the latter, the sandbox remains as long as there are processes running in the background. You can view them easily on GNOME via the Background Apps feature, or via `systemctl --user status ${friendlyName}`.