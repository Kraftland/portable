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
			"/dev/input",
			"/dev/uinput",
		];
		for path in paths {
			let path = std::path::PathBuf::from(path);
			if exists(path.clone()).await? {
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
	#[error("Could not determine if path exists")]
	IOError(std::io::Error),

	#[error("Could not determine if path exists: error spawning task: {0:#?}")]
	SpawnError(tokio::task::JoinError),

	#[error("Could not enumerate input devices: {0:#?}")]
	EnumerateError(super::EnumerateError),
}

/**
	Whether the path exists on filesystem
*/
pub async fn exists(path: std::path::PathBuf) -> Result<bool, InputError> {
	tokio::task::spawn_blocking(|| {
		std::fs::exists(path).map_err(InputError::IOError)
	}).await.map_err(InputError::SpawnError)?
}
