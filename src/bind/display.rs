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

pub async fn bind() -> Result<BindRules, DisplayError> {
	#[cfg(feature = "wayland")]
	let wayland_spawn = {
		let info = wayland::Wayland;
		tokio::spawn(async move {
			info
				.bind()
				.await
				.map_err(DisplayError::WaylandError)
		})
	};

	#[cfg(feature = "x11")]
	let x11_spawn = {
		let info = x11::X11;
		tokio::spawn(async move {
			info
				.bind()
				.await
				.map_err(DisplayError::X11Error)
		})
	};

	let mut ret = vec![];

	#[cfg(feature = "wayland")]
	ret.extend(
		wayland_spawn.await.map_err(DisplayError::SpawnError)??
	);

	#[cfg(feature = "x11")]
	ret.extend(
		x11_spawn.await.map_err(DisplayError::SpawnError)??
	);

	Ok(ret)
}
