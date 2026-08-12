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

pub struct Display {
	pub xdg:		std::sync::Arc<crate::xdg::XdgDirs>,

	pub logger:		crate::logger::LogSender,
	pub env:		crate::envs::holder::HoldChannel,

	pub portable_runtime:	crate::bind::subsystems::dirs::portable_runtime::PortableRuntime,
	pub app_id:		String,
	pub instance_id:	String,

	// socket enablement below
	pub x11:		bool,
	pub wayland:		bool,
}

impl super::GenerateBind for Display {
	async fn bind(self) -> Result<crate::bind::types::BindRules, Self::BindError> {
		bind(
			self.xdg,
			self.logger,
			self.env,
			self.portable_runtime,
			self.app_id,
			self.instance_id,
			self.x11,
			self.wayland,
		).await
	}
	type BindError = DisplayError;
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

	/**
		ime is for Input Method Editor workarounds
	*/
	fn ime(self) -> impl std::future::Future<Output = Result<crate::bind::types::BindRules, Self::DisplayBindError>> + Send;

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

mod session;

/**
	Bind the relevant display into sandbox

	This function must run after Portable's runtime dir is created

	Note that the defaults (if not specified) for sockets should be FALSE unless
	specified in configuration otherwise, as bind() will perform automatic enabling of
	native display protocols
*/
async fn bind(
	xdg:			std::sync::Arc<crate::xdg::XdgDirs>,
	logger:			crate::logger::LogSender,
	env:			crate::envs::holder::HoldChannel,

	portable_runtime:	crate::bind::subsystems::dirs::portable_runtime::PortableRuntime,
	app_id:			String,
	instance_id:		String,

	// socket enablement below
	mut x11:		bool,
	mut wayland:		bool,
) -> Result<BindRules, DisplayError> {

	/*
		Enable the native session type socket
	*/
	match session::detect().await {
		session::SessionType::Wayland	=> {
			wayland = true
		}
		session::SessionType::X11	=> {
			let _ = logger.send(
				crate::logger::LogMessage {
					level: crate::logger::LogLevel::Warn,
					message: format!("X11 is insecure!"),
				}
			).await;
			x11 = true
		}
		session::SessionType::Unknown	=> {
			let _ = logger.send(
				crate::logger::LogMessage {
					level: crate::logger::LogLevel::Warn,
					message: format!("Unknown session type!"),
				}
			).await;
		}
	};

	let mut spawn_collector = vec![];

	/*
		We use this variable to avoid applying Input Method workaround multiple times
	*/
	let mut ime_applied: bool = false;

	#[cfg(feature = "x11")]
	if x11 {
		let info = x11::X11 {
			logger:	logger.clone(),
			home:	home,
			env:	env.clone(),
		};

		if ! ime_applied {
			ime_applied = true;
			let info = info.clone();
			spawn_collector.push(
				tokio::spawn(async move {
					info
					.ime()
					.await
					.map_err(DisplayError::X11Error)
				})
			);
		}

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
		let info = wayland::Wayland {
			runtime_dir:		runtime_dir.as_path().to_path_buf(),
			env:			env,
			portable_runtime:	portable_runtime,
			logger:			logger,
			app_id:			app_id,
			instance_id:		instance_id,
		};

		if ! ime_applied {}

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
