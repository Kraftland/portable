pub mod portable_runtime;

pub mod flatpak;

pub mod documents;

pub trait RuntimePathsTrait: Sized {
	/**
		Create a new runtime path for type
	*/
	fn new(
		config:		std::sync::Arc<crate::config::config_definition::Config>,
		xdg:		std::sync::Arc<crate::xdg::XdgDirs>,
		instance_id:	std::sync::Arc<String>,
	)	->
		Self;

	/**
		Create the inner path
	*/
	fn create_path(
		&self,
		stop:	std::sync::Arc<crate::stop::Stop>
	) ->
		impl std::future::Future<Output = Result<(), Self::RuntimePathError>> + Send;

	fn path(&self) -> std::path::PathBuf;

	type RuntimePathError;
}
