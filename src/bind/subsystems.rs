
/**
	A generic trait for other subsystems to implement binding generation

	Portable's bind rule generation system is divided to multiple subsystems. Each of them may
	implement different functions and are generally controlled via Cargo feature switches.

	Every subsystem has a unique struct to pass along information.

	The Init info struct is also returned along with bind rules.
*/
pub trait GenerateBind {
	fn bind(self) -> impl std::future::Future<Output = Result<super::types::BindRules, Self::BindError>> + Send;

	type BindError;
}

pub async fn generate_bindrules(
	portable_runtime:	std::sync::Arc<crate::bind::subsystems::dirs::portable_runtime::PortableRuntime>,
	document_mount:		crate::bind::subsystems::dirs::documents::DocumentsMountPoint,
	xdg:			std::sync::Arc<crate::xdg::XdgDirs>,
	config:			std::sync::Arc<crate::config::config_definition::Config>,
	logger:			crate::logger::LogSender,
	stop_tx:		tokio::sync::mpsc::Sender<crate::stop::StopLevel>,
	stop_func:		tokio::sync::mpsc::Sender<crate::stop::StopFunc>,
	env:			crate::envs::holder::HoldChannel,
	instance_id:		String,
	flatpak_info_path:	std::sync::Arc<std::path::PathBuf>,

	runtime_opts:		std::sync::Arc<crate::pref::runtime::options::RuntimeOpts>,

	dbus_conn:		zbus::Connection,
)
-> Result<(super::types::BindRules, crate::ipc::init::info::InitInfo), BindError> {

	let mut workers = vec![];

	{
		let system_bind = system::SystemBind {
			config:			config.clone(),
			portable_runtime:	portable_runtime.clone(),
			document_mount:		document_mount,
			xdg:			xdg.clone(),
			flatpak_info:		flatpak_info_path,
			logger:			logger.clone(),
		};

		workers.push(
			tokio::spawn(
				async {
					system_bind
						.bind()
						.await
						.map_err(BindError::SystemBindError)
				}
			)
		);
	};
	{
		use crate::config::config_definition::DeviceAllow;

		let mut all_gpus = false;
		let mut bind_cam = false;
		let mut bind_input = false;
		for allow in &config.system.device_allow {
			match allow {
				DeviceAllow::DiscreteGPU	=> {
					all_gpus = true
				}
				DeviceAllow::Camera		=> {
					bind_cam = true
				}
				DeviceAllow::Input		=> {
					bind_input = true
				}
				_				=> {}
			}
		};

		let device_bind = devices::Devices {
			all_gpus:	all_gpus,
			bind_camera:	bind_cam,
			bind_input:	bind_input,
			logger:		logger.clone(),
		};

		workers.push(
			tokio::spawn(
				async {
					device_bind
						.bind()
						.await
						.map_err(BindError::DeviceBindError)
				}
			)
		);
	};

	#[cfg(feature = "display")]
	{
		let display_bind = display::Display {
			xdg:			xdg.clone(),
			logger:			logger.clone(),
			env:			env.clone(),
			portable_runtime:	portable_runtime,
			app_id:			config.metadata.sandbox_id.to_string(),
			instance_id:		instance_id,
			x11:			config.privacy.x11_compat,
			/*
				We don't have a dedicated Wayland configuration now
				It will be enabled on session type detection
			*/
			wayland:		false,
		};

		workers.push(
			tokio::spawn(
				async {
					display_bind
						.bind()
						.await
						.map_err(BindError::DisplayBindError)
				}
			)
		);
	};

	#[cfg(feature = "audio")]
	{
		let audio_bind = audio::Audio {
			logger:		logger.clone(),
			runtime_dir:	xdg.runtime.to_path_buf(),
			env:		env.clone(),
		};

		workers.push(
			tokio::spawn(
				async {
					audio_bind
						.bind()
						.await
						.map_err(BindError::AudioError)
				}
			)
		);
	};

	{
		let mask_bind = mask::Mask {};
		workers.push(
			tokio::spawn(
				async {
					mask_bind
						.bind()
						.await
						.map_err(BindError::MaskError)
				}
			)
		);
	};
	{
		let translator = crate::bind::translate::Delta::get(
			&config,
			&xdg,
		).await;

		let user_bind = user::UserBind {
			translator:	translator,
			xdg:		xdg.clone(),
			config:		config.clone(),
		};
		workers.push(
			tokio::spawn(
				async {
					user_bind
						.bind()
						.await
						.map_err(BindError::UserBindError)
				}
			)
		);
	};

	let (expose_rules, forward_map) = {
		user::forward_file(
			&runtime_opts.file_expose,
			runtime_opts.bus_activation,
			&dbus_conn,
			&config.metadata.sandbox_id,
			logger.clone(),
		)
		.await
	};

	// Previously in miscEnvs
	{
		if config.advanced.qt5_compat {
			env.send(
				crate::envs::holder::EnvMessage::Add {
					key: "QT_QPA_PLATFORMTHEME".into(),
					value: "xdgdesktopportal".into(),
				}
			)
				.await
				.map_err(BindError::EnvError)
				?
		}

		env.send(
			crate::envs::holder::EnvMessage::Add {
				key: "GDK_DEBUG".into(),
				value: "portals".into(),
			}
		)
			.await
			.map_err(BindError::EnvError)
			?;
		env.send(
			crate::envs::holder::EnvMessage::Add {
				key: "GTK_USE_PORTAL".into(),
				value: "1".into(),
			}
		)
			.await
			.map_err(BindError::EnvError)
			?;
		env.send(
			crate::envs::holder::EnvMessage::Add {
				key: "QT_AUTO_SCREEN_SCALE_FACTOR".into(),
				value: "1".into(),
			}
		)
			.await
			.map_err(BindError::EnvError)
			?;
		env.send(
			crate::envs::holder::EnvMessage::Add {
				key: "QT_ENABLE_HIGHDPI_SCALING".into(),
				value: "1".into(),
			}
		)
			.await
			.map_err(BindError::EnvError)
			?;
		env.send(
			crate::envs::holder::EnvMessage::Add {
				key: "PS1".into(),
				value: {
					let mut ps1 = String::new();
					ps1.push_str("🗃  ╰─>Portable: ");
					ps1.push_str(&config.metadata.sandbox_id);
					ps1.push_str("·👻 ➵ ");
					ps1
				},
			}
		)
			.await
			.map_err(BindError::EnvError)
			?;
		env.send(
			crate::envs::holder::EnvMessage::Add {
				key: "GDK_DEBUG".into(),
				value: "portals".into(),
			}
		)
			.await
			.map_err(BindError::EnvError)
			?;
		env.send(
			crate::envs::holder::EnvMessage::Add {
				key: "GDK_DEBUG".into(),
				value: "portals".into(),
			}
		)
			.await
			.map_err(BindError::EnvError)
			?;
		env.send(
			crate::envs::holder::EnvMessage::Add {
				key: "GDK_DEBUG".into(),
				value: "portals".into(),
			}
		)
			.await
			.map_err(BindError::EnvError)
			?;
	}

	let mut ret = vec![];

	for worker in workers {
		ret.extend(
			worker
				.await
				.map_err(BindError::SpawnError)
				?
				?
		);
	};

	ret.extend(expose_rules);

	let init_info = crate::ipc::init::info::InitInfo {
		extra_files:		forward_map,
		inhibit_suspend:	config.system.conduct_inhibit,
		flatpak_info:		config.advanced.flatpak_env,
		lockdown:		config.privacy.lockdown,
		allow_debug:		config.advanced.allow_debug,
		logger:			logger.clone(),
		stop_tx:		stop_tx,
		stop_func:		stop_func,
		target_exec:		{
			if runtime_opts.bus_activation {
				if config.dbus_activation.enable {
					config.dbus_activation.target.to_owned()
				} else {
					return Err(
						BindError::ActivationNotAllowed
					);
				}
			} else {
				if runtime_opts.debug_shell {
					String::from("bash")
				} else {
					config.exec.target.to_owned()
				}
			}
		},
		target_args:		{
			let mut base = if runtime_opts.bus_activation {
				if config.dbus_activation.enable {
					config.dbus_activation.arguments.to_owned()
				} else {
					return Err(
						BindError::ActivationNotAllowed
					);
				}
			} else {
				config.exec.arguments.to_owned()
			};
			base.extend(runtime_opts.app_args.to_owned());

			if runtime_opts.debug_shell {
				vec!["-i".to_string()]
			} else {
				base
			}
		},
		uclamp_min:		0,
		uclamp_max:		config.system.uclamp_max,
	};

	Ok((ret, init_info))
}

