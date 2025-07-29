### CLI arguments

Useful to include in .desktop entries

`--actions f5aaebc6-0014-4d30-beba-72bce57e0650`: Toggle Sandbox, requires user confirmation.

`--actions opendir`: Open the sandbox's home directory.

`--actions share-files`: Choose multiple files to share with the sandbox. The file will be temporarily stored in `XDG_DATA_HOME/stateDirectory/Shared`, which will be purged each launch.

`--actions reset-documents`: Resets XDG Desktop Portal's documents portal and other permissions, causing an instant redaction the permission to read & write for sandboxed apps. **Pleased be advised that this is suggested not to be executed when there are applications using the documents portal**!

`--actions quit`: Stop sandbox and D-Bus proxy. If the app fails to stop after 20s, it'll be killed.

`--actions debug-shell`: Start a debugging bash shell inside the sandbox. This works regardless whether the app is running.

Arguments for the application can be passed by putting your arguments in the end of the list of arguments and separate it with `--`