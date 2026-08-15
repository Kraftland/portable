/*
	Proxy for the D-Bus session bus address
*/

mod portal_allowlist;
mod address;
mod sandbox;

/**
	The public struct proxy is used to define rules and sandboxing layer for xdg-dbus-proxy

	The flatpak info file will always be under Portable's runtime directory, named flatpak-info
*/
pub struct SessionProxy {
	pub logger:		crate::logger::LogSender,
	pub config:		std::sync::Arc<crate::config::config_definition::Config>,
	pub stop_token:		Option<tokio::sync::mpsc::Sender<crate::stop::StopLevel>>,
	pub portable_dir:	std::sync::Arc<crate::bind::subsystems::dirs::portable_runtime::PortableRuntime>,
	#[cfg(feature = "flatpak")]
	pub status_fd:		Option<std::os::fd::OwnedFd>,
	#[cfg(feature = "flatpak")]
	pub flatpak_info:		std::path::PathBuf,
}

impl crate::bind::bus::StartProxy for SessionProxy {

	async fn new(
			self
	) -> Result<crate::bind::bus::Proxy, Self::ProxyError>
	{
		compile_rules(
			self.logger,
			self.config,
			self.stop_token,
			self.portable_dir,
			#[cfg(feature = "flatpak")]
			self.status_fd,
			#[cfg(feature = "flatpak")]
			self.flatpak_info,
		).await
	}

	type ProxyError = ProxyError;

}

async fn compile_rules(
	logger:		crate::logger::LogSender,
	config:		std::sync::Arc<crate::config::config_definition::Config>,
	stop_token:	Option<tokio::sync::mpsc::Sender<crate::stop::StopLevel>>,
	portable_dir:	std::sync::Arc<crate::bind::subsystems::dirs::portable_runtime::PortableRuntime>,
	#[cfg(feature = "flatpak")]
	status_fd:	Option<std::os::fd::OwnedFd>,
	#[cfg(feature = "flatpak")]
	flatpak_info:	std::path::PathBuf,
) -> Result<crate::bind::bus::Proxy, ProxyError> {
	let bus_address = tokio::spawn(get_session_bus_address());

	let (proxy_socket_path, app_sandbox) = address::get_address_with_sandbox(portable_dir)
		.await
		?;

	let bus_access_spawn = tokio::spawn(
		generate_bus_rules(
			config.metadata.sandbox_id.to_string(),
			config.advanced.allow_kde_status,
			config.advanced.mpris_names.clone(),
			config.privacy.classic_notif,
			config.system.allow_inhibit,
		),
	);

	let sandbox_rules = tokio::spawn(
		sandbox::generate_sandbox_rules(
			proxy_socket_path.to_path_buf(),

			#[cfg(feature = "flatpak")]
			flatpak_info,
		),
	);

	Ok(
		crate::bind::bus::Proxy {
			sandbox:		sandbox_rules
							.await
							.map_err(ProxyError::SpawnError)
							?
							?,
			bus_access:		bus_access_spawn
							.await
							.map_err(ProxyError::SpawnError)
							?
							?,
			bus_address: 		bus_address
							.await
							.map_err(ProxyError::SpawnError)
							?
							?,
			logger:			logger,
			proxy_socket:		proxy_socket_path,
			bind_lifetime:		stop_token,
			sloppy_names:		false,
			#[cfg(feature = "flatpak")]
			json_status_file:	status_fd,
			app_sandbox:		Some(app_sandbox),
			envs:			{
				let mut map = std::collections::HashMap::new();
				map.insert(
					"DBUS_SESSION_BUS_ADDRESS".to_string(),
					"unix:path=/run/session_bus/bus".to_string(),
				);
				Some(map)
			},
		}
	)
}



