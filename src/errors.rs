use thiserror::Error;
#[derive(Debug, Error)]
pub enum StartError {
	#[error("Could not contact logging thread: {0:#?}")]
	LogError(tokio::sync::mpsc::error::SendError<crate::logger::messages::LogMessage>),

	#[error("Could not read config: {0:#?}")]
	ConfigError(crate::config::ConfigError),

	#[error("Could not populate XDG Base Directories: {0:#?}")]
	XdgError(crate::xdg::XdgError),

	#[error("Could not spawn thread: {0:#?}")]
	SpawnError(tokio::task::JoinError),

	#[error("Could not register D-Bus service: {0:#?}")]
	BusError(crate::ipc::register::RegisterError),

	#[error("Could not parse runtime options: {0:#?}")]
	RuntimeOptError(crate::pref::runtime::cmdline::RuntimeOptsError),

	#[error("Could not share files or directories: {0:#?}")]
	ShareError(crate::pref::runtime::cmdline::share_file::ShareError),

	#[error("Could not stop remote sandbox: {0:#?}")]
	StopControllerError(crate::ipc::controller::quit::StopError),

	#[error("Could not generate a suitable instance ID: {0:#?}")]
	InstanceIDError(crate::spawn::instance_id::InstanceIDError),

	#[cfg(feature = "flatpak")]
	#[error("Could not create Flatpak runtime path: {0:#?}")]
	FlatpakRuntimePathError(crate::bind::subsystems::dirs::flatpak::Error),

	#[error("Could not create Flatpak Info: {0:#?}")]
	FlatpakInfoError(crate::bind::flatpak_info::FlatpakInfoError),

	#[error("Could not create Portable runtime path: {0:#?}")]
	PortableRuntimePathError(crate::bind::subsystems::dirs::portable_runtime::Error),

	#[error("Could not create Documents Portal path: {0:#?}")]
	DocumentRuntimePathError(crate::bind::subsystems::dirs::documents::DocumentError),

	#[error("Could not start D-Bus proxy: {0:#?}")]
	ProxyError(crate::bind::bus::proxies::StartProxyError),

	#[error("Could not generate bind rules: {0:#?}")]
	BindError(crate::bind::subsystems::BindError),

	#[error("Could not publish info for Init: {0:#?}")]
	InitPublishError(zbus::Error),

	#[error("Could not spawn sandbox: {0:#?}")]
	SpawnSandboxError(crate::spawn::StartAppError),

	#[error("Could not start auxiliary instance: {0:#?}")]
	AuxStartError(crate::ipc::init::aux_start::AuxStartError),

	#[error("Could not wake up application via tray: {0:#?}")]
	TrayError(crate::ipc::init::tray::WakeError),

	#[error("Could not reset app permission: {0:#?}")]
	ResetError(crate::pref::runtime::cmdline::ResetError),

	#[error("Could not open sandbox home: {0:#?}")]
	OpenHomeError(zbus::Error),

	#[cfg(feature = "desktop-file")]
	#[error("Could not parse desktop file: {0}")]
	DesktopFileError(crate::pref::runtime::desktop_file::InstallDesktopFileError),

	#[error("Could not start the sandbox: a fatal error has occured")]
	FatalShutdown,
}
