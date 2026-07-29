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



