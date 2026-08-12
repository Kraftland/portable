mod paths;
mod theming;
mod create_state_dir;

/**
	The user bind subsystem exposes several paths under the user home. As opposed to system
	subsystem managing system paths.

	It is designed to provide theming consistency, mount the sandbox home, and expose
	user selected paths using bind-mount or via Portals.

	As such, the second argument is a path map for later use in the D-Bus subsystem.
*/
pub struct UserBind {
	pub translator:	crate::bind::translate::Delta,
	pub xdg:	std::sync::Arc<crate::xdg::XdgDirs>,
	pub config:	std::sync::Arc<crate::config::config_definition::Config>,
}

impl super::GenerateBind for UserBind {
	async fn bind(self) -> Result<crate::bind::types::BindRules, Self::BindError> {
		let mut binds = paths::bind(
			self.xdg.data_home.to_path_buf(),
			self.xdg.home.clone(),
			self.config.metadata.state_directory,
		).await?;
	}
}

#[derive(thiserror::Error, Debug)]
pub enum UserBindError {
	#[error("I/O error creating sandbox home: {0:#?}")]
	CreateHomeError(std::io::Error),

	#[error("Error translating user path: {0:#?}")]
	TranslatePathError(crate::bind::translate::TranslatePathError),
}
