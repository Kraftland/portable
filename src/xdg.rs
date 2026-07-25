use thiserror::Error;

#[derive(Debug, Error)]
pub enum XdgError {
	#[error("Could not determine XDG dirs: invalid variable: {0:#?}")]
	InvalidVar(std::env::VarError),
}

pub struct XDG_DIRS {
	pub runtime:	std::path::PathBuf,
}

impl XDG_DIRS {

	async fn runtime (
	) -> Result<std::path::PathBuf, XdgError> {
		match std::env::var("XDG_RUNTIME_DIR") {
			Ok(v)	=> {Ok(std::path::PathBuf::from(v))}
			Err(e)	=> {
				return Err(XdgError::InvalidVar(e))
			}
		}
	}
}
