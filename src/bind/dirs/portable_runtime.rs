/**
	Runtime directory for Portable to store data
*/
pub struct PortableRuntime {
	path:	std::path::PathBuf,
}

impl super::RuntimePaths for PortableRuntime {
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
		Self { path }
	}

	async fn create_path(&self)		->
			Result<(), Self::RuntimePathError>
	{
		tokio::fs::create_dir_all(&self.path)
			.await
			.map_err(Error::CreateError)
	}

	type RuntimePathError = Error;
}

#[derive(thiserror::Error, Debug)]
pub enum Error {
	#[error("Could not spawn task: {0:#?}")]
	SpawnError(tokio::task::JoinError),

	#[error("Could not create directory: {0:#?}")]
	CreateError(std::io::Error),
}
