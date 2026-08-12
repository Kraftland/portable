
/**
	A generic trait for other subsystems to implement binding generation

	Portable's bind rule generation system is divided to multiple subsystems. Each of them may
	implement different functions and are generally controlled via Cargo feature switches.

	Every subsystem has a unique struct to pass along information.
*/
pub trait GenerateBind {
	fn bind(self) -> impl std::future::Future<Output = Result<super::types::BindRules, Self::BindError>> + Send;

	type BindError;
}

pub async fn generate_bindrules(
	portable_runtime:	crate::bind::subsystems::dirs::portable_runtime::PortableRuntime,
	document_mount:		crate::bind::subsystems::dirs::documents::DocumentsMountPoint,

	xdg:			std::sync::Arc<crate::xdg::XdgDirs>,
	config:			std::sync::Arc<crate::config::config_definition::Config>,

	logger:			crate::logger::LogSender,
	env:			crate::envs::holder::HoldChannel,
	instance_id:		String,

	flatpak_info_path:	std::sync::Arc<std::path::PathBuf>,

)
-> Result<super::types::BindRules, BindError> {

	let mut workers = vec![];

	{
		let system_bind = system::SystemBind {
			config:			config.clone(),
			portable_runtime:	portable_runtime.clone(),
			document_mount:		document_mount,
			xdg:			xdg.clone(),
			flatpak_info:		flatpak_info_path,
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
		for allow in config.system.device_allow {
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
	{
		let display_bind = display::Display {
			xdg:			xdg.clone(),
			logger:			logger,
			env:			env,
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
			xdg_dir)

		let user_bind = user::UserBind {

		};
	};

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

	Ok(ret)
}

#[derive(thiserror::Error, Debug)]
pub enum BindError {
	#[error("Could not bind system paths: {0:#?}")]
	SystemBindError(system::SystemBindError),

	#[error("Could not bind devices: {0:#?}")]
	DeviceBindError(devices::DeviceError),

	#[error("Could not bind display sockets: {0:#?}")]
	DisplayBindError(display::DisplayError),

	#[error("Could not mask certain paths: {0:#?}")]
	MaskError(mask::MaskError),

	#[error("Could not spawn bind task: {0:#?}")]
	SpawnError(tokio::task::JoinError),
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
