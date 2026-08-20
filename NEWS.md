# 20.0

## Breaking Changes:
- Removed deprecated configuration fields, including `privacy.camera`, `privacy.input`, `system.gameMode` and `system.virtualization`. Please migrate to the unified device allow array. [#1034](https://github.com/Kraftland/portable/pull/1034)
- Removed toggle for process tracking, it is now always enabled. [#1034](https://github.com/Kraftland/portable/pull/1034)
- Default configuration changes. [#1035](https://github.com/Kraftland/portable/pull/1035)
	- KDE status indicator	-> `false`
	- Classic Notifications	-> `false`
	- Network		-> `false`

## Notable non-breaking changes:
- Display protocol:
	- The native display protocol will always be enabled
	- Wayland socket is now mounted at `/run/wayland`

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
- The `io_uring_setup` system call is now allowed
- The primary GPU is now determined with boot display attribute value, rather than connector status. Allowing multi-GPU multi-head system to operate optimally ([#1072](https://github.com/Kraftland/portable/pull/1072))
- It is now less likely for Init to be stuck when running for a few days
- Non-native system calls will now cause the thread to be killed, rather than returning ENOSYS silently.
- The D-Bus proxy has been more properly restricted from other local processes.
- Portal responses has been locked down more appropriately.
- Fixed an issue with D-Bus filtering where legacy notification action would not trigger anything.
- The PulseAudio server is being parsed rather than assuming default path
- The PulseAudio server is activated before starting sandbox
- Portable now utilises the wp-security-context-v1 protocol to provide a secure way of window identification
- Secondary instances are now streaming to the console via PTY, and will automatically re-scale their terminal size
- It is now possible to disable certain components of Portable by subsystem.
- Fixed a bug causing secondary instances to stuck while calling prohibited system calls.
- Each instance now has a new session. Console is given as controlling terminal. Fixes various shell implementations not launching.
- When starting the Primary and secondary instance at once, it is now much less likely to drop AuxStart message due to Init starting slow.
- Warns about outdated version of Init
- When starting Portable without a valid terminal, it will no longer allocate new pairs of pesudo-terminal
- When using a supported desktop environment, Portable will display the sandbox status in _Background Apps_ area.

## Internal Changes

### 20.alpha

#### Daemon
* init crate by @Kimiblock in https://github.com/Kraftland/portable/pull/1033
* implement configuration deserializer by @Kimiblock in https://github.com/Kraftland/portable/pull/1034
* implement default values for configuration by @Kimiblock in https://github.com/Kraftland/portable/pull/1035
* config: implement outside struct Config by @Kimiblock in https://github.com/Kraftland/portable/pull/1036
* daemon: scale StatusNotifier with CPU core count by @Kimiblock in https://github.com/Kraftland/portable/pull/1038
* next: cargo workspaces by @Kimiblock in https://github.com/Kraftland/portable/pull/1039
* build(deps): bump golang.org/x/term from 0.44.0 to 0.45.0 in /lib/daemon by @dependabot[bot] in https://github.com/Kraftland/portable/pull/1020
* bump NEWS by @Kimiblock in https://github.com/Kraftland/portable/pull/1041
* include legacy_conf in deps by @Kimiblock in https://github.com/Kraftland/portable/pull/1042
* update legacy config by @Kimiblock in https://github.com/Kraftland/portable/pull/1043
* next: init logger + stop worker + legacy fun deserialization by @Kimiblock in https://github.com/Kraftland/portable/pull/1044
* logger: implement terminal detection and no colour pref by @Kimiblock in https://github.com/Kraftland/portable/pull/1045
* logger: implement console restore by @Kimiblock in https://github.com/Kraftland/portable/pull/1046
* stop: await from future by @Kimiblock in https://github.com/Kraftland/portable/pull/1047
* logger: get colour status + listen on SIGINT by @Kimiblock in https://github.com/Kraftland/portable/pull/1048
* stop: listen for SIGTERM as well by @Kimiblock in https://github.com/Kraftland/portable/pull/1049
* logger: implement colour detection by @Kimiblock in https://github.com/Kraftland/portable/pull/1050
* logger: work together with stop() by @Kimiblock in https://github.com/Kraftland/portable/pull/1051
* make use of logging daemon by @Kimiblock in https://github.com/Kraftland/portable/pull/1052
* next: read TOML config by @Kimiblock in https://github.com/Kraftland/portable/pull/1053
* epic: read config using new modules by @Kimiblock in https://github.com/Kraftland/portable/pull/1054
* xdg: define struct and implement runtime() call by @Kimiblock in https://github.com/Kraftland/portable/pull/1056
* xdg: implement home and config_home by @Kimiblock in https://github.com/Kraftland/portable/pull/1057
* xdg: publish runtime and home fields by @Kimiblock in https://github.com/Kraftland/portable/pull/1058
* xdg: implement get() for XdgDirs by @Kimiblock in https://github.com/Kraftland/portable/pull/1059
* logger: align tabs by @Kimiblock in https://github.com/Kraftland/portable/pull/1060
* spawn XDG base directories by @Kimiblock in https://github.com/Kraftland/portable/pull/1061
* epic: implement the udev bits by @Kimiblock in https://github.com/Kraftland/portable/pull/1062
* restructure devices module by @Kimiblock in https://github.com/Kraftland/portable/pull/1063
* types: abstract bind type by @Kimiblock in https://github.com/Kraftland/portable/pull/1064
* next: implement path translation by @Kimiblock in https://github.com/Kraftland/portable/pull/1065
* xdg: implement DATA_HOME by @Kimiblock in https://github.com/Kraftland/portable/pull/1066
* translator: implement a better translation approach by @Kimiblock in https://github.com/Kraftland/portable/pull/1068
* translator: implement async trait for PathBuf by @Kimiblock in https://github.com/Kraftland/portable/pull/1069
* types: implement deduplication trait for BindRules by @Kimiblock in https://github.com/Kraftland/portable/pull/1070
* epic: implement device enumerator and boot display detection by @Kimiblock in https://github.com/Kraftland/portable/pull/1072
* bump init submodule by @Kimiblock in https://github.com/Kraftland/portable/pull/1073
* GPU: implement card and renderer association by @Kimiblock in https://github.com/Kraftland/portable/pull/1074
* devices: implement subsystem with devtype filtering by @Kimiblock in https://github.com/Kraftland/portable/pull/1075
* gpu: implement enumerate GPUs function by @Kimiblock in https://github.com/Kraftland/portable/pull/1076
* move module definitions into lib.rs by @Kimiblock in https://github.com/Kraftland/portable/pull/1077
* gpu: implement vendor detection by @Kimiblock in https://github.com/Kraftland/portable/pull/1078
* gpu: implement /dev nvidia discovery by @Kimiblock in https://github.com/Kraftland/portable/pull/1079
* next: include GPU test by @Kimiblock in https://github.com/Kraftland/portable/pull/1080
* types: introduce tmpfs mount by @Kimiblock in https://github.com/Kraftland/portable/pull/1081
* gpu: support drm nodes under /sys by @Kimiblock in https://github.com/Kraftland/portable/pull/1082
* gpu: handle errors gracefully by @Kimiblock in https://github.com/Kraftland/portable/pull/1083
* devices: implement bind_udev_device by @Kimiblock in https://github.com/Kraftland/portable/pull/1084
* gpu: implement active gpus detection by @Kimiblock in https://github.com/Kraftland/portable/pull/1085
* initial mockup of environment holder by @Kimiblock in https://github.com/Kraftland/portable/pull/1086
* logger: create type alias for log channel by @Kimiblock in https://github.com/Kraftland/portable/pull/1087
* GPU: implement NVIDIA bits by @Kimiblock in https://github.com/Kraftland/portable/pull/1088
* move config_* into submodule by @Kimiblock in https://github.com/Kraftland/portable/pull/1089
* config: introduce async exists helper by @Kimiblock in https://github.com/Kraftland/portable/pull/1090
* config: implement path finder for TOML config by @Kimiblock in https://github.com/Kraftland/portable/pull/1091
* config: implement default traits for config by @Kimiblock in https://github.com/Kraftland/portable/pull/1092
* refactor: wrap logic in run() to have a nice error message by @Kimiblock in https://github.com/Kraftland/portable/pull/1093
* main: remove unused wait by @Kimiblock in https://github.com/Kraftland/portable/pull/1094
* config: stop guessing TOML path by @Kimiblock in https://github.com/Kraftland/portable/pull/1095
* config: migrate bash config to new path finder by @Kimiblock in https://github.com/Kraftland/portable/pull/1096
* ipc: fix init and introduce Info interface by @Kimiblock in https://github.com/Kraftland/portable/pull/1097
* ipc: provide interfaces by @Kimiblock in https://github.com/Kraftland/portable/pull/1098
* register bus connection on the main function by @Kimiblock in https://github.com/Kraftland/portable/pull/1099
* main: hint about aux mode by @Kimiblock in https://github.com/Kraftland/portable/pull/1100
* gate systemd feature behind default by @Kimiblock in https://github.com/Kraftland/portable/pull/1101
* spawn: implement instance ID collision detector for flatpak by @Kimiblock in https://github.com/Kraftland/portable/pull/1102
* the instance_id u32 does not need to be mutable by @Kimiblock in https://github.com/Kraftland/portable/pull/1103
* console: implement new() by @Kimiblock in https://github.com/Kraftland/portable/pull/1104
* cargo: require zbus-systemd by @Kimiblock in https://github.com/Kraftland/portable/pull/1105
* start_transient: port some properties over by @Kimiblock in https://github.com/Kraftland/portable/pull/1106
* start_transient: port notify access by @Kimiblock in https://github.com/Kraftland/portable/pull/1107
* port NoNewPrivileges over by @Kimiblock in https://github.com/Kraftland/portable/pull/1108
* port KillMode over by @Kimiblock in https://github.com/Kraftland/portable/pull/1109
* port IPAccounting over by @Kimiblock in https://github.com/Kraftland/portable/pull/1110
* port the rest of props over by @Kimiblock in https://github.com/Kraftland/portable/pull/1111
* types: define type for exporting file descriptor to downstream by @Kimiblock in https://github.com/Kraftland/portable/pull/1112
* wire ExecStartEx to use no-setuid by @Kimiblock in https://github.com/Kraftland/portable/pull/1113
* typo fix by @Kimiblock in https://github.com/Kraftland/portable/pull/1114
* next: drop FD, pass $HOME by @Kimiblock in https://github.com/Kraftland/portable/pull/1115
* BindType: expose OverlayFS mounting and implement to_cmdline by @Kimiblock in https://github.com/Kraftland/portable/pull/1116
* types: implement VFS type devtmpfs by @Kimiblock in https://github.com/Kraftland/portable/pull/1117
* types: implement VFS type procfs by @Kimiblock in https://github.com/Kraftland/portable/pull/1118
* types: implement tmpfs with mode and size via VFS enum by @Kimiblock in https://github.com/Kraftland/portable/pull/1119
* implement mqueue mounting by @Kimiblock in https://github.com/Kraftland/portable/pull/1120
* create D-Bus access abstraction by @Kimiblock in https://github.com/Kraftland/portable/pull/1121
* types: map BindRules with type alias by @Kimiblock in https://github.com/Kraftland/portable/pull/1122
* epic: port and enhance bus sandboxing by @Kimiblock in https://github.com/Kraftland/portable/pull/1123
* bus: wire up bus rule generation to new() by @Kimiblock in https://github.com/Kraftland/portable/pull/1125
* bus: implement the Start method by @Kimiblock in https://github.com/Kraftland/portable/pull/1126
* audio: implement server address parsing and activation by @Kimiblock in https://github.com/Kraftland/portable/pull/1127
* devices: deduplicate rules by default by @Kimiblock in https://github.com/Kraftland/portable/pull/1128
* camera: implement scan by @Kimiblock in https://github.com/Kraftland/portable/pull/1129
* display: introduce skeleton by @Kimiblock in https://github.com/Kraftland/portable/pull/1130
* suppress unused result by @Kimiblock in https://github.com/Kraftland/portable/pull/1131
* don't use async in public traits by @Kimiblock in https://github.com/Kraftland/portable/pull/1132
* bind: implement display protocol abstraction by @Kimiblock in https://github.com/Kraftland/portable/pull/1133
* x11: implement xauth detection by @Kimiblock in https://github.com/Kraftland/portable/pull/1134
* display enablement: X11 by @Kimiblock in https://github.com/Kraftland/portable/pull/1135
* display: Wayland and session type support by @Kimiblock in https://github.com/Kraftland/portable/pull/1136
* introduce Runtime Paths by @Kimiblock in https://github.com/Kraftland/portable/pull/1138
* wayland: security context v1 by @Kimiblock in https://github.com/Kraftland/portable/pull/1139
* display: make use of Wayland security context by @Kimiblock in https://github.com/Kraftland/portable/pull/1140
* epic: abstract bind generation into subsystems by @Kimiblock in https://github.com/Kraftland/portable/pull/1141
* implement Input device subsystem by @Kimiblock in https://github.com/Kraftland/portable/pull/1142
* xdg: implement data_dirs by @Kimiblock in https://github.com/Kraftland/portable/pull/1143
* subsystems: implement desktop file writing by @Kimiblock in https://github.com/Kraftland/portable/pull/1144
* stop: use spawn_blocking since they are not async by @Kimiblock in https://github.com/Kraftland/portable/pull/1145
* dirs: implement leftover removal by @Kimiblock in https://github.com/Kraftland/portable/pull/1146
* dirs: implement Flatpak dirs by @Kimiblock in https://github.com/Kraftland/portable/pull/1147
* bus: change the trait to implement Proxy by @Kimiblock in https://github.com/Kraftland/portable/pull/1148
* bus: implement at-spi accessibility proxy by @Kimiblock in https://github.com/Kraftland/portable/pull/1149
* subsystem: mask by @Kimiblock in https://github.com/Kraftland/portable/pull/1150
* move config into the pref subsystem by @Kimiblock in https://github.com/Kraftland/portable/pull/1151
* console: return slave's pts name by @Kimiblock in https://github.com/Kraftland/portable/pull/1152
* spawn: attach pty to the process by @Kimiblock in https://github.com/Kraftland/portable/pull/1153
* runtime subsystem: define types by @Kimiblock in https://github.com/Kraftland/portable/pull/1154
* epic: Cmdline parsing by @Kimiblock in https://github.com/Kraftland/portable/pull/1155
* ipc: Notifications by @Kimiblock in https://github.com/Kraftland/portable/pull/1156
* share_file: implement helper alive ping by @Kimiblock in https://github.com/Kraftland/portable/pull/1157
* portals: export legacy_notif by @Kimiblock in https://github.com/Kraftland/portable/pull/1158
* share_file: implement warning systems of Init presence by @Kimiblock in https://github.com/Kraftland/portable/pull/1159
* runtime: implement file sharing logic by @Kimiblock in https://github.com/Kraftland/portable/pull/1160
* ipc: return connection on secondary by @Kimiblock in https://github.com/Kraftland/portable/pull/1161
* ipc: split up register and connect function by @Kimiblock in https://github.com/Kraftland/portable/pull/1162
* ipc: implement stop app by @Kimiblock in https://github.com/Kraftland/portable/pull/1163
* main: handle Quit cmdline by @Kimiblock in https://github.com/Kraftland/portable/pull/1164
* ipc: define info to pass down via Bus IPC by @Kimiblock in https://github.com/Kraftland/portable/pull/1165
* pref: drop show stats by @Kimiblock in https://github.com/Kraftland/portable/pull/1166
* info: also add in uclamp by @Kimiblock in https://github.com/Kraftland/portable/pull/1167
* dirs: implement DocumentPortal path by @Kimiblock in https://github.com/Kraftland/portable/pull/1168
* user subsystem: bind home by @Kimiblock in https://github.com/Kraftland/portable/pull/1169
* documnts: publish per-app instance by @Kimiblock in https://github.com/Kraftland/portable/pull/1170
* subsystems: introduce system binding by @Kimiblock in https://github.com/Kraftland/portable/pull/1171
* bind: import the system subsystem by @Kimiblock in https://github.com/Kraftland/portable/pull/1172
* bind: make use of the system bind subsystem by @Kimiblock in https://github.com/Kraftland/portable/pull/1173
* subsystems: bind devices by @Kimiblock in https://github.com/Kraftland/portable/pull/1174
* bind: activate the display subsystem by @Kimiblock in https://github.com/Kraftland/portable/pull/1175
* dirs: portable runtime: implement shallow clone via std::sync::Arc by @Kimiblock in https://github.com/Kraftland/portable/pull/1177
* bind: convert runtime_dir to Arc pointer by @Kimiblock in https://github.com/Kraftland/portable/pull/1178
* subsystems: enable masking by @Kimiblock in https://github.com/Kraftland/portable/pull/1179
* translate: use atomic arc reference by @Kimiblock in https://github.com/Kraftland/portable/pull/1180
* subsystem: bring up by @Kimiblock in https://github.com/Kraftland/portable/pull/1181
* subsystems: handle debug-shell request by @Kimiblock in https://github.com/Kraftland/portable/pull/1182
* dirs: use Arc pointers by @Kimiblock in https://github.com/Kraftland/portable/pull/1183
* main: make use of the bind subsystem by @Kimiblock in https://github.com/Kraftland/portable/pull/1184
* main: bring up the D-Bus proxy by @Kimiblock in https://github.com/Kraftland/portable/pull/1185
* publish init info by @Kimiblock in https://github.com/Kraftland/portable/pull/1186
* ipc: strip trailing NUL byte from Document Portal by @Kimiblock in https://github.com/Kraftland/portable/pull/1187
* gpu: handle NVIDIA driver name properly by @Kimiblock in https://github.com/Kraftland/portable/pull/1188
* bus: fixes by @Kimiblock in https://github.com/Kraftland/portable/pull/1189
* spawn: build cmdline by @Kimiblock in https://github.com/Kraftland/portable/pull/1190
* spawn: implement console streaming by @Kimiblock in https://github.com/Kraftland/portable/pull/1191
* bring up the main system by @Kimiblock in https://github.com/Kraftland/portable/pull/1192
* rework the GPU binding system by @Kimiblock in https://github.com/Kraftland/portable/pull/1193
* rework the console streaming infrastructure by @Kimiblock in https://github.com/Kraftland/portable/pull/1194
* envs: forward XDG_ACTIVATION_TOKEN by @Kimiblock in https://github.com/Kraftland/portable/pull/1195
* D-Bus: introduce auxiliary mode by @Kimiblock in https://github.com/Kraftland/portable/pull/1196
* Next: bug fixes by @Kimiblock in https://github.com/Kraftland/portable/pull/1197
* devices subsystem: fix devlinks handling by @Kimiblock in https://github.com/Kraftland/portable/pull/1199
* epic: Swap out the Go version for Rust Portable by @Kimiblock in https://github.com/Kraftland/portable/pull/1198
* GPU: zink by @Kimiblock in https://github.com/Kraftland/portable/pull/1200
* GPU: bind the parent device by @Kimiblock in https://github.com/Kraftland/portable/pull/1201
* bus: mount a compatibility D-Bus socket at /run/sessionBus by @Kimiblock in https://github.com/Kraftland/portable/pull/1202
* ipc: drop legacy Info endpoint by @Kimiblock in https://github.com/Kraftland/portable/pull/1203
* unify environment forwarding by @Kimiblock in https://github.com/Kraftland/portable/pull/1204
* console: buffer fix by @Kimiblock in https://github.com/Kraftland/portable/pull/1205
* tray fixes by @Kimiblock in https://github.com/Kraftland/portable/pull/1206
* cmdline: only share network namespace when network is allowed by @Kimiblock in https://github.com/Kraftland/portable/pull/1208
* bump init by @Kimiblock in https://github.com/Kraftland/portable/pull/1209
* quit: fix return type of the bus name by @Kimiblock in https://github.com/Kraftland/portable/pull/1210
* bump init by @Kimiblock in https://github.com/Kraftland/portable/pull/1211
* stop: migrate to async execution and don't rely on thread block by @Kimiblock in https://github.com/Kraftland/portable/pull/1212
* subsystem: introduce console for terminal detection by @Kimiblock in https://github.com/Kraftland/portable/pull/1213
* bump init by @Kimiblock in https://github.com/Kraftland/portable/pull/1214
* ipc: support silent start by @Kimiblock in https://github.com/Kraftland/portable/pull/1215
* terminate service after exit by @Kimiblock in https://github.com/Kraftland/portable/pull/1216
* try to unbust security context by @Kimiblock in https://github.com/Kraftland/portable/pull/1217
* feat: Background Portal status by @Kimiblock in https://github.com/Kraftland/portable/pull/1218
* update news by @Kimiblock in https://github.com/Kraftland/portable/pull/1219
* documents: implement the List call by @Kimiblock in https://github.com/Kraftland/portable/pull/1220
* documents: expose the list function by @Kimiblock in https://github.com/Kraftland/portable/pull/1221
* documents: support the Delete action by @Kimiblock in https://github.com/Kraftland/portable/pull/1222
* main: make use of permission reset by @Kimiblock in https://github.com/Kraftland/portable/pull/1223
* logger: use biased tokio selection to always process log message first by @Kimiblock in https://github.com/Kraftland/portable/pull/1224
* cmdline: support opening sandbox home by @Kimiblock in https://github.com/Kraftland/portable/pull/1225


**Full Changelog**: https://github.com/Kraftland/portable/compare/18.0.1...20.alpha

#### Legacy configuration deserialiser
* de: remove unused imports by @Kimiblock in https://github.com/Kimiblock/portable-legacyconf/pull/1
* define config and deserializer for commands by @Kimiblock in https://github.com/Kimiblock/portable-legacyconf/pull/2
* define a legacy config struct by @Kimiblock in https://github.com/Kimiblock/portable-legacyconf/pull/3
* define complete config types by @Kimiblock in https://github.com/Kimiblock/portable-legacyconf/pull/4
* def: publish fields by @Kimiblock in https://github.com/Kimiblock/portable-legacyconf/pull/5
* config: deserialize Wayland by @Kimiblock in https://github.com/Kimiblock/portable-legacyconf/pull/6
* def: make several configurations default to true by @Kimiblock in https://github.com/Kimiblock/portable-legacyconf/pull/7
* test: decode standard Komikku config by @Kimiblock in https://github.com/Kimiblock/portable-legacyconf/pull/8
* lib: export config public by @Kimiblock in https://github.com/Kimiblock/portable-legacyconf/pull/9
* config_def: define default values for keys by @Kimiblock in https://github.com/Kimiblock/portable-legacyconf/pull/10
* update cargo info by @Kimiblock in https://github.com/Kimiblock/portable-legacyconf/pull/11
* specify license by @Kimiblock in https://github.com/Kimiblock/portable-legacyconf/pull/12

#### Init
* seccomp: run in a native OS thread because seccomp unotify is blocking by @Kimiblock in https://github.com/Kraftland/portable-init/pull/39
* envs: simplify lockdown env parsing by @Kimiblock in https://github.com/Kraftland/portable-init/pull/40
* seccomp: make filter compilation async by @Kimiblock in https://github.com/Kraftland/portable-init/pull/41
* spawn: log fatally first, then panic by @Kimiblock in https://github.com/Kraftland/portable-init/pull/42
* seccomp: spawn blocking thread for list compilation by @Kimiblock in https://github.com/Kraftland/portable-init/pull/43
* uclamp: spawn a blocking thread for uclamp writing because it's IO in… by @Kimiblock in https://github.com/Kraftland/portable-init/pull/44
* spawn: rip out unused streaming directory logic by @Kimiblock in https://github.com/Kraftland/portable-init/pull/45
* spawn: halt execution on landlock rules failure by @Kimiblock in https://github.com/Kraftland/portable-init/pull/46
* spawn: properly handle counter error by @Kimiblock in https://github.com/Kraftland/portable-init/pull/47
* spawn: properly handle openpty errors by @Kimiblock in https://github.com/Kraftland/portable-init/pull/48
* seccomp: allow io_uring_setup by @Kimiblock in https://github.com/Kraftland/portable-init/pull/50
* cargo: use async features by @Kimiblock in https://github.com/Kraftland/portable-init/pull/51
* seccomp: specify bad arch action to kill thread, and handle magic sys… by @Kimiblock in https://github.com/Kraftland/portable-init/pull/52
* seccomp: allow io_uring_enter as async_io syscall by @Kimiblock in https://github.com/Kraftland/portable-init/pull/53
* envs: get appID via cmdline, and others via D-Bus IPC by @Kimiblock in https://github.com/Kraftland/portable-init/pull/55
* console: rework by @Kimiblock in https://github.com/Kraftland/portable-init/pull/56
* ipc: support activating tray by @Kimiblock in https://github.com/Kraftland/portable-init/pull/57
* seccomp: load filter to the whole process by @Kimiblock in https://github.com/Kraftland/portable-init/pull/58
* tray fixes by @Kimiblock in https://github.com/Kraftland/portable-init/pull/59
* spawn: use pre_exec to handle console streaming by @Kimiblock in https://github.com/Kraftland/portable-init/pull/60
* spawn: remove unused attempt by @Kimiblock in https://github.com/Kraftland/portable-init/pull/61
* update dependencies by @Kimiblock in https://github.com/Kraftland/portable-init/pull/62
* apply landlock to all threads using ABI v8 by @Kimiblock in https://github.com/Kraftland/portable-init/pull/63
* spawn: dynamically detect if current master instance is direct or pty by @Kimiblock in https://github.com/Kraftland/portable-init/pull/64
* ipc: report cargo ver by @Kimiblock in https://github.com/Kraftland/portable-init/pull/65
* ipc: advertise silent start by @Kimiblock in https://github.com/Kraftland/portable-init/pull/66
* counter: Unify notification sender and add Portal Background Status support by @Kimiblock in https://github.com/Kraftland/portable-init/pull/67


**Full Changelog**: https://github.com/Kraftland/portable-init/compare/18.0...20.alpha

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
