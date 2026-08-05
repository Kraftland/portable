use crate::bind::types::BindRules;

#[derive(thiserror::Error, Debug)]
pub enum DisplayError {
	#[error("Could not bind display: spawn error: {0:#?}")]
	SpawnError(tokio::task::JoinError),

	#[cfg(feature = "wayland")]
	#[error("Could not bind Wayland display: {0:#?}")]
	WaylandError(wayland::DisplayBindError),

	#[cfg(feature = "x11")]
	#[error("Could not bind X11 display: {0:#?}")]
	X11Error(x11::DisplayBindError),
}

#[cfg(feature = "x11")]
pub mod x11;

#[cfg(feature = "wayland")]
pub mod wayland;

/**
	The BindDisplay trait unifies display binding across different APIs
*/
pub trait BindDisplay {
	fn bind(self) -> impl std::future::Future<Output = Result<crate::bind::types::BindRules, Self::DisplayBindError>> + Send;

	type DisplayBindError;
}

#[derive(thiserror::Error, Debug)]
pub enum ExistError {
	#[error("Could not determine if path exists")]
	IOError(std::io::Error),

	#[error("Could not determine if path exists: error spawning task: {0:#?}")]
	SpawnError(tokio::task::JoinError),
}

/**
	Whether the socket or file exists on filesystem
*/
pub async fn exists(path: std::path::PathBuf) -> Result<bool, ExistError> {
	tokio::task::spawn_blocking(|| {
		std::fs::exists(path).map_err(ExistError::IOError)
	}).await.map_err(ExistError::SpawnError)?
}

pub async fn bind(
	logger:		crate::logger::LogSender,
	home:		std::path::PathBuf,
	env:		crate::envs::holder::HoldChannel,

	// socket enablement below
	x11:		bool,
	wayland:	bool,
) -> Result<BindRules, DisplayError> {
	let mut spawn_collector = vec![];

	/**
		We use this variable to avoid applying Input Method workaround multiple times
	*/
	let mut ime_applied: bool;

	#[cfg(feature = "x11")]
	if x11 {
		let info = x11::X11 {
			logger:	logger.clone(),
			home:	home,
			env:	env.clone(),
		};
		spawn_collector.push(
			tokio::spawn(async move {
				info
					.bind()
					.await
					.map_err(DisplayError::X11Error)
			})
		);
	};

	#[cfg(feature = "wayland")]
	if wayland {
		let info = wayland::Wayland;
		spawn_collector.push(
			tokio::spawn(async move {
				info
					.bind()
					.await
					.map_err(DisplayError::WaylandError)
			})
		);
	};

	let mut ret = vec![];

	for spawn in spawn_collector {
		ret.extend(
			spawn
				.await
				.map_err(DisplayError::SpawnError)
				?
				?
		);
	};

	Ok(ret)
}
