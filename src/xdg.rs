use thiserror::Error;

#[derive(Debug, Error)]
pub enum XdgError {
	#[error("Could not determine XDG dirs: invalid variable: {0:#?}")]
	InvalidVar(std::env::VarError),

	#[error("Could not find $HOME")]
	HomeNotFound,
}

pub struct XDG_DIRS {
	pub runtime:		std::path::PathBuf,
	pub home:		std::path::PathBuf,
	pub config_home:	std::path::PathBuf,
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

	async fn home() -> Option<std::path::PathBuf> {
		std::env::home_dir()
	}

	async fn config_home (home: &std::path::PathBuf) -> Result<std::path::PathBuf, XdgError> {
		match std::env::var("XDG_CONFIG_HOME") {
			Ok(v)	=> {
				Ok(std::path::PathBuf::from(v))
			}
			Err(e)	=> {
				match e {
					std::env::VarError::NotPresent	=> {
						let mut path: std::path::PathBuf =
							[home].iter().collect();
						path.push(".config");
						Ok(path)
					}
					_				=> {
						return Err(XdgError::InvalidVar(e))
					}
				}
			}
		}
	}
}
