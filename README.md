# What is this
Portable is a sandbox framework targeted for Desktop usage and offers ease of use for distro packagers, which should work on any recent FHS compliant system:

- enables unprivileged user namespaces
- uses systemd >=258
- Does not have mount points under /usr/bin, and use a supported fs of OverlayFS (NOT BcacheFS)

It offers many useful features for users and packagers.

# Why Portable?

See [Docs](https://github.com/Kraftland/portable/blob/master/doc/Why%20Portable.md)

## Available for

- [Minecraft](https://github.com/Kimiblock/moeOS.config/blob/master/usr/bin/mcLaunch)
- Arch Linux
	- Arch Linux CN Repository
		- Only selected free/OSS apps
		- Updates faster
	- Portable for Arch
		- Configure paru to use [portable-arch](https://github.com/Kraftland/portable-arch): https://github.com/Kraftland/portable-arch
		- Current support status (as of 14 Oct 2025): 26 packages in repo.

# Limitations:

1. **Running untrusted code is never safe, sandboxing does not change this.**
2. WebKitGTK on a hybrid graphics laptop may require `gameMode=on`, otherwise may display a blank screen.
3. Steam will not work due to the requirement of Flatpak spawn portal.
4. On KDE Plasma window grouping may not work properly unless your desktop file name exactly matches certain arguments.
5. Due to some desktop portal implementations being insecure (without requiring user consent), some features will only be available on GNOME
	- The Location portal is only available on GNOME
6. Portable acts like Flatpak, to trick XDG Desktop Portal.
	- The correct way for this situation is to specify another sandboxing engine in XDP, which I have a PoC [here](https://github.com/Kimiblock/xdg-desktop-portal/commit/199c0934035789986b98738b01b15edf0443d675)
		- I barely understand C at all! Please help if you will.
	- The other possibly "correct way" is to wait until [busd#34](https://github.com/dbus2/busd/issues/34), and XDP's implementation.
		- Is it dead? idk.

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

# FAQ / Troubleshooting
1. Portable fails with something like _no such device_
	- Try reboot your system
2. Portable fails with something like _invalid argument_
	- BcacheFS is not supported, or you have mountpoints under `/usr/bin` and `/usr/lib`
3. Portable eats a full CPU core!
	- Try updating your microcode first, if not fixed then report an issue with `PORTABLE_LOGGING=debug` environment variable.

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
