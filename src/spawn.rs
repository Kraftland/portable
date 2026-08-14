/*
	Application spawning is designed to be init-independent here
	While we will only support systemd with feature systemd for now, it is possible to implement
	the "Start" trait and StartAppError error enum with other init systems
*/

#[cfg(feature = "systemd")]
pub mod systemd;
#[cfg(feature = "systemd")]
pub use systemd::start_transient::*;

pub trait Start {
	fn start(
		self,
		dbus_conn:	&zbus::Connection,
	// ) -> Result<(), crate::spawn::StartAppError>;
	) -> impl std::future::Future<Output = Result<(), crate::spawn::StartAppError>> + Send;
}

pub mod rng;
pub mod console;
pub mod instance_id;

pub struct Spawn {
	pub app_id:		String,

	/// Instance ID
	pub uid:		String,
	pub fs_rules:		crate::bind::types::BindRules,
	pub logger:		crate::logger::LogSender,
	pub stop:		tokio::sync::mpsc::Sender<crate::stop::StopLevel>,
	pub envs:		crate::envs::holder::HoldChannel,
	pub sandbox_home:	std::path::PathBuf,
}

