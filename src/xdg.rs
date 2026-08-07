use thiserror::Error;

#[derive(Debug, Error)]
pub enum XdgError {
	#[error("Could not determine XDG dirs: invalid variable: {0:#?}")]
	InvalidVar(std::env::VarError),

	#[error("Could not find $HOME")]
	HomeNotFound,
}

#[derive(Debug)]
pub struct XdgDirs {
	pub runtime:		std::path::PathBuf,
	pub home:		std::path::PathBuf,
	pub config_home:	std::path::PathBuf,
	pub data_home:		std::path::PathBuf,

	/**
		$XDG_DATA_DIRS defines the preference-ordered set of base directories
		to search for data files in addition to the $XDG_DATA_HOME base directory.

		Portable's data_dirs definition includes DATA_HOME
	*/
	pub data_dirs:		Vec<std::path::PathBuf>
}

impl XdgDirs {
	pub async fn get () -> Result<Self, XdgError> {
		let home = match Self::home().await {
			Some(v)	=> {v}
			None	=> {
				return Err(XdgError::HomeNotFound);
			}
		};

		let runtime_dir = Self::runtime().await?;
		let data_home = Self::data_home(&home).await?;

		let data_dirs = {
			let mut ret = vec![];
			ret.push(data_home.clone());

			let data_env = match std::env::var("XDG_DATA_DIRS") {
				Ok(v)	=> {
					let iter = v.split(":");
					let mut env = vec![];
					for pth in iter {
						env.push(std::path::PathBuf::from(pth));
					};
					env
				}
				Err(_)	=> {
					vec![
						std::path::PathBuf::from("/usr/local/share"),
						std::path::PathBuf::from("/usr/share"),
					]
				}
			};

			ret.extend(data_env);
			ret
		};

		Ok(Self {
			runtime:		runtime_dir.clone(),
			config_home:		Self::config_home(&home).await?,
			data_home:		data_home,
			home:			home,
			data_dirs:		data_dirs,
		})
	}

	async fn runtime (
	) -> Result<std::path::PathBuf, XdgError> {
		match std::env::var("XDG_RUNTIME_DIR") {
			Ok(v)	=> {Ok(std::path::PathBuf::from(v))}
			Err(e)	=> {
				return Err(XdgError::InvalidVar(e))
			}
		}
	}

	async fn data_home(home: &std::path::PathBuf) -> Result<std::path::PathBuf, XdgError> {
		match std::env::var("XDG_DATA_HOME") {
			Ok(v)	=> {
				Ok(
					std::path::PathBuf::from(v)
				)
			}
			Err(e)	=> {
				match e {
					std::env::VarError::NotPresent	=> {
						let mut path = home.clone();
						path.push(".local");
						path.push("share");
						Ok(path)
					}
					_				=> {
						Err(
							XdgError::InvalidVar(e)
						)
					}
				}
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
