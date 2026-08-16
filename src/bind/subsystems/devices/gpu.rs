use thiserror::Error;

use crate::bind::types::BindRule;

mod enumerate;
mod associate;
mod get_info;

pub mod nvidia;
pub mod prime;
mod bind;
mod udev_dev;

#[derive(Error, Debug)]
pub enum GPUError {
	#[error("Could not determine boot display: invalid value {0:?}")]
	InvalidBootDisplay(String),
	#[error("Could not determine boot vga: invalid value {0:?}")]
	InvalidBootVGA(String),

	#[error("Could not enumerate GPUs: {0:#?}")]
	Enumerate(crate::bind::subsystems::devices::EnumerateError),

	#[error("Could not enumerate GPUs: spawn error: {0:#?}")]
	Spawn(tokio::task::JoinError),

	#[error("Could not send zink envs: {0:#?}")]
	EnvsError(tokio::sync::mpsc::error::SendError<crate::envs::holder::EnvMessage>),
}




/**
	The public function scan implements the GPU binding subsystem
*/
pub async fn scan(
	logger:		tokio::sync::mpsc::Sender<crate::logger::LogMessage>,
	all_gpus:	bool,
	envs:		crate::envs::holder::HoldChannel,
	zink:		bool,
) -> Result<Vec<BindRule>, GPUError> {

	// Block NVIDIA mounts first
	let nv_modules_mount = tokio::task::spawn(nvidia::nvidia_module_mounts(true));


	let devices = {
		match enumerate::enumerate(&logger).await {
			Ok(v)	=> {v}
			Err(e)	=> {
				let _ = logger.send(
					crate::logger::LogMessage {
						level: crate::logger::LogLevel::Warn,
						message: format!("Could not enumerate GPUs: {e:#?}"),
					},
				).await;
				return Ok(vec![]);
			}
		}
	};

	let mut rules = match nv_modules_mount.await {
		Ok(v)	=> {v}
		Err(e)	=> {
			let _ = logger.send(
				crate::logger::LogMessage {
					level: crate::logger::LogLevel::Warn,
					message: format!(
						"Could not apply NVIDIA quirks: {:#?}",
						e,
					),
				}
			).await;
			vec![]
		}
	};

	let mut workers = vec![];

	if zink {
		for dev in &devices {
			match dev.vendor {
				GPUVendor::NVIDIA { driver: _ }	=> {
					envs.send(
						crate::envs::holder::EnvMessage::Add {
							key: "__GLX_VENDOR_LIBRARY_NAME".into(),
							value: "mesa".into()
						}
					)
						.await
						.map_err(GPUError::EnvsError)
						?;
					envs.send(
						crate::envs::holder::EnvMessage::Add {
							key: "MESA_LOADER_DRIVER_OVERRIDE".into(),
							value: "zink".into()
						}
					)
						.await
						.map_err(GPUError::EnvsError)
						?;
					envs.send(
						crate::envs::holder::EnvMessage::Add {
							key: "GALLIUM_DRIVER".into(),
							value: "zink".into()
						}
					)
						.await
						.map_err(GPUError::EnvsError)
						?;
					envs.send(
						crate::envs::holder::EnvMessage::Add {
							key: "LIBGL_KOPPER_DRI2".into(),
							value: "1".into()
						}
					)
						.await
						.map_err(GPUError::EnvsError)
						?;
					envs.send(
						crate::envs::holder::EnvMessage::Add {
							key: "__EGL_VENDOR_LIBRARY_FILENAMES".into(),
							value: "/usr/share/glvnd/egl_vendor.d/50_mesa.json".into()
						}
					)
						.await
						.map_err(GPUError::EnvsError)
						?;
					break;
				}
				_				=> {}
			}
		};
	}

	match all_gpus {
		true	=> {
			for gpu in devices {
				let logger = logger.clone();
				prime::prime_offload_envs(&gpu.vendor, &envs).await;
				workers.push(
					tokio::spawn(
						bind::generate_bind_rules(gpu, logger)
					)
				);
			};
		}
		false	=> {
			for gpu in devices {
				if ! gpu.boot_display {
					continue;
				};
				let logger = logger.clone();
				workers.push(
					tokio::spawn(
						bind::generate_bind_rules(gpu, logger)
					)
				);
			};
		}
	};

	for worker in workers {
		rules.extend(
			worker
				.await
				.map_err(GPUError::Spawn)
				?
		);
	}


	Ok(crate::bind::types::DeDupRules::dedup(rules))

}


pub async fn gputest_print_all_devices(
	tx: &tokio::sync::mpsc::Sender<crate::logger::LogMessage>
) -> String {
	let res = enumerate::enumerate(&tx).await.unwrap();
	format!("{res:#?}")
}

#[derive(Debug, Clone)]
struct GPUDevice {
	card_node:	udev::Device,
	render_node:	udev::Device,
}

#[derive(Debug)]
pub struct GPUInfo {
	boot_display:	bool,
	nodes:		GPUDevice,
	vendor:		GPUVendor,
}

#[derive(Debug)]
pub enum GPUVendor {
	Intel,
	AMD,
	NVIDIA	{driver: nvidia::NVIDIADriver},
	Others,
}

