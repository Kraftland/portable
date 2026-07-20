# 20.0

## Breaking Changes:
- Removed deprecated configuration fields, including `privacy.camera`, `privacy.input`, `system.gameMode` and `system.virtualization`. Please migrate to the unified device allow array. [#1034](https://github.com/Kraftland/portable/pull/1034)
- Removed toggle for process tracking, it is now always enabled. [#1034](https://github.com/Kraftland/portable/pull/1034)
- Default configuration changes. [#1035](https://github.com/Kraftland/portable/pull/1035)
	- KDE status indicator	-> `false`
	- Classic Notifications	-> `false`
	- Network		-> `false`

## Improvements:
- Implemented overlay execution for D-Bus activation
- Made seccomp filter compilation asynchronous to avoid blocking
- Made seccomp unotify run in an exclusive thread
- Made seccomp list compiling run in a blocking thread to improve performance on single-core systems
- Made uclamp writing run in a blocking thread to improve performance
- Removed deprecated logic in process spawner to improve performance
- Fixed a possible situation where init may panic but continue running
- Properly handled channel send error in spawner
- Properly handled OpenPty errors

# 18.0 - Lawn

This release of Portable brings a prominent rewrite of the sandbox supervisor system, includes XDG activation protocol support, greatly reduces installed files, and much more. It sets a new bar for a secure and performant sandbox.

The new supervisor is carefully and thoughtfully engineered. It represents our vision for the future of sandboxing. It is responsive while feature-packed. In addition to existing security model from the Go version, we also introduced better system call filtering with allow-listing and custom return values for a smaller attack surface and better concealment of secure execution environment to malicious applications. The default error has also been changed from _Permision Denied_ to _ENOSYS_, which represents that kernel lacks support for said system call, allowing well-behaved applications to fall back gracefully.

Among a set of security features, there're also several quality of life changes for the Init and daemon. When launching multiple instances, Portable now streams the entire console rather than standard input, output and error, providing better integration and navigation for terminal applications. The supervisor will now reclaim expired files to avoid cluttering up shared directory, as well as offloading argument calculation to the daemon. The logging infrastructure has also been reworked to feature coloured output.

With all those new features, the average lifetime of supervisor went from 5.9 milliseconds to 5.4 milliseconds, bringing this release in line with the commitment of a fast, private, secure sandbox, for the Linux desktop.

## 18.rc

### Init
* seccomp: introduce a list of syscalls to fake status by @Kimiblock in https://github.com/Kraftland/portable-init/pull/32
* seccomp: expand the built-in list of syscall by @Kimiblock in https://github.com/Kraftland/portable-init/pull/33
* seccomp: deny keyring syscalls by default by @Kimiblock in https://github.com/Kraftland/portable-init/pull/34
* seccomp: allow mincore syscall by @Kimiblock in https://github.com/Kraftland/portable-init/pull/35
* seccomp: reply with ENOSYS to gracefully fallback application requests by @Kimiblock in https://github.com/Kraftland/portable-init/pull/36
* seccomp: use negative errors by @Kimiblock in https://github.com/Kraftland/portable-init/pull/37
* seccomp: add comments for mincore by @Kimiblock in https://github.com/Kraftland/portable-init/pull/38

## 18.beta
### Daemon
* introduce NEWS entry by @Kimiblock in https://github.com/Kraftland/portable/pull/1018
* next: remove advanced.landlock config key by @Kimiblock in https://github.com/Kraftland/portable/pull/1019
* multi-instance: disable append mode because extra arguments apply whi… by @Kimiblock in https://github.com/Kraftland/portable/pull/1022
* build(deps): bump golang.org/x/sys from 0.46.0 to 0.47.0 in /lib/daemon by @dependabot[bot] in https://github.com/Kraftland/portable/pull/1021
* init: bump to latest commit by @Kimiblock in https://github.com/Kraftland/portable/pull/1023

### Init:
* logger: call unwrap by @Kimiblock in https://github.com/Kraftland/portable-init/pull/23
* ipc: create shared directory only when missing by @Kimiblock in https://github.com/Kraftland/portable-init/pull/24
* ipc: resolve Portal responses using URL encoding by @Kimiblock in https://github.com/Kraftland/portable-init/pull/25
* next: implement idle inhibit by @Kimiblock in https://github.com/Kraftland/portable-init/pull/26
* seccomp: actually make use of cancel token, don't print mystic errors… by @Kimiblock in https://github.com/Kraftland/portable-init/pull/27
* feat: remove expired shared files automatically by @Kimiblock in https://github.com/Kraftland/portable-init/pull/28
* envs: while resolving commandline arguments, start loop with 1 or more items left by @Kimiblock in https://github.com/Kraftland/portable-init/pull/29
* seccomp: reply 0 to capset calls by @Kimiblock in https://github.com/Kraftland/portable-init/pull/30

## 18.alpha
### Daemon
* next: prepare for submodules by @Kimiblock in https://github.com/Kraftland/portable/pull/1002
* daemon: signal helper about debugging status by @Kimiblock in https://github.com/Kraftland/portable/pull/1007
* specify seccomp >= 2.6 requirement by @Kimiblock in https://github.com/Kraftland/portable/pull/1008
* daemon: pass debugging var to init by @Kimiblock in https://github.com/Kraftland/portable/pull/1009
* readConf: treat undefined flatpakInfo as true by @Kimiblock in https://github.com/Kraftland/portable/pull/1010
* daemon: pass runtime dir by @Kimiblock in https://github.com/Kraftland/portable/pull/1013
* epic: rewrite helper by @Kimiblock in https://github.com/Kraftland/portable/pull/1014
* rework packaging by @Kimiblock in https://github.com/Kraftland/portable/pull/1016
* init: checkout at alpha 2 by @Kimiblock in https://github.com/Kraftland/portable/pull/1017

### Changes from Init
* initial seccomp filtering by @Kimiblock in https://github.com/Kraftland/portable-init/pull/1
* add actions by @Kimiblock in https://github.com/Kraftland/portable-init/pull/2
* next: decode has_info by @Kimiblock in https://github.com/Kraftland/portable-init/pull/3
* next: landlock by @Kimiblock in https://github.com/Kraftland/portable-init/pull/4
* next: implement uclamp settings by @Kimiblock in https://github.com/Kraftland/portable-init/pull/6
* next: implement PID counter by @Kimiblock in https://github.com/Kraftland/portable-init/pull/7
* connect to session bus by @Kimiblock in https://github.com/Kraftland/portable-init/pull/8
* envs: load sandbox ID by @Kimiblock in https://github.com/Kraftland/portable-init/pull/9
* next: implement parser for _portableHelperExtraFiles by @Kimiblock in https://github.com/Kraftland/portable-init/pull/10
* actually parse pass files by @Kimiblock in https://github.com/Kraftland/portable-init/pull/11
* next: Graceful Shutdown by @Kimiblock in https://github.com/Kraftland/portable-init/pull/12
* next: preliminary IPC by @Kimiblock in https://github.com/Kraftland/portable-init/pull/13
* next: cmdline rewrite by @Kimiblock in https://github.com/Kraftland/portable-init/pull/14
* next: implement request fs access by @Kimiblock in https://github.com/Kraftland/portable-init/pull/15
* ipc: create shared directory first by @Kimiblock in https://github.com/Kraftland/portable-init/pull/17
* next: spawner logic by @Kimiblock in https://github.com/Kraftland/portable-init/pull/16
* landlock: compile rules first, then load rules right before spawning by @Kimiblock in https://github.com/Kraftland/portable-init/pull/18
* drop all OsString ref by @Kimiblock in https://github.com/Kraftland/portable-init/pull/19
* envs: don't add unknown files into map by @Kimiblock in https://github.com/Kraftland/portable-init/pull/20
* next: proper unotify impl by @Kimiblock in https://github.com/Kraftland/portable-init/pull/21
* next: drop async logging by @Kimiblock in https://github.com/Kraftland/portable-init/pull/22
* logger: call unwrap by @Kimiblock in https://github.com/Kraftland/portable-init/pull/23
* ipc: create shared directory only when missing by @Kimiblock in https://github.com/Kraftland/portable-init/pull/24
* ipc: resolve Portal responses using URL encoding by @Kimiblock in https://github.com/Kraftland/portable-init/pull/25

**Full Changelog**: https://github.com/Kraftland/portable-init/commits/18.0.alpha.2