#[derive(thiserror::Error, Debug)]
pub enum BindError {
	#[error("Could not bind system paths: {0:#?}")]
	SystemBindError(system::SystemBindError),

	#[error("Could not bind devices: {0:#?}")]
	DeviceBindError(devices::DeviceError),

	#[error("Could not bind audio service: {0:#?}")]
	AudioError(audio::PulseError),

	#[error("Could not bind display sockets: {0:#?}")]
	DisplayBindError(display::DisplayError),

	#[error("Could not mask certain paths: {0:#?}")]
	MaskError(mask::MaskError),

	#[error("Could not bind user paths: {0:#?}")]
	UserBindError(user::UserBindError),

	#[error("D-Bus activation was requested while not enabled in configuration")]
	ActivationNotAllowed,

	#[error("Subsystem called on non-normal action")]
	ActionError,

	#[error("Could not spawn bind task: {0:#?}")]
	SpawnError(tokio::task::JoinError),

	#[error("Could not send environment variables: {0:#?}")]
	EnvError(tokio::sync::mpsc::error::SendError<crate::envs::holder::EnvMessage>),
}

#[cfg(feature = "audio")]
pub mod audio;

#[cfg(feature = "devices")]
pub mod devices;

pub mod dirs;

#[cfg(feature = "display")]
mod display;

pub mod desktop_file;

pub mod mask;

pub mod user;

mod system;
