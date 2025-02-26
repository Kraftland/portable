# What is this
Portable is a sandbox framework targeted for Desktop usage and offers ease of use for distro packagers. It offers many useful features for users and packagers:

- Background Portal support.
- Access Control: Limits what the application can see, write and modify. Sandboxed applications are self-contained.
- Sharing files with the application, even if it doesn't support portals. portable creates a directory within the sandbox home to contain shared files.
- D-Bus filtering & accessibility support: Cuts off unneeded D-Bus messages thus eliminates the possibility to locate, spawn a process outside of the sandbox, mess with the host system and other possible exploits.
- Process Management: Monitors running processes and quit them with one click.
- Packaging Friendly as portable only requires a config file to function.
- Storage efficient compared to Flatpak: Using host system as the "runtime".
- Hybrid GPU workarounds are automatically applied to prevent waking up discrete GPUs, often caused by Vulkan and Electron applications.

Portable itself is still in development and have already been applied to [Minecraft](https://github.com/Kimiblock/moeOS.config/blob/master/usr/bin/mcLaunch), WeChat, Wemeet, Obsidian, QQ and Discord.

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

Install aur/portable-git, aur/portable or install the following files directly

```
install -Dm755 portable.sh /usr/bin/portable
install -Dm755 open.sh /usr/lib/portable/open
install -Dm755 portable-pools /usr/bin/portable-pools
install -Dm755 mimeapps.list /usr/lib/portable/mimeapps.list
install -Dm755 flatpak-info /usr/lib/portable/flatpak-info
install -Dm755 bwrapinfo.json /usr/lib/portable/bwrapinfo.json
install -Dm755 portable-helper.sh /usr/lib/portable/helper
```

## Configurations

1. Download the config file from [here](https://raw.githubusercontent.com/Kraftland/portable/refs/heads/master/config) and modify it.

2. Install the config file:

```bash
# Modify before installing
install -Dm755 config /usr/lib/portable/info/${appID}/config
```

### Modifying the .desktop entry

- The file name of your .desktop file **must** match the appID, like `top.kimiblock.example.desktop`

- Your .desktop file *should* contain the following entries:

```
X-Flatpak-Tags=aTag;
X-Flatpak=appID;
X-Flatpak-RenamedFrom=previousName.desktop;
```

## Environment Variables

Environment variables are sourced from `XDG_DATA_HOME/stateDirectory/portable.env`.

You can also specify environment variables in the config file.

## Starting portable

Start portable with environment variable `_portableConfig`, which can be 1) the appID of the sandbox, 2) an absolute path (if exists), 3) a file name interpreted as `$(pwd)/${_portableConfig}`. It searches for each of them respectively.

- Debugging output can be enabled using a environment variable `PORTABLE_LOGGING=debug`

### Launching multiple instances

Portable itself allows multiple instances. It automatically creates an identical sandbox and launches the application. Application itself may or may not support waking up itself. It is advised to set `SingleMainWindow=true` for applications that doesn't have well multi-instance support.

### CLI arguments

`--actions f5aaebc6-0014-4d30-beba-72bce57e0650`: Toggle Sandbox, requires user confirmation.

`--actions opendir`: Open the sandbox's home directory.

`--actions share-files`: Choose multiple files to share with the sandbox. The file will be temporarily stored in `XDG_DATA_HOME/stateDirectory/Shared`, which will be purged each launch.

`--actions quit`: Stop sandbox and D-Bus proxy. If the app fails to stop after 20s, it'll be killed.

`--actions debug-shell`: Start a debugging bash shell inside the sandbox. This works regardless whether the app is running.

### Debugging

#### Entering sandbox

To manually execute programs instead of following the `launchTarget` config, start portable with argument `--actions debug-shell`. This will open a bash prompt and gives you full control of the sandbox environment.

