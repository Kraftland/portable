# What is this
Portable is a sandbox framework targeted for Desktop usage and offers ease of use for distro packagers, which should work on any recent FHS compliant system (plus, enables unprivileged user namespaces and uses recent version of systemd). It offers many useful features for users and packagers:

- Background Portal support.
- Wayland Security Context support.
- Access Control: Limits what the application can see, write and modify. Sandboxed applications are self-contained.
- Sharing files with the application, even if it doesn't support portals. portable creates a directory within the sandbox home to contain shared files.
- D-Bus filtering & accessibility support: Cuts off unneeded D-Bus messages thus eliminates the possibility to locate, spawn a process outside of the sandbox, mess with the host system and other possible exploits.
- Process Management: Monitors running processes and quit them with one click.
- Packaging Friendly as portable only requires a config file to function.
- Storage efficient compared to Flatpak: Using host system as the "runtime".
- Hybrid GPU workarounds are automatically applied to prevent waking up discrete GPUs, often caused by Vulkan and Electron applications.
- Input Method automatic detection.

## Available for

- [Minecraft](https://github.com/Kimiblock/moeOS.config/blob/master/usr/bin/mcLaunch)
- AUR
    - WeChat (aur/wechat)
    - Wemeet (aur/wemeet-bwrap)
    - Prism Launcher (aur/prismlauncher-bwrap)
    - Obsidian (aur/obsidian-bwrap)
    - Z-Library (aur/z-library-bwrap)
    - Wiliwili (aur/wiliwili-wayland)
    - WPS (aur/wps-office-cn-bwrap)
    - Genshin Impact Launcher
    - Bottles (aur/bottles-bwrap)
    - Visual Studio Code (aur/visual-studio-code-portable)
    - Discord (aur/discord-bwrap)
    - Larksuite (aur/larksuite-portable)
    - Firefox (aur/firefox-portable)
    - Thunderbird (aur/thunderbird-portable)
    - Baidu Netdisk (aur/baidunetdisk-portable)
    - DaVinci Resolve (aur/davinci-resolve-portable)
    - Zen Browser (aur/zen-browser-portable)
    - PCSX2 (aur/pcsx2-portable)
    - Spotify (aur/spotify-portable)
    - QQ (aur/linuxqq-portable)
    - Bitwarden (aur/bitwarden-portable)

# Limitations:

1. **Running untrusted code is never safe, sandboxing does not change this.**
2. WebKitGTK on a hybrid graphics laptop may require `gameMode=on`, otherwise may display a blank screen.
3. Steam will not work due to the requirement of Flatpak spawn portal.
4. On KDE Plasma window grouping may not work properly unless your desktop file name exactly matches certain arguments.
5. Due to some desktop portal implementations being insecure (without requiring user consent), some features will only be available on GNOME
    - The Global Shortcuts portal is only available on GNOME

Discuss Development at [#portable-dev:matrix.org](https://matrix.to/#/#portable-dev:matrix.org)

<h1 align="center">
  <img src="https://raw.githubusercontent.com/Kraftland/portable/refs/heads/master/share/example.webp" alt="The Portable Project" width="1024" />
  <br>
  Demo
  <br>
</h1>

---

# How to package?

See [Docs](https://github.com/Kraftland/portable/tree/master/doc)

## Starting portable

Start portable with environment variable `_portableConfig`, which can be 1) the appID of the sandbox, 2) an absolute path (if exists), 3) a file name interpreted as `$(pwd)/${_portableConfig}`. It searches for each of them respectively.

- Debugging output can be enabled using a environment variable `PORTABLE_LOGGING=debug`

### Launching multiple instances

Portable itself allows multiple instances. It automatically creates an identical sandbox and launches the application. Application itself may or may not support waking up itself. It is advised to set `SingleMainWindow=true` for applications that doesn't have well multi-instance support.

### Debugging

#### Entering sandbox

To manually execute programs instead of following the `launchTarget` config, start portable with argument `--actions debug-shell`. This will open a bash prompt and gives you full control of the sandbox environment.

---

# Pools

Pools is a user friendly sandbox generator. To create and enter a user sandbox, simply execute portable-pools with your sandbox name.

Example: Create a test sandbox:

```bash
portable-pools test

╰─>Portable Sandbox·top.kimiblock.test·🧐⤔
```

Usage:

```
portable-pools [Options] <Sandbox Name>

Options:
	--quit: Terminates the sandbox
```

# Code of Conduct

Portable and any of its social environment follows the [Kraftland Code of Conduct](https://blog.kimiblock.top/notice/#Code-of-Conduct). Please be sure not to violate such rule set.

# Version Scheme
Portable follows a major.minor.patch version scheme. We thrives to provide a stable experiences with no breaking changes, however, if said change is necessary, will land in a major release.

The patch release is exclusive for bug fixes. Whereas minor releases contain new features. If a feature or a set of features needs time to test or is important enough, we conduct a major release.

Portable has and always will be only supporting the latest release. Generally users can upgrade without manual intervention, but between major releases it's advised to run `systemctl --user stop portable.slice` to stop the portable framework.