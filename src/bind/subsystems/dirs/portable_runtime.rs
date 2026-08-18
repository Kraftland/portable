
/**
	Runtime directory for Portable to store data
*/
#[derive(Debug, Clone)]
pub struct PortableRuntime {
	path:	std::sync::Arc<std::path::PathBuf>,
}

impl super::RuntimePathsTrait for PortableRuntime {
	fn new(
		config:		std::sync::Arc<crate::config::config_definition::Config>,
		xdg:		std::sync::Arc<crate::xdg::XdgDirs>,
		instance_id:	std::sync::Arc<String>,
	)	->
			Self
	{
		let mut path = std::path::PathBuf::from(&xdg.runtime);
		path.push("portable");
		path.push(&config.metadata.sandbox_id);
		path.push(instance_id.as_str());
		Self { path: std::sync::Arc::new(path) }
	}

	async fn create_path(&self, stop: std::sync::Arc<crate::stop::Stop>)	->
			Result<(), Self::RuntimePathError>
	{
		tokio::fs::create_dir_all(self.path.clone().as_path())
			.await
			.map_err(Error::CreateError)?;

		let remove_path = self.path.to_path_buf();
		let cancel_token = stop.pre_cancel.clone();

		stop.stop_funcs.send(
			crate::stop::StopMessage::Prepare {
				task:	tokio::spawn(
					async move {
						cancel_token.cancelled().await;

						tokio::fs::remove_dir_all(
							remove_path
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
		self.path.to_path_buf()
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

