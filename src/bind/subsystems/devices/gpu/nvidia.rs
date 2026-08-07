/*
	Scan /dev/ for nvidia device nodes that aren't in udev database
	This function needs spawn_blocking for std I/O
*/
async fn get_nvidia_devices(
	logger:		&tokio::sync::mpsc::Sender<crate::logger::LogMessage>,
) -> Vec<std::path::PathBuf> {
	use crate::logger::LogMessage;
	use crate::logger::LogLevel;
	let dir = {
		let dir = std::fs::read_dir("/dev");
		match dir {
			Ok(v)	=> {v}
			Err(e)	=> {
				let _ = logger.send(
					LogMessage {
						level: LogLevel::Warn,
						message: format!("Could not read /dev: {e:#?}"),
					}
				).await;
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
				let _ = logger.send(
					LogMessage {
						level: LogLevel::Warn,
						message: format!("Could not read /dev entry: {e:#?}"),
					}
				).await;
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
		let driver = match device.driver() {
			Some(v)	=> {v}
			None	=> {return NVIDIADriver::Unknown { driver: None };}
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
	This needs spawn_blocking
*/
pub fn nvidia_module_mounts(block: bool) -> Vec<crate::bind::types::BindRule> {
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
		if ! super::path_exists(&path.into()) {
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
