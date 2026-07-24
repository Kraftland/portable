use thiserror::Error;

#[derive(Error, Debug)]
pub enum ConfigError {
	#[error("Could not use TOML config path: invalid path: {0:#?}")]
	InvalidTomlPath(std::io::Error),

	#[error("Could not use TOML config: null or invalid environment variable: {0:#?}")]
	InvalidTomlVar(std::env::VarError),

	#[error("Could not use legacy Bash config path: invalid path: {0:#?}")]
	InvalidBashPath(std::io::Error),

	#[error("Could not use legacy Bash config: null or invalid environment variable: {0:#?}")]
	InvalidBashVar(std::env::VarError),

	#[error("Could not determine config type: spawn failed: {0:#?}")]
	SpawnError(tokio::task::JoinError),

	#[error("Could not find a useable configuration of TOML or legacy Bash: {0:?}")]
	NoAvailableConfig(
		Vec<String>,
	),

	#[error("Could not decode TOML configuration: {0:#?}")]
	InvalidTomlConfig(crate::config_toml::ParseTomlConfigError),

	#[error("Could not decode legacy Bash configuration: {0:#?}")]
	InvalidBashConfig(crate::config_legacy::LegacyConfigError),
}

enum ConfigType {
	TOML { path: std::path::PathBuf },
	LegacyBash { path: std::path::PathBuf },
}

impl crate::config_definition::Config {
	pub async fn get() -> Result<crate::config_definition::Config, ConfigError> {

		/*
			The trick here is that IntoIter implementation in std causes them to be
				placed in a top-to-down manner. Whose behaviour can be used to
				declear a priority between configuration types.

			The TOML configuration will always run first, allowing us to prioritise it
				over the legacy bash configuration.

			Ref: https://doc.rust-lang.org/std/vec/struct.IntoIter.html
		*/
		let config_spawns = vec![
			tokio::spawn(get_toml_path()),
			tokio::spawn(get_legacy_bash_path()),
		];

		let config_info;
		let mut config_errors = vec![];
		for spawn in config_spawns {
			match spawn.await.map_err(ConfigError::SpawnError)? {
				Ok(v)	=> {
					config_info = Some(v);
					break;
				}
				Err(e)	=> {
					config_errors.push(format!("{e:?}"));
				}
			}
		};

		let config_info = match config_info {
			Some(v)	=> {v}
			None	=> {
				return Err(ConfigError::NoAvailableConfig(config_errors));
			}
		};

		match config_info {
			ConfigType::TOML { path }	=> {
				crate::config_toml::read_config(&path)
					.await
					.map_err(ConfigError::InvalidTomlConfig)
			}
			ConfigType::LegacyBash { path }	=> {
				crate::config_legacy::get_legacy_conf(&path)
					.await
					.map_err(ConfigError::InvalidBashConfig)
			}
		}
	}
}

async fn get_toml_path() -> Result<ConfigType, ConfigError> {
	match std::env::var("PORTABLE_CONF") {
		Ok(v)	=> {
			let path = std::path::PathBuf::from(v);
			let path = std::path::absolute(path)
				.map_err(ConfigError::InvalidTomlPath)?;
			Ok(
				ConfigType::TOML { path: path }
			)
		}
		Err(e)	=> {
			Err(ConfigError::InvalidTomlVar(e))
		}
	}
}

async fn get_legacy_bash_path() -> Result<ConfigType, ConfigError> {
	match std::env::var("_portableConfig") {
		Ok(v)	=> {
			let path = std::path::PathBuf::from(v);
			let path = std::path::absolute(path)
				.map_err(ConfigError::InvalidBashPath)?;
			Ok(
				ConfigType::TOML { path: path }
			)
		}
		Err(e)	=> {
			Err(ConfigError::InvalidBashVar(e))
		}
	}
}
