mod passwd;
mod nsswitch;
mod resolv;
mod bin;
mod machine_id;
mod etc;
pub mod exist;

/**
	The system bind subsystem
*/
pub struct SystemBind {
	pub config:		std::sync::Arc<crate::config::Config>,

	pub xdg:		std::sync::Arc<crate::xdg::XdgDirs>,

	pub portable_runtime:	std::sync::Arc<crate::bind::subsystems::dirs::portable_runtime::PortableRuntime>,
	pub document_mount:	crate::bind::subsystems::dirs::documents::DocumentsMountPoint,

	pub flatpak_info:	std::sync::Arc<std::path::PathBuf>,
}

impl super::GenerateBind for SystemBind {
	async fn bind(self) -> Result<crate::bind::types::BindRules, Self::BindError> {
		bind(
			self.config,
			self.portable_runtime,
			self.document_mount,
			self.xdg,
			self.flatpak_info,
		).await
	}
	type BindError = SystemBindError;
}

/**
	The system bind subsystem exposes several paths of system root.

	It is designed to provide theming consistency in mind. Masking is done via the mask subsystem.
*/
async fn bind(
	config:			std::sync::Arc<crate::config::Config>,
	portable_runtime:	std::sync::Arc<crate::bind::subsystems::dirs::portable_runtime::PortableRuntime>,
	document_mount:		crate::bind::subsystems::dirs::documents::DocumentsMountPoint,
	xdg:			std::sync::Arc<crate::xdg::XdgDirs>,
	flatpak_info:		std::sync::Arc<std::path::PathBuf>,
)
-> Result<crate::bind::types::BindRules, SystemBindError>
{
	let bin_spawn = tokio::spawn(
		bin::bind(config.clone())
	);

	let etc_spawn = tokio::spawn(
		etc::bind_etc(
			config.clone(),
			xdg.clone(),
			portable_runtime.clone(),
		)
	);

	use crate::bind::subsystems::dirs::RuntimePathsTrait;
	use crate::bind::types::BindRule;
	let mut ret = vec![
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

		/*
			/dev/null pesudo device
		*/
		BindRule::Path {
		source:	"/dev/null".into(),
			dest:	"/dev/null".into(),
			class:	crate::bind::types::BindType::Device,
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

	{
		let rw_paths = [
			// Games require this
			"/sys/devices/system",

		];

		let ro_paths = [
			// Character devices are needed
			"/sys/dev/char",
			"/sys/module",
		];

		let device_paths = [
			"/dev/udmabuf",
			"/dev/ntsync",
		];

		for path in exist::list_exists(rw_paths).await.map_err(SystemBindError::ExistError)? {
			ret.push(
				BindRule::Path {
					source:	path.to_path_buf(),
					dest:	path.to_path_buf(),
					class:	crate::bind::types::BindType::ReadWrite,
				}
			);
		};

		for path in exist::list_exists(device_paths).await.map_err(SystemBindError::ExistError)? {
			ret.push(
				BindRule::Path {
					source:	path.to_path_buf(),
					dest:	path.to_path_buf(),
					class:	crate::bind::types::BindType::Device,
				}
			);
		};

		for path in exist::list_exists(ro_paths).await.map_err(SystemBindError::ExistError)? {
			ret.push(
				BindRule::Path {
					source:	path.to_path_buf(),
					dest:	path.to_path_buf(),
					class:	crate::bind::types::BindType::ReadOnly,
				}
			);
		};
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

	ret.extend(
		bin_spawn
			.await
			.map_err(SystemBindError::SpawnError)
			?
			.map_err(SystemBindError::BinError)
			?
	);

	ret.extend(
		etc_spawn
			.await
			.map_err(SystemBindError::SpawnError)
			?
			.map_err(SystemBindError::EtcError)
			?
	);

	Ok(ret)
}

#[derive(thiserror::Error, Debug)]
pub enum SystemBindError {
	#[error("I/O error while reading path: {0}")]
	IOError(std::io::Error),

	#[error("Could not spawn task: {0}")]
	SpawnError(tokio::task::JoinError),

	#[error("Could not mount binaries: {0}")]
	BinError(bin::BinError),

	#[error("Could not mount /etc: {0}")]
	EtcError(etc::EtcError),

	#[error("Could not detect if path exists: {0}")]
	ExistError(exist::ExistError),
}
