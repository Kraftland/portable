#[derive(Debug, thiserror::Error)]
pub enum DisplayBindError {
	#[error("I/O error: {0:#?}")]
	IOError(crate::bind::display::ExistError),

	#[error("Could not use Wayland socket: path does not exist")]
	NonExistentError,
}

pub mod find_socket;

pub struct Wayland;

impl super::BindDisplay for Wayland {
	async fn bind(self) -> Result<crate::bind::types::BindRules, Self::DisplayBindError> {
		unimplemented!()
	}

	async fn ime(self) -> Result<crate::bind::types::BindRules, Self::DisplayBindError> {
		unimplemented!()
	}

	type DisplayBindError = DisplayBindError;
}
