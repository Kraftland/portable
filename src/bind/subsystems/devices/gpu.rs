use thiserror::Error;

use crate::bind::types::BindRule;

pub mod nvidia;
pub mod prime;

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
}



pub async fn scan(
	logger:		tokio::sync::mpsc::Sender<crate::logger::LogMessage>,
	all_gpus:	bool,
) -> Result<Vec<BindRule>, GPUError> {

	// Block NVIDIA mounts first
	let nv_modules_mount = tokio::task::spawn_blocking(|| {
		nvidia_module_mounts(true)
	});


	let devices = {
		enumerate_gpus(&logger)
		.await
		?
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

	if all_gpus {
		for gpu in devices {
			rules.extend(generate_bind_rules(gpu, &logger).await);
		};
	} else {
		for gpu in devices {
			if gpu.boot_display {
				rules.extend(generate_bind_rules(gpu, &logger).await);
			}
		}
	}


	Ok(rules)

}

async fn generate_bind_rules(
	gpu:		GPUInfo,
	logger:		&tokio::sync::mpsc::Sender<crate::logger::LogMessage>,
) -> Vec<BindRule> {
	let (tx, mut rx) = tokio::sync::mpsc::unbounded_channel();

	/*
		sys/class/drm/ does not seem to be covered by udev
	*/

	{
		let drm_path = std::path::PathBuf::from("/sys/class/drm");
		match gpu.nodes.card_node {
			Some(ref v)	=> {
				let mut card_path = drm_path.clone();
				card_path.push(v.sysname());
				tx.send(
					BindRule::Path {
						source: card_path.clone(),
						dest: card_path,
						class: crate::bind::types::BindType::Device,
					},
				).expect("Error sending drm bind rules");
			}
			None	=> {
				let _ = logger.send(
					crate::logger::LogMessage {
						level: crate::logger::LogLevel::Warn,
						message: format!("Missing card node: {:#?}", gpu),
					},
				).await;
			}
		};
		match gpu.nodes.render_node {
			Some(ref v)	=> {
				let mut card_path = drm_path.clone();
				card_path.push(v.sysname());
				tx.send(
					BindRule::Path {
						source: card_path.clone(),
						dest: card_path,
						class: crate::bind::types::BindType::Device,
					},
				).expect("Error sending drm bind rules");
			}
			None	=> {
				let _ = logger.send(
					crate::logger::LogMessage {
						level: crate::logger::LogLevel::Warn,
						message: format!("Missing renderer node: {:#?}", gpu),
					},
				).await;
			}
		}
	};

	match gpu.vendor {
		GPUVendor::AMD			=> {
			tx.send(
				BindRule::Path {
					source: std::path::PathBuf::from("/dev/kfd"),
					dest: std::path::PathBuf::from("/dev/kfd"),
					class: crate::bind::types::BindType::Device,
				},
			).unwrap();
		}

		GPUVendor::Intel		=> {
			// Intel doesn't need any new tricks, yet
		}
		GPUVendor::NVIDIA {driver}	=> {
			match driver {
				nvidia::NVIDIADriver::Nouveau			=> {
					// TODO: what about nouveau's modules?
				}
				nvidia::NVIDIADriver::Proprietary		=> {

				}
				nvidia::NVIDIADriver::Unknown { driver }	=> {
					let _ = logger.send(
						crate::logger::LogMessage {
							level: crate::logger::LogLevel::Warn,
							message: format!("Unknown nvidia driver: {driver:?}"),
						},
					).await;
				}
			}

			let nv_modules_mount = tokio::task::spawn_blocking(|| {
				nvidia_module_mounts(false)
			});

			let nv_modules_mount = match nv_modules_mount.await {
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
					return vec![];
				}
			};
			for rule in nv_modules_mount {
				tx.send(rule).unwrap();
			};
		}

		GPUVendor::Others	=> {
			let _ = logger.send(
				crate::logger::LogMessage {
					level: crate::logger::LogLevel::Warn,
					message: format!("Could not apply GPU quirks: unknown vendor"),
				}
			).await;
		}
	};

	rx.close();

	let mut ret = vec![];

	loop {
		let msg = rx.recv().await;
		match msg {
			Some(v)	=> {
				ret.push(v);
			}
			None	=> {
				break;
			}
		}
	}
	ret
}

pub async fn gputest_print_all_devices(
	tx: &tokio::sync::mpsc::Sender<crate::logger::LogMessage>
) -> String {
	let res = enumerate_gpus(&tx).await.unwrap();
	format!("{res:#?}")
}


/**
	This needs spawn_blocking
*/
fn nvidia_module_mounts(block: bool) -> Vec<BindRule> {
	let paths = vec![
		"/sys/module/nvidia",
		"/sys/module/nvidia_drm",
		"/sys/module/nvidia_modeset",
		"/sys/module/nvidia_uvm",
		"/sys/module/nvidia_wmi_ec_backlight",
	];

	let mut ret = vec![];

	for path in paths {
		if ! path_exists(&path.into()) {
			continue;
		}
		match block {
			true	=> {
				ret.push(
					BindRule::VirtualFS {
						dest: path.into(),
						class: crate::bind::types::VirtualFS::Tmpfs {
							size_mb: Some(0),
							perms: None,
						},
					}
				);
			}
			false	=> {
				ret.push(
					BindRule::Path {
						source: path.into(),
						dest: path.into(),
						class: crate::bind::types::BindType::Device,
					},
				);
			}
		}
	};
	ret
}

