
/**
	Runtime directory for Portable to store data
*/
#[derive(Debug, Clone)]
pub struct PortableRuntime {
	path:	std::sync::Arc<std::path::PathBuf>,
}

impl super::RuntimePathsTrait for PortableRuntime {
	fn new(
		config:		&crate::config::Config,
		xdg:		&crate::xdg::XdgDirs,
		instance_id:	&str,
	)	->
			Self
	{
		let mut path = std::path::PathBuf::from(&xdg.runtime);
		path.push("portable");
		path.push(&config.metadata.sandbox_id);
		path.push(instance_id);
		Self { path: std::sync::Arc::new(path) }
	}

	async fn create_path(&self, stop: tokio::sync::mpsc::Sender<crate::stop::StopFunc>)	->
			Result<(), Self::RuntimePathError>
	{
		tokio::fs::create_dir_all(self.path.clone().as_path())
			.await
			.map_err(Error::CreateError)?;

		let remove_path = self.path.clone();

		stop.send(
			crate::stop::StopFunc {
				layer: crate::stop::FunctionLayer::Pre,
				function: Box::new(move || {
					match std::fs::remove_dir_all(remove_path.as_path()) {
						Ok(_)	=> {}
						Err(e)	=> {
							eprintln!("Could not remove runtime directory: {e:#?}")
						}
					};
				}),
			},
		).await.map_err(Error::StopError)?;

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
	StopError(tokio::sync::mpsc::error::SendError<crate::stop::StopFunc>),
}

