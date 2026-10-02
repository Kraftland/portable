# Abstract
Portable is a security and privacy focused speedy Linux desktop sandbox.

Discuss Development at [#portable-dev:matrix.org](https://matrix.to/#/#portable-dev:matrix.org), or [Portable Sandbox](https://t.me/PortableSandbox)

## Why Portable?

### Device management
Device access is controlled tightly in Portable, applications are forbidden to render on or wake discrete GPU except when user gives permission, devices are mostly hidden by default, guarded behind multiple fine-grained permissions (see [blog post](https://blog.kimiblock.top/2026/09/16/gpu-boot-display/) for more details).

Applications use Integrated GPU by default, Discrete GPU when needed. Configurable per-package, authorised by user, no extra workaround needed.

Input devices support without allowing all devices access, enables full DualSense capability.

### Privacy and Security

Processes running inside a Portable sandbox have multiple lines of defence imposed. Multiple LSM, kernel and userspace technologies like Seccomp, Wayland security-context-v1 protocol and Landlock work in unity to ensure a secure, locked-down environment backed by multi-layer defence.

Security boundaries, such as filtering on the session D-Bus are mandatory and can't be turned off.

Portable actively hides multiple Operating System persistent identifiers, including but not limited to:

- Machine ID (see [blog post](https://blog.kimiblock.top/2026/09/11/machine-id/) for more details)
- User DB records
- Password File
- Control Group hierarchy
- PCI system devices
- Running Processes

### Dynamic Permissions
Dynamic Permissions in Portable avoids unintended resource access by enforcing user consent.

Portable requires an user audit before enabling static, package-defined permissions. Dynamic / Hybrid Permissions persists until user reset or package change.

See [blog post](https://blog.kimiblock.top/2026/09/30/dynamic-versus-static/) for more details.

### Packaging friendly
Use distribution packaging infrastructure without re-inventing the wheel. Companion projects make packaging easy.

### Transient sandbox
Create a secure sandbox in one step using `bawn <name>`. Perfect for compiling foreign projects, running command-line programs, trying out new apps, etc.

See [bawn](https://github.com/Kimiblock/bawn) for the successor of Portable Pools.

### Desktop Integration
Get sandbox status in _Background Apps_ at a glance. (requires supported desktop environment, like GNOME Shell)

Automatic input method settings application for legacy X11 environment.

Accessibility support for the at-spi bus, including the orca screen reader.

File sharing for legacy applications.

### Built for speed and efficiency
Portable employs multiple techniques, such as asynchronous execution to get faster startup. Being a host sandbox enables packagers to re-use system libraries and resources.

### Resource management
Powered by systemd and unified control group hierarchy, Portable prohibits silent background process from staying.

systemd drop-in support to configure resource limit individually.

Utilisation clamping to influence kernel Energy / Capacity Aware Scheduling and schedutil frequency scaling.


This is a rewrite of a rewrite of Portable!
- For the legacy Go version, see `legacy-go` branch.
- For the original Bash version, see `legacy` branch.

Portable has companion projects for packaging and sandboxing:
| Project       | Descripton      |
| ------------- | -------------   |
| [Init](https://github.com/Kraftland/portable-init) | Sandbox PID 1 supervisor |
| [StashPak](https://github.com/Kimiblock/stashpak) | Build a Portable package for Arch Linux |
| [Packer](https://github.com/Kimiblock/portable-packer) | Packaging utility for Portable |
| [Netsock](https://github.com/Kimiblock/netsock) | Network firewall |
| [bawn](https://github.com/Kimiblock/bawn) | CLI for transient sandbox |
| [Config](https://github.com/Kraftland/portable-config) | Configuration parser for Portable |

# System Requirements
- enables unprivileged user namespaces
- uses systemd >=258
- has libseccomp >= 2.6
- a thread-safe libudev implementation (systemd-udevd is)
- has landlock ABI 8 and above (Linux kernel >= 7.0)
- Follows the [FHS (Filesystem Hierarchy Standard)](https://specifications.freedesktop.org/fhs/latest/)
	- Note that `/lib` `/lib64` `/bin` `/sbin` should be symlinks to their respective locations under `/usr`

- Does not have mount points under /usr/bin, and use a supported fs of OverlayFS (NOT BcacheFS)

# Available for

- [Minecraft](https://github.com/Kimiblock/moeOS.config/blob/master/usr/bin/mcLaunch)
- Arch Linux
	- Arch Linux CN Repository
		- Only selected free/OSS apps
		- Updates faster
	- Portable for Arch
		- Configure your system to use [portable-arch](https://github.com/Kraftland/portable-arch): https://github.com/Kraftland/portable-arch
		- Current support status (as of 09 Sep 2026): 37 packages in repo.

# Limitations:

1. **Running untrusted code is never safe, sandboxing does not change this.**
2. On KDE Plasma window grouping may not work properly unless your desktop file name exactly matches certain arguments.
	- This is a [KWin issue](https://bugs.kde.org/show_bug.cgi?id=502309)
3. Portable acts like Flatpak, to trick XDG Desktop Portal.
	- Blocked until [systemd-appd](https://github.com/systemd/systemd/pull/43885) lands with Portal support.

<h1 align="center">
  <img src="https://raw.githubusercontent.com/Kraftland/portable/refs/heads/master/share/example.webp" alt="The Portable Project" width="1024" />
  <br>
  Demo
  <br>
</h1>


# How to package?

See [Docs](https://github.com/Kraftland/portable/tree/master/doc)

# FAQ / Troubleshooting
1. Portable fails with something like _invalid argument_
	- BcacheFS is not supported, or you have mountpoints under `/usr/bin` and `/usr/lib`

## Starting portable

Start portable with environment variable `PORTABLE_CONF`, which can be 1) the ID of the sandbox, 2) an absolute path (if exists), 3) a file name interpreted as `$(pwd)/${PORTABLE_CONF}`. It searches for each of them respectively.

- Debugging output can be enabled by building with debug assertions (debug builds).

### Debugging

#### Entering sandbox

To manually execute programs instead of following the `exec.target` config, start portable with argument `--actions debug-shell`. This will open a bash prompt and gives you full control of the sandbox environment.

#### D-Bus

When debug-shell is enabled on a debug build of Portable, D-Bus proxy will log to the standard output of primary instance.

# Code of Conduct

Portable and any of its social environment follows the [Kraftland Code of Conduct](https://blog.kimiblock.top/notice/#Code-of-Conduct). Please be sure not to violate such rule set.

# Version Scheme
Portable follows a major.minor.patch version scheme. We thrives to provide a stable experiences with no breaking changes, however, if said change is necessary, will land in a major release.

The patch release is exclusive for bug fixes. Whereas minor releases contain new features. If a feature or a set of features needs time to test or is important enough, we conduct a major release.

Portable has and always will be only supporting the latest release. Generally users can upgrade without manual intervention, but between major releases it's advised to run `systemctl --user stop portable.slice` to stop the portable framework.
