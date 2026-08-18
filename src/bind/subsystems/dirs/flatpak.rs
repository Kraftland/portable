/**
	Runtime directory to mimic Flatpak
*/
pub struct FlatpakRuntime {
	appid_path:	std::sync::Arc<std::path::PathBuf>,

	instance_path:	std::sync::Arc<std::path::PathBuf>,

	tmp_subdir:	std::sync::Arc<std::path::PathBuf>,
}

impl super::RuntimePathsTrait for FlatpakRuntime {
	fn new(
		config:		std::sync::Arc<crate::config::config_definition::Config>,
		xdg:		std::sync::Arc<crate::xdg::XdgDirs>,
		instance_id:	std::sync::Arc<String>,
	)	->
			Self
	{
		let mut base_path = std::path::PathBuf::from(&xdg.runtime);
		base_path.push(".flatpak");

		let mut appid_path = std::path::PathBuf::from(&base_path);
		appid_path.push(&config.metadata.sandbox_id);

		let mut tmp_path = std::path::PathBuf::from(&appid_path);
		tmp_path.push("tmp");

		let mut instance_path = std::path::PathBuf::from(&base_path);
		instance_path.push(instance_id.as_str());

		Self {
			appid_path:	std::sync::Arc::new(appid_path),
			instance_path:	std::sync::Arc::new(instance_path),
			tmp_subdir:	std::sync::Arc::new(tmp_path),
		}
	}

	async fn create_path(&self, stop: std::sync::Arc<crate::stop::Stop>)	->
			Result<(), Self::RuntimePathError>
	{

		// tmp_subdir is inside appid_path
		tokio::fs::create_dir_all(self.tmp_subdir.as_path())
			.await
			.map_err(Error::CreateError)?;
		let path_clone = self.appid_path.to_path_buf();

		let pre_cancel = stop.pre_parent.child_token();

		stop.stop_funcs.send(
			crate::stop::StopMessage::Prepare {
				task:	tokio::spawn(
					async move {
						pre_cancel.cancelled().await;

						tokio::fs::remove_dir_all(
							path_clone
						)
							.await
							.map_err(crate::stop::StopError::RemoveFsError)
					}
				),
			}
		)
			.map_err(Error::StopError)
			?;


		tokio::fs::create_dir_all(self.instance_path.as_path())
			.await
			.map_err(Error::CreateError)?;
		let path_clone = self.instance_path.to_path_buf();
		let pre_cancel = stop.pre_parent.child_token();
		stop.stop_funcs.send(
			crate::stop::StopMessage::Prepare {
				task:	tokio::spawn(
					async move {
						pre_cancel.cancelled().await;

						tokio::fs::remove_dir_all(
							path_clone
						)
							.await
							.map_err(crate::stop::StopError::RemoveFsError)
					}
				),
			}
		)
			.map_err(Error::StopError)
			?;

		Ok(())
	}

	fn path(&self) -> std::path::PathBuf {
		self.instance_path.to_path_buf()
	}

	type RuntimePathError = Error;
}

#[derive(thiserror::Error, Debug)]
pub enum Error {
	#[error("Could not spawn task: {0:#?}")]
	SpawnError(tokio::task::JoinError),

	#[error("Could not create directory: {0:#?}")]
	CreateError(std::io::Error),

	#[error("Could not contact stop worker: {0:#?}")]
	StopError(tokio::sync::mpsc::error::SendError<crate::stop::StopMessage>),
}