async fn generate_bus_rules(
	app_id:			String,
	kde_status:		bool,
	mpris_names:		Vec<String>,
	classic_notif:		bool,
	inhibit:		bool,
) -> Result<Vec<crate::bind::bus::rules::BusAccessLevel>, ProxyError> {
	use crate::bind::bus::rules::BusAccessLevel;
	use crate::bind::bus::rules::BusName;

	let status_notifier_spawn = tokio::spawn(generate_status_notifier_rules());

	let mut rules: Vec<BusAccessLevel> = vec![
		BusAccessLevel::OwnName {
			bus_name: {
				let mut name = String::from(&app_id);
				name.push_str(".*");
				BusName::try_from(name)
					.map_err(ProxyError::InvalidBusNameError)
					?
			},
		},
		BusAccessLevel::OwnName {
			bus_name: {
				let name = String::from(&app_id);
				BusName::try_from(name)
					.map_err(ProxyError::InvalidBusNameError)
					?
			},
		},

		/*
			TODO: We need to properly sandbox UnifiedPush distributors!
		*/
		BusAccessLevel::WellknownName {
			bus_name: BusName::try_from("org.unifiedpush.Distributor.*")
				.map_err(ProxyError::InvalidBusNameError)
				?,
		},
		BusAccessLevel::See {
			bus_name: BusName::try_from("org.a11y.Bus")
				.map_err(ProxyError::InvalidBusNameError)
				?,
		},

		BusAccessLevel::Call {
			bus_name: BusName::try_from("org.a11y.Bus")
				.map_err(ProxyError::InvalidBusNameError)
				?,
			method: "org.a11y.Bus.GetAddress".to_string(),
			object_path: "/org/a11y/bus".into(),
		},
		BusAccessLevel::Call {
			bus_name: BusName::try_from("org.a11y.Bus")
				.map_err(ProxyError::InvalidBusNameError)
				?,
			method: "org.freedesktop.DBus.Properties.Get".into(),
			object_path: "/org/a11y/bus".into(),
		},

		// Request interface
		BusAccessLevel::Call {
			bus_name: BusName::try_from("org.freedesktop.portal.Desktop")
				.map_err(ProxyError::InvalidBusNameError)
				?,
			method: "org.freedesktop.portal.Request.*".into(),
			object_path: "/org/freedesktop/portal/desktop/request/*".into(),
		},
		BusAccessLevel::GetBroadcast {
			bus_name: BusName::try_from("org.freedesktop.portal.Desktop")
				.map_err(ProxyError::InvalidBusNameError)
				?,
			method: "org.freedesktop.portal.Request.*".into(),
			object_path: "/org/freedesktop/portal/desktop/request/*".into(),
		},

		// Session interface
		BusAccessLevel::Call {
			bus_name: BusName::try_from("org.freedesktop.portal.Desktop")
				.map_err(ProxyError::InvalidBusNameError)
				?,
			method: "org.freedesktop.portal.Session.*".into(),
			object_path: "/org/freedesktop/portal/desktop/session/*".into(),
		},
		BusAccessLevel::GetBroadcast {
			bus_name: BusName::try_from("org.freedesktop.portal.Desktop")
				.map_err(ProxyError::InvalidBusNameError)
				?,
			method: "org.freedesktop.portal.Session.*".into(),
			object_path: "/org/freedesktop/portal/desktop/session/*".into(),
		},

		// Properties interface
		BusAccessLevel::Call {
			bus_name: BusName::try_from("org.freedesktop.portal.Desktop")
				.map_err(ProxyError::InvalidBusNameError)
				?,
			method: "org.freedesktop.DBus.Properties.*".into(),
			object_path: "/org/freedesktop/portal/desktop/*".into(),
		},

		/*
			TODO: We need to properly sandbox the global menu!
		*/
		BusAccessLevel::WellknownName {
			bus_name: BusName::try_from("com.canonical.AppMenu.Registrar")
				.map_err(ProxyError::InvalidBusNameError)
				?,
		},

		// Stop the sandbox
		BusAccessLevel::Call {
			bus_name: {
				let mut name = String::from("top.kimiblock.portable.");
				name.push_str(&app_id);
				BusName::try_from(name)
					.map_err(ProxyError::InvalidBusNameError)
					?
			},
			method: "top.kimiblock.Portable.Controller.Stop".into(),
			object_path: "/top/kimiblock/portable/daemon".into(),
		},

		// Calling StatusNotifier endpoints
		BusAccessLevel::Call {
			bus_name: BusName::try_from("org.kde.StatusNotifierWatcher")
				.map_err(ProxyError::InvalidBusNameError)
				?,
			method: "org.kde.StatusNotifierWatcher.*".into(),
			object_path: "/StatusNotifierWatcher".into(),
		},
		// Receiving broadcast from StatusNotifier endpoints
		BusAccessLevel::GetBroadcast {
			bus_name: BusName::try_from("org.kde.StatusNotifierWatcher")
				.map_err(ProxyError::InvalidBusNameError)
				?,
			method: "org.kde.StatusNotifierWatcher.*".into(),
			object_path: "/StatusNotifierWatcher".into(),
		},

		// Documents Portal, seems well sandboxed so we just expose the full name
		BusAccessLevel::WellknownName {
			bus_name: BusName::try_from("org.freedesktop.portal.Documents")
				.map_err(ProxyError::InvalidBusNameError)
				?,
		},

		// CreateInputContext for /inputmethod obj path
		BusAccessLevel::Call {
			bus_name: BusName::try_from("org.freedesktop.portal.Fcitx")
				.map_err(ProxyError::InvalidBusNameError)
				?,
			method: "org.fcitx.Fcitx.InputMethod1.CreateInputContext".into(),
			object_path: "/inputmethod".into(),
		},
		BusAccessLevel::Call {
			bus_name: BusName::try_from("org.freedesktop.portal.Fcitx")
				.map_err(ProxyError::InvalidBusNameError)
				?,
			method: "org.fcitx.Fcitx.InputMethod1.CreateInputContext".into(),
			object_path: "/org/freedesktop/portal/inputmethod".into(),
		},
		// iBus portal
		BusAccessLevel::Call {
			bus_name: BusName::try_from("org.freedesktop.portal.IBus")
				.map_err(ProxyError::InvalidBusNameError)
				?,
			method: "org.freedesktop.IBus.Portal.*".into(),
			object_path: "/org/freedesktop/IBus".into(),
		},

		// Call FileManager1
		BusAccessLevel::Call {
			bus_name: BusName::try_from("org.freedesktop.FileManager1")
				.map_err(ProxyError::InvalidBusNameError)
				?,
			method: "org.freedesktop.FileManager1.*".into(),
			object_path: "/org/freedesktop/FileManager1".into(),
		},
	];

	// MPRIS default names
	{
		let app_id_last_segment = match app_id.split(".").last() {
			Some(v)	=> {v}
			None	=> {
				return Err(ProxyError::InfiniteSegmentError);
			}
		};
		let mut mpris_compat = String::from("org.mpris.MediaPlayer2.");
		let mut mpris_appid = String::from("org.mpris.MediaPlayer2.");
		mpris_compat.push_str(app_id_last_segment);
		mpris_compat.push_str(".*");

		mpris_appid.push_str(&app_id);
		mpris_appid.push_str(".*");

		let bus_names = vec![
			mpris_compat,
			mpris_appid,
		];

		for bus_name in bus_names {
			rules.push(
				BusAccessLevel::OwnName {
					bus_name: BusName::try_from(bus_name)
						.map_err(ProxyError::InvalidBusNameError)
						?,
					}
			);
		};
	};

	if mpris_names.len() > 0 {
		for mpris_name in mpris_names {
			rules.push(
				BusAccessLevel::OwnName {
					bus_name: {
						let mut name = String::from("org.mpris.MediaPlayer2.");
						name.push_str(&mpris_name);
						BusName::try_from(name)
							.map_err(ProxyError::InvalidBusNameError)
							?
					},
				}
			);
		}
	}

	if classic_notif {
		rules.push(
			BusAccessLevel::Call {
				bus_name: BusName::try_from("org.freedesktop.Notifications")
					.map_err(ProxyError::InvalidBusNameError)
					?,
				method: "org.freedesktop.Notifications.*".into(),
				object_path: "/org/freedesktop/Notifications".into(),
			}
		);
		rules.push(
			BusAccessLevel::GetBroadcast {
				bus_name: BusName::try_from("org.freedesktop.Notifications")
					.map_err(ProxyError::InvalidBusNameError)
					?,
				method: "org.freedesktop.Notifications.*".into(),
				object_path: "/org/freedesktop/Notifications".into(),
			}
		);
	}

	if kde_status {
		rules.push(
			BusAccessLevel::Call {
				bus_name: BusName::try_from("org.kde.JobViewServer")
					.map_err(ProxyError::InvalidBusNameError)
					?,
				method: "org.kde.JobViewServerV2.requestView".into(),
				object_path: "/JobViewServer".into(),
			}
		);
		rules.push(
			BusAccessLevel::Call {
				bus_name: BusName::try_from("org.kde.JobViewServer")
					.map_err(ProxyError::InvalidBusNameError)
					?,
				method: "org.kde.JobViewV3.update".into(),
				object_path: "/org/kde/notificationmanager/jobs/*".into(),
			}
		);
		rules.push(
			BusAccessLevel::Call {
				bus_name: BusName::try_from("org.kde.JobViewServer")
					.map_err(ProxyError::InvalidBusNameError)
					?,
				method: "org.kde.JobViewServer=org.kde.JobViewV3.terminate".into(),
				object_path: "/org/kde/notificationmanager/jobs/*".into(),
			}
		);
	}

	{
		let portals = portal_allowlist::get_allowed_portals(inhibit).await;
		for portal in portals {
			rules.push(
				BusAccessLevel::Call {
					bus_name: BusName::try_from("org.freedesktop.portal.Desktop")
						.map_err(ProxyError::InvalidBusNameError)
						?,
					method: {
						let mut method = String::from("org.freedesktop.portal.");
						method.push_str(&portal);
						method.push_str(".*");
						method
					},
					object_path: "/org/freedesktop/portal/desktop".into(),
				}
			);
		};
	};

	{
		let tray_rules = status_notifier_spawn
			.await
			.map_err(ProxyError::SpawnError)
			?
			?;
		for rule in tray_rules {
			rules.push(rule);
		}
	};


	Ok(rules)
}

