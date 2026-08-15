use thiserror::Error;

#[cfg(feature = "gpu")]
pub mod gpu;

#[cfg(feature = "camera")]
pub mod camera;

#[cfg(feature = "input")]
mod input;

/**
	Implementation of Devices subsystem
*/
#[derive(Debug)]
pub struct Devices {
	pub all_gpus:		bool,
	pub bind_camera:	bool,
	pub bind_input:		bool,
	pub logger:		tokio::sync::mpsc::Sender<crate::logger::LogMessage>,
}

impl super::GenerateBind for Devices {
	async fn bind(self) -> Result<crate::bind::types::BindRules, Self::BindError> {
		let mut tasks = vec![];

		let logger_clone = self.logger.clone();

		#[cfg(feature = "gpu")]
		tasks.push(
			tokio::spawn(
				async move {
					gpu::scan(
						logger_clone,
						self.all_gpus,
					)
					.await
					.map_err(DeviceError::GPUError)
				}
			),
		);

		#[cfg(feature = "input")]
		if self.bind_input {
			tasks.push(
				tokio::spawn(
					async move {
						input::scan()
							.await
							.map_err(DeviceError::InputError)
					}
				)
			);
		};

		#[cfg(feature = "camera")]
		if self.bind_camera {
			tasks.push(
				tokio::spawn(
					async move {
						camera::scan()
						.await
						.map_err(DeviceError::CameraError)
					}
				)
			);
		}


		let mut ret = vec![];
		for task in tasks {
			ret.extend(
				task
					.await
					.map_err(DeviceError::Spawn)
					?
					?
				);
		};

		Ok(ret)
	}

	type BindError = DeviceError;
}

#[derive(Debug, Error)]
pub enum DeviceError {
	#[error("Could not handle GPU devices: {0:#?}")]
	GPUError(gpu::GPUError),

	#[error("Could not handle Camera devices: {0:#?}")]
	CameraError(camera::CameraError),

	#[error("Could not handle Input devices: {0:#?}")]
	InputError(input::InputError),

	#[error("Could not spawn task: {0:#?}")]
	Spawn(tokio::task::JoinError),
}

#[derive(Debug, Error)]
pub enum EnumerateError {
	#[error("Could not enumerate devices: create enumerator failed: {0:#?}")]
	CreateEnumeratorError(String),
	#[error("Could not enumerate devices: add match failed: {0:#?}")]
	AddMatchError(std::io::Error),
	#[error("Could not enumerate devices: scan failed: {0:#?}")]
	ScanError(std::io::Error),
}

#[derive(Debug)]
pub enum Filter {
	// Enumerate by subsystem, this implies initialised
	Subsystem {subsystem: String},

	// Enumerate by subsystem with DEVTYPE, this implies initialised
	SubsystemWithDevtype {subsystem: String, devtype: String},
}

pub async fn enumerate(filter: Filter) -> Result<Vec<udev::Device>, EnumerateError> {
	let mut enumerator = match filter {
		Filter::Subsystem { subsystem }				=> {
			let mut enumerator = {
				match udev::Enumerator::new() {
					Ok(v)	=> {v}
					Err(e)	=> {
						return Err(
							EnumerateError::CreateEnumeratorError(
								format!("{e:#?}"),
							),
						);
					}
				}
			};
			enumerator
				.match_is_initialized()
				.map_err(EnumerateError::AddMatchError)
				?;
			enumerator
				.match_subsystem(subsystem)
				.map_err(EnumerateError::AddMatchError)
				?;

			enumerator
		}

		Filter::SubsystemWithDevtype { subsystem, devtype }	=> {
			let mut enumerator = {
				match udev::Enumerator::new() {
					Ok(v)	=> {v}
					Err(e)	=> {
						return Err(
							EnumerateError::CreateEnumeratorError(
								format!("{e:#?}"),
							),
						);
					}
				}
			};
			enumerator
				.match_is_initialized()
				.map_err(EnumerateError::AddMatchError)
				?;
			enumerator
				.match_subsystem(subsystem)
				.map_err(EnumerateError::AddMatchError)
				?;
			enumerator
				.match_property("DEVTYPE", devtype)
				.map_err(EnumerateError::AddMatchError)
				?;
			enumerator
		}
	};
	let list = enumerator
		.scan_devices()
		.map_err(EnumerateError::ScanError)?;
	let mut ret = vec![];
	for dev in list {
		ret.push(dev);
	};
	Ok(ret)
}

async fn bind_udev_device(device: udev::Device) -> Vec<crate::bind::types::BindRule> {
	use crate::bind::types::BindType;
	use crate::bind::types::BindRule;

	let mut ret = vec![];

	{
		let devlink = device
			.property_value("DEVLINKS")
			.unwrap_or(std::ffi::OsStr::new(""))
			.to_os_string();

		let devlinks_string = devlink
			.to_str()
			.unwrap_or("");

		for link in devlinks_string.split(" ") {
			ret.push(
				BindRule::Path {
					source: std::path::PathBuf::from(link),
					dest: std::path::PathBuf::from(link),
					class: BindType::Device,
				},
			);
		};
	};

	match device.devnode() {
		Some(v)	=> {
			ret.push(
				BindRule::Path {
					source: v.to_path_buf(),
					dest: v.to_path_buf(),
					class: BindType::Device,
				}
			);
		}
		None	=> {}
	};



	{
		let devpath = std::path::PathBuf::from(device.syspath());
		let path = std::path::PathBuf::from("/sys");
		let path = path.join(devpath);
		ret.push(
			BindRule::Path {
				source: path.clone(),
				dest: path,
				class: BindType::Device,
			},
		);
	};

	ret
}
