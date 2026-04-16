# File Forwarding

Portable has a logic to forward files specified in command line options. This allows text editors to open files outside of a sandbox without calling Portal File Chooser.

From the previous chapter, we already know that:

> Any argument after this double dash will be passed to the application

When File Forwarding is enabled via the `--forward-file` flag, Portable will evaluate all application arguments. An argument will be designated as suitable for forwarding if:

1. It is an absolute path
2. It is not a directory
3. The user gives consent for this operation

After a path is determined as eligible, it is passed in form of a open file descriptor into the sandbox and stored under a location `$XDG_RUNTIME_DIR/doc/$ID/$(basename $FILE)`. Though there is no need to care about this technical detail because Portal stores a mapping of the outside host path and this sandbox path. When the sandbox starts a new executable, arguments will be automatically replaced to reflect the sandbox path. This is essentially the same as `--expose /path null` for now, but it may change in the future.