pub mod portable_runtime;

pub mod flatpak;

pub mod documents;

pub trait RuntimePathsTrait: Sized {
	/**
		Create a new runtime path for type
	*/
	fn new(
		config:		&crate::config::Config,
		xdg:		&crate::xdg::XdgDirs,
		instance_id:	&str,
	)	->
		Self;

	/**
		Create the inner path
	*/
	fn create_path(&self, stop_func: tokio::sync::mpsc::Sender<crate::stop::StopFunc>) ->
		impl std::future::Future<Output = Result<(), Self::RuntimePathError>> + Send;

	fn path(&self) -> std::path::PathBuf;

	type RuntimePathError;
}
