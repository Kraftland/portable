use thiserror::Error;

#[derive(Error, Debug)]
pub enum InstanceIDError {
	#[error("Could not generate instance ID: spawn error: {0:#?}")]
	SpawnError(tokio::task::JoinError),

	#[cfg(feature = "flatpak")]
	#[error("Could not generate instance ID: error checking Flatpak instance ID collision: {0:#?}")]
	FlatpakIDCollision(std::io::Error),
}

pub async fn generate_instance_id(
	runtime_dir:	&std::path::PathBuf,
	logger:		crate::logger::LogSender,
) -> Result<String, InstanceIDError> {
	let mut rng = super::rng::Rng::new();

	let mut instance_id: u32;

	loop {
		let id = rng.generate();

		#[cfg(feature = "flatpak")]
		let result_flatpak = {
			tokio::spawn(test_flatpak_instance_id(runtime_dir.clone(), id.clone()))
		};

		#[cfg(feature = "flatpak")]
		{
			let res = result_flatpak
				.await
				.map_err(InstanceIDError::SpawnError)?
				?;
			match res {
				true	=> {
					let _ = logger.send(
						crate::logger::LogMessage {
							level: crate::logger::LogLevel::Warn,
							message: format!(
								"Instance ID collided with Flatpak: {}",
								id,
							),
						},
					).await;
					continue;
				}
				false	=> {}
			};
		};

		instance_id = id;
		break;
	};
}

/*
	Test if the instance ID has been used by a Flatpak app
	True means it's occupied
*/
#[cfg(feature = "flatpak")]
async fn test_flatpak_instance_id(
	runtime_dir: std::path::PathBuf,
	id: u32,
) -> Result<bool, InstanceIDError> {
	let mut path = std::path::PathBuf::from(runtime_dir);
	path.push(".flatpak");
	path.push(id.to_string());

	tokio::task::spawn_blocking(|| {
		std::fs::exists(path)
	})
		.await
		.map_err(InstanceIDError::SpawnError)
		?
		.map_err(InstanceIDError::FlatpakIDCollision)
}
