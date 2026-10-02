/**
	The Input subsystem
*/
pub async fn scan() -> Result<crate::bind::types::BindRules, InputError> {
	use crate::bind::types::BindRule;

	let mut ret = vec![];

	{
		let paths = vec![
			"/sys/class/leds",
			"/sys/class/input",
			"/sys/class/hidraw",
			"/dev/uinput",
		];
		for path in paths {
			let path = std::path::PathBuf::from(path);
			if tokio::fs::try_exists(&path).await.map_err(InputError::IOError)? {
				ret.push(
					BindRule::Path {
						source: path.clone(),
						dest: path,
						class: crate::bind::types::BindType::Device,
					}
				);
			}
		};
	};

	let devices = {
		let input = super::enumerate(
			crate::bind::subsystems::devices::Filter::Subsystem {
				subsystem: "input".to_string(),
			},
		)
			.await
			.map_err(InputError::EnumerateError)
			?;

		let hid = super::enumerate(
			crate::bind::subsystems::devices::Filter::Subsystem {
				subsystem: "hid".to_string(),
			},
		)
			.await
			.map_err(InputError::EnumerateError)
			?;

		let hidraw = super::enumerate(
			crate::bind::subsystems::devices::Filter::Subsystem {
				subsystem: "hidraw".to_string(),
			},
		)
			.await
			.map_err(InputError::EnumerateError)
			?;

		let mut devices = vec![];
		devices.extend(input);
		devices.extend(hid);
		devices.extend(hidraw);
		devices
	};

	for device in devices {
		ret.extend(super::bind_udev_device(&device).await);
	};

	Ok(crate::bind::types::DeDupRules::dedup(ret))
}

#[derive(thiserror::Error, Debug)]
pub enum InputError {
	#[error("Could not determine if path exists: {0}")]
	IOError(std::io::Error),

	#[error("Could not determine if path exists: error spawning task: {0:#?}")]
	SpawnError(tokio::task::JoinError),

	#[error("Could not enumerate input devices: {0:#?}")]
	EnumerateError(super::EnumerateError),
}
