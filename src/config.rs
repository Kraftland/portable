use thiserror::Error;

pub mod config_toml;
pub mod config_legacy;
pub mod config_definition;

pub use config_definition::Config;

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
	InvalidTomlConfig(config_toml::ParseTomlConfigError),

	#[error("Could not decode legacy Bash configuration: {0:#?}")]
	InvalidBashConfig(config_legacy::LegacyConfigError),
}

#[derive(Debug)]
enum ConfigType {
	TOML { path: std::path::PathBuf },
	LegacyBash { path: std::path::PathBuf },
}

impl config_definition::Config {
	pub async fn get(
		logger:		crate::logger::LogSender,
		config_home:	std::path::PathBuf,
	) -> Result<config_definition::Config, ConfigError> {
	let config_clone = config_home.clone();
		/*
			The trick here is that IntoIter implementation in std causes them to be
				placed in a top-to-down manner. Whose behaviour can be used to
				declare a priority between configuration types.

			The TOML configuration will always run first, allowing us to prioritise it
				over the legacy bash configuration.

			Ref: https://doc.rust-lang.org/std/vec/struct.IntoIter.html
		*/
		let config_spawns = vec![
			tokio::spawn(get_toml_path(config_clone)),
			tokio::spawn(get_legacy_bash_path()),
		];

		let mut config_info = None;
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

		let _ = logger.send(
			crate::logger::LogMessage {
				level: crate::logger::LogLevel::Debug,
				message: format!("Picked configuration: {config_info:?}"),
			},
		).await;

		match config_info {
			ConfigType::TOML { path }	=> {
				config_toml::read_config(&path)
					.await
					.map_err(ConfigError::InvalidTomlConfig)
			}
			ConfigType::LegacyBash { path }	=> {
				config_legacy::get_legacy_conf(&path)
					.await
					.map_err(ConfigError::InvalidBashConfig)
			}
		}
	}
}

async fn get_toml_path(config_home: std::path::PathBuf) -> Result<ConfigType, ConfigError> {
	use std::path::PathBuf;
	match std::env::var("PORTABLE_CONF") {
		Ok(v)	=> {
			/*
				This dictates what preference we prefer among configurations
				The last means least preferred, while the first means most preferred

				It holds a tuple: PathBuf to potentially useable path,
				and JoinHandle to confirm whether it's available.
			*/
			let try_config_path = vec![
				{
					/*
						Raw path configuration
					*/
					let base = PathBuf::from(&v);
					(base.clone(), tokio::spawn(path_exist(base)))
				},
				{
					/*
						User level configuration
						$XDG_CONFIG_HOME/portable/info/appID/config.toml
					*/
					let mut base = PathBuf::from(&config_home);
					base.push("portable");
					base.push("info");
					base.push(&v);
					base.push("config.toml");
					(base.clone(), tokio::spawn(path_exist(base)))
				},

				{
					/*
						System level configuration
						/usr/lib/portable/info/appID/config.toml
					*/
					let mut base = PathBuf::from("/usr/lib/portable/info");
					base.push(&v);
					base.push("config.toml");
					(base.clone(), tokio::spawn(path_exist(base)))
				},
			];

			for (path, result) in try_config_path {
				match result.await.map_err(ConfigError::SpawnError)? {
					true	=> {
						return	Ok(
							ConfigType::TOML { path }
						);
					}
					false	=> {}
				}
			};


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
				ConfigType::LegacyBash { path: path }
			)
		}
		Err(e)	=> {
			Err(ConfigError::InvalidBashVar(e))
		}
	}
}

async fn path_exist(path: std::path::PathBuf) -> bool {
	let path = path.clone();
	tokio::task::spawn_blocking(
		move || {
			std::fs::exists(path)
		},
	)
		.await
		.unwrap_or(Ok(false))
		.unwrap_or(false)
}
