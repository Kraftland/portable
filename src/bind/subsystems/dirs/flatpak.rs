/**
	Runtime directory to mimic Flatpak
*/
pub struct FlatpakRuntime {
	appid_path:	std::path::PathBuf,

	instance_path:	std::path::PathBuf,

	tmp_subdir:	std::path::PathBuf,
}

impl super::RuntimePathsTrait for FlatpakRuntime {
	fn new(
		config:		&crate::config::Config,
		xdg:		&crate::xdg::XdgDirs,
		instance_id:	&str,
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
		instance_path.push(instance_id);

		Self {
			appid_path:	appid_path,
			instance_path:	instance_path,
			tmp_subdir: 	tmp_path,
		}
	}

	async fn create_path(&self, stop: tokio::sync::mpsc::Sender<crate::stop::StopFunc>)	->
			Result<(), Self::RuntimePathError>
	{

		// tmp_subdir is inside appid_path
		tokio::fs::create_dir_all(&self.tmp_subdir)
			.await
			.map_err(Error::CreateError)?;
		let path_clone = self.appid_path.clone();
		stop.send(
			crate::stop::StopFunc {
				layer: crate::stop::FunctionLayer::Pre,
				function: Box::new(|| {
					match std::fs::remove_dir_all(path_clone) {
						Ok(_)	=> {}
						Err(e)	=> {
							eprintln!("Could not remove runtime directory: {e:#?}")
						}
					};
				}),
			},
		).await.map_err(Error::StopError)?;


		tokio::fs::create_dir_all(&self.instance_path)
			.await
			.map_err(Error::CreateError)?;
		let path_clone = self.instance_path.clone();
		stop.send(
			crate::stop::StopFunc {
				layer: crate::stop::FunctionLayer::Pre,
				function: Box::new(|| {
					match std::fs::remove_dir_all(path_clone) {
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
		std::path::PathBuf::from(&self.instance_path)
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
