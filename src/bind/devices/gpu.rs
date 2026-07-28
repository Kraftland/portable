use thiserror::Error;

use crate::bind::types::BindRule;

#[derive(Error, Debug)]
pub enum GPUError {
	#[error("Could not determine boot display: invalid value {0:?}")]
	InvalidBootDisplay(String),
	#[error("Could not determine boot vga: invalid value {0:?}")]
	InvalidBootVGA(String),

	#[error("Could not enumerate GPUs: {0:#?}")]
	Enumerate(crate::bind::devices::EnumerateError),
}

// pub async fn scan(all_gpus: bool) -> Result<Vec<BindRule>, GPUError> {

// }


struct GPUDevice {
	card_node:	Option<udev::Device>,
	render_node:	Option<udev::Device>,
}

struct GPUInfo {
	boot_display:	bool,
	nodes:		GPUDevice,
}

/*
	Eumerates all graphics cards (and renderer nodes, paired together) as vectors of udev devices
	See GPUDevice struct for more details
	Errors needs to be handled gracefully.
*/
// async fn enumerate_gpus() -> Result<Vec<GPUInfo>, GPUError> {
// 	let devices = crate::bind::devices::enumerate(
// 		super::Filter::Subsystem { subsystem: "drm".to_string() },
// 	)
// 		.await
// 		.map_err(GPUError::Enumerate)
// 		?;

// }

/*
	Associates the card device with renderer, using the GPUDevice struct above
	Internally uses the ID_PATH approach just like the previous impl
*/
async fn associate_card_render(
	devices: Vec<udev::Device>,
	logger: tokio::sync::mpsc::Sender<crate::logger::LogMessage>,
) -> Vec<GPUDevice> {
	// The map is for holding (card, renderer)
	let mut map: std::collections::HashMap
		<std::ffi::OsString, (Option<udev::Device>, Option<udev::Device>)>
		= std::collections::HashMap::new();
	for dev in devices {
		let id_path = {
			match dev.property_value("ID_PATH") {
				Some(v)	=> {
					v
				}
				None	=> {
					let _ = logger.send(
						crate::logger::LogMessage {
							level: crate::logger::LogLevel::Warn,
							message: format!("GPU {:?} does not have ID_PATH property", dev.sysname())
						}
					).await;
					continue;
				}
			}
		};

		let device_type = match card_type(&dev).await {
			Some(v)	=> {v}
			None	=> {
				crate::logger::LogMessage {
					level:		crate::logger::LogLevel::Warn,
					message:	format!("GPU {:?} has unknown type", dev.sysname())
				};
				continue;
			}
		};

		if map.contains_key(id_path) {
			// unwrap is safe here
			let mut value = map.get(id_path).unwrap().to_owned();
			let updated = match device_type {
				NodeType::Card		=> {
					value.0 = Some(dev.to_owned());
					value
				}
				NodeType::Renderer	=> {
					value.1 = Some(dev.to_owned());
					value
				}
			};
			map.insert(id_path.into(), updated.clone());
		} else {
			map.insert(
				id_path.into(),
				match device_type {
					NodeType::Card		=> {
						(Some(dev), None)
					}
					NodeType::Renderer	=> {
						(None, Some(dev))
					}
				},
			);
		};
	};
	let mut ret = vec![];
	for (_k, v) in map {
		let _ = logger.send(
			crate::logger::LogMessage {
				level: crate::logger::LogLevel::Debug,
				message: format!(
					"Bound GPU card {:?} to renderer node {:?}",
					v.0,
					v.1,
				)
			}
		).await;
		ret.push(
			GPUDevice {
				card_node:	v.0,
				render_node:	v.1
			}
		);
	};
	ret
}

enum NodeType {
	Card,
	Renderer,
}

async fn card_type (device: &udev::Device) -> Option<NodeType> {
	let sys_name = device.sysname();
	let sys_name = match sys_name.to_str() {
		Some(v)	=> {v}
		None	=> {return None}
	};

	if sys_name.starts_with("card") {
		return Some(NodeType::Card)
	} else if sys_name.starts_with("render") {
		return Some(NodeType::Renderer);
	} else {
		return None;
	}
}

/*
	Check if a device is connected to boot display.
	There is a udev attribute boot_display implemented in
		https://github.com/torvalds/linux/blob/master/drivers/gpu/drm/drm_sysfs.c
	Which can be more reliable and faster than querying device connector status ourselves.
	The attribute is exported within DEVPATH/boot_display when true

	The error needs to be handled gracefully.

	We will see whether this wakes up discrete GPU, if it does, then read the file manually
*/
async fn device_is_boot_display(card_device: &udev::Device) -> Result<bool, GPUError> {
	let boot_display_attr_value = card_device.attribute_value("boot_display");
	match boot_display_attr_value {
		Some(v)	=> {
			if v == "1" {
				return Ok(true)
			} else {
				return Err(
					GPUError::InvalidBootDisplay(format!("{v:?}"))
				);
			}
		}
		None	=> {}
	};

	let parent_device = {
		let parent_device = card_device.parent();
		match parent_device {
			Some(v)	=> {v}
			None	=> {
				return Ok(false);
			}
		}
	};

	match parent_device.attribute_value("boot_vga") {
		Some(v)	=> {
			if v == "1" {
				Ok(true)
			} else {
				Err(
					GPUError::InvalidBootVGA(format!("{v:?}"))
				)
			}
		}
		None	=> {
			Ok(false)
		}
	}
}
