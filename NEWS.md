#18.rc

## Init
* seccomp: introduce a list of syscalls to fake status by @Kimiblock in https://github.com/Kraftland/portable-init/pull/32
* seccomp: expand the built-in list of syscall by @Kimiblock in https://github.com/Kraftland/portable-init/pull/33
* seccomp: deny keyring syscalls by default by @Kimiblock in https://github.com/Kraftland/portable-init/pull/34

# 18.beta
## Daemon
* introduce NEWS entry by @Kimiblock in https://github.com/Kraftland/portable/pull/1018
* next: remove advanced.landlock config key by @Kimiblock in https://github.com/Kraftland/portable/pull/1019
* multi-instance: disable append mode because extra arguments apply whi… by @Kimiblock in https://github.com/Kraftland/portable/pull/1022
* build(deps): bump golang.org/x/sys from 0.46.0 to 0.47.0 in /lib/daemon by @dependabot[bot] in https://github.com/Kraftland/portable/pull/1021
* init: bump to latest commit by @Kimiblock in https://github.com/Kraftland/portable/pull/1023

## Init:
* logger: call unwrap by @Kimiblock in https://github.com/Kraftland/portable-init/pull/23
* ipc: create shared directory only when missing by @Kimiblock in https://github.com/Kraftland/portable-init/pull/24
* ipc: resolve Portal responses using URL encoding by @Kimiblock in https://github.com/Kraftland/portable-init/pull/25
* next: implement idle inhibit by @Kimiblock in https://github.com/Kraftland/portable-init/pull/26
* seccomp: actually make use of cancel token, don't print mystic errors… by @Kimiblock in https://github.com/Kraftland/portable-init/pull/27
* feat: remove expired shared files automatically by @Kimiblock in https://github.com/Kraftland/portable-init/pull/28
* envs: while resolving commandline arguments, start loop with 1 or more items left by @Kimiblock in https://github.com/Kraftland/portable-init/pull/29
* seccomp: reply 0 to capset calls by @Kimiblock in https://github.com/Kraftland/portable-init/pull/30

# 18.alpha
## Daemon
* next: prepare for submodules by @Kimiblock in https://github.com/Kraftland/portable/pull/1002
* daemon: signal helper about debugging status by @Kimiblock in https://github.com/Kraftland/portable/pull/1007
* specify seccomp >= 2.6 requirement by @Kimiblock in https://github.com/Kraftland/portable/pull/1008
* daemon: pass debugging var to init by @Kimiblock in https://github.com/Kraftland/portable/pull/1009
* readConf: treat undefined flatpakInfo as true by @Kimiblock in https://github.com/Kraftland/portable/pull/1010
* daemon: pass runtime dir by @Kimiblock in https://github.com/Kraftland/portable/pull/1013
* epic: rewrite helper by @Kimiblock in https://github.com/Kraftland/portable/pull/1014
* rework packaging by @Kimiblock in https://github.com/Kraftland/portable/pull/1016
* init: checkout at alpha 2 by @Kimiblock in https://github.com/Kraftland/portable/pull/1017


**Full Changelog**: https://github.com/Kraftland/portable/compare/17.0.3...18.alpha

## Changes from Init
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