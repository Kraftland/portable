/*
	Scan /dev/ for nvidia device nodes that aren't in udev database
*/
pub fn get_nvidia_devices(
	logger:		&tokio::sync::mpsc::Sender<crate::logger::LogMessage>,
) -> Vec<std::path::PathBuf> {
	use crate::logger::LogMessage;
	use crate::logger::LogLevel;
	let dir = {
		let dir = std::fs::read_dir("/dev");
		match dir {
			Ok(v)	=> {v}
			Err(e)	=> {
				let _ = logger.blocking_send(
					LogMessage {
						level: LogLevel::Warn,
						message: format!("Could not read /dev: {e:#?}"),
					}
				);
				return vec![];
			}
		}
	};

	let mut ret = vec![];

	for entry in dir {
		match entry {
			Ok(v)	=> {
				if v.file_name().to_string_lossy().starts_with("nvidia") {
					ret.push(v.path());
				}
			}
			Err(e)	=> {
				let _ = logger.blocking_send(
					LogMessage {
						level: LogLevel::Warn,
						message: format!("Could not read /dev entry: {e:#?}"),
					}
				);
			}
		}
	};
	ret
}

#[derive(Debug)]
pub enum NVIDIADriver {
	Nouveau,
	Proprietary,
	Unknown		{driver: Option<String>},
}

impl NVIDIADriver {
	pub fn get(device: &udev::Device)	-> Self {
		// eprintln!("Device {0:?} has driver {1:?}", device.syspath(), device.driver());
		let driver = match device.driver() {
			Some(v)	=> {v.to_owned()}
			None	=> {
				let parent = match device.parent() {
					Some(v)	=> {v}
					None	=> {
						return NVIDIADriver::Unknown {
							driver: None,
						};
					}
				};
				match parent.driver() {
					Some(v)	=> {v.to_owned()}
					None	=> {
						return NVIDIADriver::Unknown {
							driver: None,
						};
					}
				}
			}
		};
		match driver.to_str() {
			Some("nvidia")	=> NVIDIADriver::Proprietary,
			Some("nouveau")	=> NVIDIADriver::Nouveau,
			Some(v)		=> NVIDIADriver::Unknown { driver: Some(v.to_string()) },
			None		=> NVIDIADriver::Unknown { driver: Some(format!("Invalid unicode")) },
		}
	}
}


/**
	Block or mount NVIDIA module paths

	When blocked, prevents accidental discrete GPU wake up.
*/
pub async fn nvidia_module_mounts(block: bool) -> Vec<crate::bind::types::BindRule> {
	use crate::bind::types::BindRule;
	let paths = vec![
		"/sys/module/nvidia",
		"/sys/module/nvidia_drm",
		"/sys/module/nvidia_modeset",
		"/sys/module/nvidia_uvm",
		"/sys/module/nvidia_wmi_ec_backlight",
	];

	let mut ret = vec![];

	for path in paths {
		if ! tokio::fs::try_exists(path).await.unwrap_or(false) {
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