async fn generate_status_notifier_rules() -> Result<Vec<crate::bind::bus::rules::BusAccessLevel>, ProxyError> {
	use crate::bind::bus::rules::BusAccessLevel;
	use crate::bind::bus::rules::BusName;

	let threads = std::thread::available_parallelism()
		.map(|n| n.get())
		.map_err(ProxyError::CoreCountError)
		?;

	let mut counter: u8 = 0;
	let mut pid: usize = threads - 1;
	let mut ret = vec![];
	let name_prefix = String::from("org.kde.StatusNotifierItem-");

	loop {
		if counter > 10 {
			return Ok(ret);
		}
		counter += 1;

		let mut name = String::from(&name_prefix);
		name.push_str(&pid.to_string());
		name.push_str("-1");

		ret.push(
			BusAccessLevel::OwnName {
				bus_name: BusName::try_from(name)
					.map_err(ProxyError::InvalidBusNameError)
					?,
			}
		);
		pid += 1;
	}

}

async fn get_session_bus_address() -> Result<String, ProxyError> {
	let env = std::env::var("DBUS_SESSION_BUS_ADDRESS")
		.map_err(ProxyError::AddressUnknownError)
		?;
	Ok(env)
}



use thiserror::Error;
#[derive(Debug, Error)]
pub enum ProxyError {
	#[error("I/O error: {0:#?}")]
	IOError(std::io::Error),

	#[error("No parent directory")]
	NoParentDirectory,

	#[error("Could not start D-Bus proxy for session bus: invalid address: {0:#?}")]
	AddressUnknownError(std::env::VarError),

	#[error("Could not start D-Bus proxy for session bus: thread spawn error: {0:#?}")]
	SpawnError(tokio::task::JoinError),

	#[error("Could not start D-Bus proxy for session bus: invalid bus name: {0:#?}")]
	InvalidBusNameError(crate::bind::bus::rules::BusNameError),

	#[error("Could not start D-Bus proxy for session bus: invalid character: {0:#?}")]
	OsStringError(std::io::Error),

	#[error("Could not start D-Bus proxy: segment is infinite")]
	InfiniteSegmentError,

	#[error("Could not start D-Bus proxy: error obtaining CPU core count: {0:#?}")]
	CoreCountError(std::io::Error),
}
