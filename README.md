# Abstract
A sandboxing framework, originally designed for WeChat. Still in heavy development.

Discuss Development at [#portable-dev:matrix.org](https://matrix.to/#/#portable-dev:matrix.org)

<h1 align="center">
  <img src="https://raw.githubusercontent.com/Kraftland/portable/refs/heads/master/example.webp" alt="The Portable Project" width="1024" />
  <br>
  Portable
  <br>
</h1>

---

# File installment

## Portable

Install aur/portable-git, aur/portable or install files directly

```
install -Dm755 portable.sh /usr/bin/portable
install -Dm755 open.sh /usr/lib/portable/open
install -Dm755 user-dirs.dirs /usr/lib/portable/user-dirs.dirs
install -Dm755 mimeapps.list /usr/lib/portable/mimeapps.list
install -Dm755 flatpak-info /usr/lib/portable/flatpak-info
```

## Configurations


Preferred location:

```
# Modify before installing
install -Dm755 config /usr/lib/portable/info/appID/config
```

## Runtime

Environment variables are read from `XDG_DATA_HOME/stateDirectory/portable.env`

Start portable with environment variable `_portalConfig`, which is pointed to the actual config.

### Arguments

`--actions f5aaebc6-0014-4d30-beba-72bce57e0650`: Toggle Sandbox, requires user confirmation.

`--actions opendir`: Open the sandbox's home directory

`--actions quit`: Stop sandbox and D-Bus proxy. If the app fails to stop after 20s, it'll be killed.

### Debugging

#### Entering sandbox
Start portable with argument `--actions connect-tty debug-shell`

Optionally enable debugging output using environment variable `PORTABLE_LOGGING=debug`
