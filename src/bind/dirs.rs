pub mod portable_runtime;

pub trait RuntimePaths: Sized {
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
	async fn create_path(&self)		->
		Result<(), Self::RuntimePathError>;

	type RuntimePathError;
}