fn path_exists(path: &std::path::PathBuf) -> bool {
	match std::fs::exists(path) {
		Ok(v)	=> v,
		Err(_)	=> false,
	}
}




#[derive(Debug, Clone)]
struct GPUDevice {
	card_node:	Option<udev::Device>,
	render_node:	Option<udev::Device>,
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

async fn get_vendor(device: udev::Device) -> GPUVendor {
	match device.attribute_value("vendor") {
		Some(v)	=> {
			return map_to_vendor(v, &device)
		}
		None	=> {}
	};

	let parent = match device.parent() {
		Some(v)	=> {v}
		None	=> {return GPUVendor::Others}
	};

	match parent.attribute_value("vendor") {
		Some(v)	=> {
			return map_to_vendor(v, &device);
		}
		None	=> {
			return GPUVendor::Others;
		}
	}
}

fn map_to_vendor(vendor_string: &std::ffi::OsStr, device: &udev::Device) -> GPUVendor {
	let string = vendor_string.to_str().unwrap_or("unknown");
	match string {
		"0x8086"	=> {GPUVendor::Intel}
		"0x10de"	=> {
			GPUVendor::NVIDIA {
				driver: nvidia::NVIDIADriver::get(device),
			}
		}
		"0x1002"	=> {GPUVendor::AMD}
		_		=> {GPUVendor::Others}
	}
}

/**
	Eumerates all graphics cards (and renderer nodes, paired together) as vectors of udev devices
	See GPUDevice struct for more details
	Errors needs to be handled gracefully.
*/
async fn enumerate_gpus(
	logger:		&tokio::sync::mpsc::Sender<crate::logger::LogMessage>,
) -> Result<Vec<GPUInfo>, GPUError> {
	let devices = crate::bind::subsystems::devices::enumerate(
		super::Filter::SubsystemWithDevtype {
			subsystem: "drm".to_string(),
			devtype: "drm_minor".to_string(),
		},
	)
		.await
		.map_err(GPUError::Enumerate)
		?;

	let _ = logger.send(
		crate::logger::LogMessage {
			level: crate::logger::LogLevel::Debug,
			message: format!("Udev returned {} cards and nodes", devices.len())
		},
	).await;

	let devices = associate_card_render(devices, logger).await;

	let (tx, mut rx) = tokio::sync::mpsc::unbounded_channel();
	let tracker = tokio_util::task::TaskTracker::new();

	for dev in devices {
		let tx_clone = tx.clone();
		let log_clone = logger.clone();
		// let dev_clone = dev.clone();
		tracker.spawn(async move {
			// let dev = dev.to_owned();
			if dev.card_node.is_none() || dev.render_node.is_none() {
				let _ = log_clone.send(
					crate::logger::LogMessage {
						level: crate::logger::LogLevel::Warn,
						message: format!("Could not identify GPU: {:?}", dev)
					}
				).await;
				return;
			};

			// Unwrap is then safe

			let vendor_spawn = tokio::task::spawn(
				get_vendor(dev.render_node.clone().unwrap())
			);

			let boot_display = {
				let sysname = dev.card_node.as_ref().unwrap().sysname();
				match device_is_boot_display(&dev.card_node.as_ref().unwrap()) {
					Ok(v)	=> {
						let _ = log_clone.send(
							crate::logger::LogMessage {
							level: crate::logger::LogLevel::Debug,
							message: format!(
								"{:?} is a boot display: {}",
								sysname,
								v,
							) }
						).await;
						v
					}
					Err(e)	=> {
						match e {
							GPUError::InvalidBootVGA(_)	=> {
								false
							}

							_				=> {
								let _ = log_clone.send(
									crate::logger::LogMessage {
									level: crate::logger::LogLevel::Warn,
									message: format!(
						"Could not determine boot display status for {:?}: {:#?}",
						sysname,
						e,
										)
									}
								).await;
								false
							}
						}

					}
				}
			};

			let vendor = {
				let vendor = vendor_spawn.await;
				match vendor {
					Ok(v)	=> {v}
					Err(e)	=> {
						let _ = log_clone.send(
							crate::logger::LogMessage {
								level: crate::logger::LogLevel::Warn,
								message: format!(
									"Could not parse GPU vendor: {:?}",
									e,
								),
							}
						).await;

						GPUVendor::Others
					}
				}
			};

			let _ = tx_clone.send(
				GPUInfo {
					boot_display:	boot_display,
					vendor:		vendor,
					nodes:		dev,
				}
			);
		});
	};

	tracker.close();
	tracker.wait().await;

	rx.close();

	let mut ret = vec![];

	loop {
		match rx.recv().await {
			Some(v)	=> {
				ret.push(v);
			}
			None	=> {
				break;
			}
		}
	}
	Ok(ret)
}

/**
	Associates the card device with renderer, using the GPUDevice struct above
	Internally uses the ID_PATH approach just like the previous impl
*/
async fn associate_card_render(
	devices:	Vec<udev::Device>,
	logger:		&tokio::sync::mpsc::Sender<crate::logger::LogMessage>,
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
fn device_is_boot_display(card_device: &udev::Device) -> Result<bool, GPUError> {
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
