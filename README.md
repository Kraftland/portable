# What is this
Portable is a sandbox framework targeted for Desktop usage and offers ease of use for packagers. It provides isolation to the filesystem, in addition to blocking non-portal calls, filtering accessibility calls, it also stops unsafe portals from being used like the location portal and screenshot portal. Portable itself is still in development and have already been applied to [Minecraft](https://github.com/Kimiblock/moeOS.config/blob/master/usr/bin/mcLaunch), WeChat, Wemeet, QQ and Discord.

**Running untrusted code is never safe, sandboxing does not change this.**

Discuss Development at [#portable-dev:matrix.org](https://matrix.to/#/#portable-dev:matrix.org)

<h1 align="center">
  <img src="https://raw.githubusercontent.com/Kraftland/portable/refs/heads/master/example.webp" alt="The Portable Project" width="1024" />
  <br>
  Demo
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
install -Dm755 bwrapinfo.json /usr/lib/portable/bwrapinfo.json
```

## Configurations


Preferred location:

```
# Modify before installing
install -Dm755 config /usr/lib/portable/info/appID/config
```

## Runtime

Environment variables are read from `XDG_DATA_HOME/stateDirectory/portable.env`

Start portable with environment variable `_portableConfig`, which is pointed to the actual config. It searches absolute path (if exists), `/usr/lib/portable/info/${_portableConfig}/config` and `$(pwd)/${_portableConfig}` respectively. The legacy `_portalConfig` will work for future releases.

- Debugging output can be enabled using a environment variable `PORTABLE_LOGGING=debug`

### Launching multiple instances

Portable itself allows multiple instances. It automatically creates an identical sandbox and launches the application. Application itself may or may not support waking up itself.

## .desktop requirements

The name of your .desktop file should match the appID, like `top.kimiblock.example.desktop`

Your .desktop file should contain the following entries:

```
X-Flatpak-Tags=aTag;
X-Flatpak=appID;
X-Flatpak-RenamedFrom=previousName.desktop;
```

### Arguments

`--actions f5aaebc6-0014-4d30-beba-72bce57e0650`: Toggle Sandbox, requires user confirmation.

`--actions opendir`: Open the sandbox's home directory.

`--actions share-files`: Choose multiple files to share with the sandbox. The file will be temporarily stored in `XDG_DATA_HOME/stateDirectory/Shared`, which is purged each launch.

`--actions quit`: Stop sandbox and D-Bus proxy. If the app fails to stop after 20s, it'll be killed.

`--actions inspect`: Enters the sandbox.

`--actions debug-shell`: Start bash in the sandbox instead of the application itself.

### Debugging

#### Entering sandbox

To manually execute programs instead of following the `launchTarget` config, start portable with argument `--actions debug-shell`. This will open a bash prompt and gives you full control of the sandbox environment.

If the sandbox is already running, then _debug-shell_ is not usable. Instead you may want to use `--actions inspect`. This utilizes nsenter to enter the sandbox's user, mount and other namespaces respectively. This does now exactly mimic a portable sandbox environment, but useful if you want to poke around while the app is running.

# Repository mirror
This repository is available @ [Codeberg](https://codeberg.org/Kimiblock/portable) due to AUR packaging for Chinese users.
