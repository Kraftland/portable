
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
	xdg_runtime:		std::path::PathBuf,
	data_dir:		std::path::PathBuf,
	state_dir:		String,
	overlay_bin:		bool,
	device_allow:		Vec<crate::config::config_definition::DeviceAllow>,
	app_id:			String,

	logger:			crate::logger::LogSender,

	allow_x11:		bool,
	home:			std::path::PathBuf,
	env:			crate::envs::holder::HoldChannel,
	instance_id:		String,

)
-> Result<super::types::BindRules, BindError> {
	let xdg_runtime = std::sync::Arc::new(xdg_runtime);

	let mut workers = vec![];

	{
		let system_bind = system::SystemBind {
			portable_runtime:	portable_runtime.clone(),
			document_mount:		document_mount,
			xdg_runtime:		xdg_runtime.clone(),
			data_dir:		data_dir,
			state_dir:		state_dir,
			overlay_bin:		overlay_bin,
			device_allow:		device_allow.clone(),
			app_id:			app_id.clone(),
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
		for allow in device_allow {
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
			logger:			logger,
			home:			home,
			runtime_dir:		xdg_runtime.clone(),
			env:			env,
			portable_runtime:	portable_runtime,
			app_id:			app_id,
			instance_id:		instance_id,
			x11:			allow_x11,
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

	unimplemented!();

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
