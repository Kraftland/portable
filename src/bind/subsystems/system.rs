mod passwd;
mod nsswitch;
mod kvm;
mod resolv;
mod bin;
mod machine_id;

/**
	The system bind subsystem
*/
pub struct SystemBind {
	pub config:		std::sync::Arc<crate::config::config_definition::Config>,

	pub xdg:		std::sync::Arc<crate::xdg::XdgDirs>,

	pub portable_runtime:	std::sync::Arc<crate::bind::subsystems::dirs::portable_runtime::PortableRuntime>,
	pub document_mount:	crate::bind::subsystems::dirs::documents::DocumentsMountPoint,

	pub flatpak_info:	std::sync::Arc<std::path::PathBuf>,

	pub logger:			crate::logger::LogSender,
}

impl super::GenerateBind for SystemBind {
	async fn bind(self) -> Result<crate::bind::types::BindRules, Self::BindError> {
		bind(
			self.config,
			self.portable_runtime,
			self.document_mount,
			self.xdg,
			self.flatpak_info,
			self.logger,
		).await
	}
	type BindError = SystemBindError;
}

/**
	The system bind subsystem exposes several paths of system root.

	It is designed to provide theming consistency in mind. Masking is done via the mask subsystem.
*/
async fn bind(
	config:			std::sync::Arc<crate::config::config_definition::Config>,
	portable_runtime:	std::sync::Arc<crate::bind::subsystems::dirs::portable_runtime::PortableRuntime>,
	document_mount:		crate::bind::subsystems::dirs::documents::DocumentsMountPoint,
	xdg:			std::sync::Arc<crate::xdg::XdgDirs>,
	flatpak_info:		std::sync::Arc<std::path::PathBuf>,
	logger:			crate::logger::LogSender,
)
-> Result<crate::bind::types::BindRules, SystemBindError>
{
	let state_directory = {
		let mut path = xdg.data_home.to_path_buf();
		path.push(&config.metadata.state_directory);
		path
	};

	let machine_id = tokio::spawn(
		machine_id::bind(
			config.clone(),
			xdg.clone(),
		),
	);

	let kvm_spawn = tokio::spawn(
		kvm::mount_kvm(config.system.device_allow.clone())
	);

	let bin_spawn = tokio::spawn(
		bin::bind(config.clone())
	);

	let resolv_spawn = tokio::spawn(resolv::mount(logger.clone()));

	let passwd_spawn = tokio::spawn(
		passwd::generate(
			portable_runtime.path(),
			state_directory,
		)
	);
	let nsswitch_spawn = tokio::spawn(
		nsswitch::generate(portable_runtime.path())
	);

	use crate::bind::subsystems::dirs::RuntimePathsTrait;
	use crate::bind::types::BindRule;
	let mut ret = vec![
		/*
			/etc mounts
		*/
		BindRule::Path {
			source:	"/etc".into(),
			dest:	"/etc".into(),
			class:	crate::bind::types::BindType::ReadOnly,
		},

		BindRule::VirtualFS {
			dest:	"/host".into(),
			class:	crate::bind::types::VirtualFS::Tmpfs {
				size_mb:	Some(0),
				perms:		None,
			},
		},
		BindRule::Path {
			source:	"/opt".into(),
			dest:	"/opt".into(),
			class:	crate::bind::types::BindType::ReadOnly,
		},
		BindRule::Path {
			source:	"/usr".into(),
			dest:	"/usr".into(),
			class:	crate::bind::types::BindType::ReadOnly,
		},
		BindRule::Symlink {
			source:	"/usr/lib".into(),
			dest:	"/lib".into(),
		},
		BindRule::Symlink {
			source:	"/usr/lib".into(),
			dest:	"/lib64".into(),
		},
		BindRule::Symlink {
			source:	"/usr/bin".into(),
			dest:	"/bin".into(),
		},
		BindRule::Symlink {
			source:	"/usr/bin".into(),
			dest:	"/sbin".into(),
		},
		/*
			We don't limit tmpfs because that may break apps
		*/
		BindRule::VirtualFS {
			dest:	"/tmp".into(),
			class:	crate::bind::types::VirtualFS::Tmpfs {
				size_mb:	None,
				perms:		None,
			},
		},

		/*
			/dev/ entries below:

			/dev/mali0, /dev/mali and /dev/umplock was not bound

			It should be handled in devices/gpu subsystem, but we have no idea of its
			information. Dropping for now.
		*/
		BindRule::VirtualFS {
			dest:	"/dev".into(),
			class:	crate::bind::types::VirtualFS::Devtmpfs,
		},
		BindRule::VirtualFS {
			dest:	"/dev/shm".into(),
			class:	crate::bind::types::VirtualFS::Tmpfs {
				size_mb:	None,
				perms:		None,
			},
		},
		BindRule::VirtualFS {
			dest:	"/dev/mqueue".into(),
			class:	crate::bind::types::VirtualFS::Mqueue,
		},
		BindRule::VirtualFS {
			dest:	"/top.kimiblock.portable".into(),
			class:	crate::bind::types::VirtualFS::Tmpfs {
				size_mb:	Some(1),
				perms:		None,
			},
		},

		/*
			/sys sysfs

			Block and regular devices, together with PCI bus are hidden,
			just as the original Portable.
		*/
		BindRule::VirtualFS {
			dest:	"/sys".into(),
			class:	crate::bind::types::VirtualFS::Tmpfs {
				size_mb:	None,
				perms:		None,
			},
		},
		BindRule::VirtualFS {
			dest:	"/sys/devices".into(),
			class:	crate::bind::types::VirtualFS::Tmpfs {
				size_mb:	None,
				perms:		None,
			},
		},
		BindRule::VirtualFS {
			dest:	"/sys/block".into(),
			class:	crate::bind::types::VirtualFS::Tmpfs {
				size_mb:	Some(0),
				perms:		None,
			},
		},
		BindRule::VirtualFS {
			dest:	"/sys/bus".into(),
			class:	crate::bind::types::VirtualFS::Tmpfs {
				size_mb:	Some(0),
				perms:		None,
			},
		},
		BindRule::Path {
			source:	"/sys/kernel".into(),
			dest:	"/sys/kernel".into(),
			class:	crate::bind::types::BindType::ReadOnly,
		},
		BindRule::Path {
			source:	"/sys/devices/virtual".into(),
			dest:	"/sys/devices/virtual".into(),
			class:	crate::bind::types::BindType::ReadOnly,
		},

		/*
			/proc procfs
		*/
		BindRule::VirtualFS {
			dest:	"/proc".into(),
			class:	crate::bind::types::VirtualFS::Procfs,
		},

		/*
			FHS compliance directories
		*/
		BindRule::VirtualFS {
			dest:	"/boot".into(),
			class:	crate::bind::types::VirtualFS::Tmpfs {
				size_mb:	Some(0),
				perms:		None,
			},
		},
		BindRule::VirtualFS {
			dest:	"/srv".into(),
			class:	crate::bind::types::VirtualFS::Tmpfs {
				size_mb:	Some(0),
				perms:		None,
			},
		},
		BindRule::VirtualFS {
			dest:	"/root".into(),
			class:	crate::bind::types::VirtualFS::Tmpfs {
				size_mb:	Some(0),
				perms:		None,
			},
		},
		BindRule::VirtualFS {
			dest:	"/media".into(),
			class:	crate::bind::types::VirtualFS::Tmpfs {
				size_mb:	Some(0),
				perms:		None,
			},
		},
		BindRule::VirtualFS {
			dest:	"/mnt".into(),
			class:	crate::bind::types::VirtualFS::Tmpfs {
				size_mb:	Some(0),
				perms:		None,
			},
		},
		BindRule::VirtualFS {
			dest:	"/home".into(),
			class:	crate::bind::types::VirtualFS::Tmpfs {
				size_mb:	None,
				perms:		None,
			},
		},
		/*
			The following paths are not mounted:

			/var/run
			/var/lock
			/var/empty
			/var/lib
			/var/log
			/var/opt
			/var/spool
			/var/tmp

			due to low usage
		*/
		BindRule::VirtualFS {
			dest:	"/var".into(),
			class:	crate::bind::types::VirtualFS::Tmpfs {
				size_mb:	Some(128),
				perms:		None,
			},
		},

		/*
			/run tmpfs

			D-Bus session bus is handled by the module itself, same is the a11y bus
			PulseAudio is handled by the audio subsystem
		*/
		BindRule::Path {
			source:	portable_runtime.path(),
			dest:	"/run".into(),
			class:	crate::bind::types::BindType::ReadWrite,
		},
		BindRule::Path {
			source:	portable_runtime.path(),
			dest:	portable_runtime.path(),
			class:	crate::bind::types::BindType::ReadWrite,
		},
		BindRule::Path {
			source:	document_mount.path_per_app(),
			dest:	document_mount.path(),
			class:	crate::bind::types::BindType::ReadWrite,
		},
		BindRule::Path {
			source:	{
				passwd_spawn
					.await
					.map_err(SystemBindError::SpawnError)?
					.map_err(SystemBindError::PasswdError)?
			},
			dest:	"/etc/passwd".into(),
			class:	crate::bind::types::BindType::ReadOnly,
		},
		BindRule::Path {
			source:	{
				nsswitch_spawn
					.await
					.map_err(SystemBindError::SpawnError)?
					.map_err(SystemBindError::NsswitchError)?
			},
			dest:	"/etc/nsswitch.conf".into(),
			class:	crate::bind::types::BindType::ReadOnly,
		},
		/*
			Privacy mounts are handled in the mask subsystem
		*/
	];

	if config.advanced.flatpak_env {
		ret.push(
			BindRule::Path {
				source:	flatpak_info.to_path_buf(),
				dest:	"/.flatpak-info".into(),
				class: crate::bind::types::BindType::ReadOnly,
			}
		);

		let info_runtime_path = {
			let mut path = xdg.runtime.to_path_buf();
			path.push(".flatpak-info");
			path
		};

		ret.push(
			BindRule::Path {
				source:	flatpak_info.to_path_buf(),
				dest:	info_runtime_path,
				class: crate::bind::types::BindType::ReadOnly,
			}
		);
	};

	// systemd notify socket
	{
		let mut notify_path = xdg.runtime.to_path_buf();
		notify_path.push("systemd");
		notify_path.push("notify");
		ret.push(
			BindRule::Path {
				source: notify_path.clone(),
				dest: notify_path,
				class: crate::bind::types::BindType::ReadWrite,
			}
		);
	};

	// Global fontconfig cache
	if tokio::fs::try_exists("/var/cache/fontconfig").await.map_err(SystemBindError::IOError)? {
		ret.push(
			BindRule::Path {
				source:	"/var/cache/fontconfig".into(),
				dest:	"/var/cache/fontconfig".into(),
				class:	crate::bind::types::BindType::ReadOnly,
			},
		);
	};

	// Mount the /dev/null pesudo device
	if tokio::fs::try_exists("/dev/null").await.map_err(SystemBindError::IOError)? {
		ret.push(
			BindRule::Path {
				source:	"/dev/null".into(),
				dest:	"/dev/null".into(),
				class:	crate::bind::types::BindType::Device,
			},
		);
	};

	// Mask certain procfs entries as they leak system info
	if tokio::fs::try_exists("/proc/uptime").await.map_err(SystemBindError::IOError)? {
		ret.push(
			BindRule::Path {
				source:	"/dev/null".into(),
				dest:	"/proc/uptime".into(),
				class:	crate::bind::types::BindType::ReadOnly,
			},
		);
	};
	if tokio::fs::try_exists("/proc/modules").await.map_err(SystemBindError::IOError)? {
		ret.push(
			BindRule::Path {
				source:	"/dev/null".into(),
				dest:	"/proc/modules".into(),
				class:	crate::bind::types::BindType::ReadOnly,
			},
		);
	};
	if tokio::fs::try_exists("/proc/cmdline").await.map_err(SystemBindError::IOError)? {
		ret.push(
			BindRule::Path {
				source:	"/dev/null".into(),
				dest:	"/proc/cmdline".into(),
				class:	crate::bind::types::BindType::ReadOnly,
			},
		);
	};
	if tokio::fs::try_exists("/proc/diskstats").await.map_err(SystemBindError::IOError)? {
		ret.push(
			BindRule::Path {
				source:	"/dev/null".into(),
				dest:	"/proc/diskstats".into(),
				class:	crate::bind::types::BindType::ReadOnly,
			},
		);
	};
	if tokio::fs::try_exists("/proc/devices").await.map_err(SystemBindError::IOError)? {
		ret.push(
			BindRule::Path {
				source:	"/dev/null".into(),
				dest:	"/proc/devices".into(),
				class:	crate::bind::types::BindType::ReadOnly,
			},
		);
	};
	if tokio::fs::try_exists("/proc/config.gz").await.map_err(SystemBindError::IOError)? {
		ret.push(
			BindRule::Path {
				source:	"/dev/null".into(),
				dest:	"/proc/config.gz".into(),
				class:	crate::bind::types::BindType::ReadOnly,
			},
		);
	};
	if tokio::fs::try_exists("/proc/loadavg").await.map_err(SystemBindError::IOError)? {
		ret.push(
			BindRule::Path {
				source:	"/dev/null".into(),
				dest:	"/proc/loadavg".into(),
				class:	crate::bind::types::BindType::ReadOnly,
			},
		);
	};

	// Games won't function without it
	if tokio::fs::try_exists("/sys/devices/system").await.map_err(SystemBindError::IOError)? {
		ret.push(
			BindRule::Path {
				source:	"/sys/devices/system".into(),
				dest:	"/sys/devices/system".into(),
				class:	crate::bind::types::BindType::ReadWrite,
			},
		);
	};


	// CGroups not showing up doesn't break Init
	if tokio::fs::try_exists("/sys/fs/cgroup").await.map_err(SystemBindError::IOError)? {
		ret.push(
			BindRule::Path {
				source:	"/sys/fs/cgroup".into(),
				dest:	"/sys/fs/cgroup".into(),
				class:	crate::bind::types::BindType::ReadWrite,
			},
		);
	};

	if tokio::fs::try_exists("/sys/dev/char").await.map_err(SystemBindError::IOError)? {
		ret.push(
			BindRule::Path {
				source:	"/sys/dev/char".into(),
				dest:	"/sys/dev/char".into(),
				class:	crate::bind::types::BindType::ReadOnly,
			},
		);
	};

	if tokio::fs::try_exists("/sys/module").await.map_err(SystemBindError::IOError)? {
		ret.push(
			BindRule::Path {
				source:	"/sys/module".into(),
				dest:	"/sys/module".into(),
				class:	crate::bind::types::BindType::ReadOnly,
			},
		);
	};

	if tokio::fs::try_exists("/dev/udmabuf").await.map_err(SystemBindError::IOError)? {
		ret.push(
			BindRule::Path {
				source: "/dev/udmabuf".into(),
				dest: "/dev/udmabuf".into(),
				class: crate::bind::types::BindType::Device,
			}
		);
	};
	if tokio::fs::try_exists("/dev/ntsync").await.map_err(SystemBindError::IOError)? {
		ret.push(
			BindRule::Path {
				source: "/dev/ntsync".into(),
				dest: "/dev/ntsync".into(),
				class: crate::bind::types::BindType::Device,
			}
		);
	};

	ret.extend(
		bin_spawn
			.await
			.map_err(SystemBindError::SpawnError)
			?
			.map_err(SystemBindError::BinError)
			?
	);

	ret.extend(
		machine_id
			.await
			.map_err(SystemBindError::SpawnError)
			?
			?
	);

	ret.extend(
		resolv_spawn
			.await
			.map_err(SystemBindError::SpawnError)?
	);

	ret.extend(
		kvm_spawn
			.await
			.map_err(SystemBindError::SpawnError)?
			.map_err(SystemBindError::KvmError)?
	);

	Ok(ret)
}

#[derive(thiserror::Error, Debug)]
pub enum SystemBindError {
	#[error("Error while generating machine-id: expected parent path to be Some")]
	NoneParentForMachineID,

	#[error("I/O error while reading path: {0:#?}")]
	IOError(std::io::Error),

	#[error("Could not generate nsswitch file: {0:#?}")]
	NsswitchError(nsswitch::NsswitchError),

	#[error("Could not generate passwd file: {0:#?}")]
	PasswdError(passwd::PasswdError),

	#[error("Could not spawn task: {0:#?}")]
	SpawnError(tokio::task::JoinError),

	#[error("Could not mount kvm device: {0:#?}")]
	KvmError(kvm::KvmError),

	#[error("Could not mount binaries: {0:#?}")]
	BinError(bin::BinError),
}
