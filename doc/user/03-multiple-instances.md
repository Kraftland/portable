# Multiple instances in Portable

---

Portable starts the `launchTarget` by default, of which is the command line defined by package maintainers. It is considered the main application. Whenever you start an auxiliary instance, or launch the sandbox itself, processes are tracked by the sandbox init. They are considered as "user started processes".

Once all of them exits for whatever reason, Portable by default terminates the sandbox. This prevents nasty stale processes while also ensures user processes are not killed. This behaviour can be changed however: when configuration option `terminateImmediately` is false, the sandbox stays in the background forever.  You can view them easily on GNOME via the Background Apps feature, or via `portable --actions stats`.