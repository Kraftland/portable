mod paths;
mod create_state_dir;

/**
	The user bind subsystem exposes several paths under the user home. As opposed to system
	subsystem managing system paths.

	It is designed to provide theming consistency, mount the sandbox home, and expose
	user selected paths using bind-mount or via Portals.

	As such, the second argument is a path map for later use in the D-Bus subsystem.
*/
pub async fn bind()
-> Result<(crate::bind::types::BindRules, std::collections::HashMap<String, String>), UserBindError>
{
	unimplemented!()
}

#[derive(thiserror::Error, Debug)]
pub enum UserBindError {
	#[error("I/O error creating sandbox home: {0:#?}")]
	CreateHomeError(std::io::Error),
}
