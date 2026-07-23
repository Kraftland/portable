use thiserror::Error;
use crate::config_definition::*;

#[derive(Debug, Error)]
pub enum ParseTomlConfigError {
	#[error("Could not open file: {0:#?}")]
	OpenConfigError(std::io::Error),

	#[error("Could not read file: {0:#?}")]
	ReadConfigError(std::io::Error),

	#[error("Could not deserialise file: {0:#?}")]
	DeserializeError(toml::de::Error),
}

pub async fn read_config(path: &std::path::Path) -> Result<Config, ParseTomlConfigError> {
	use tokio::io::AsyncReadExt;
	let mut file = tokio::fs::OpenOptions::new()
		.read(true)
		.write(false)
		.append(false)
		.truncate(false)
		.open(path)
		.await.map_err(ParseTomlConfigError::OpenConfigError)
		?;

	let mut content = String::new();
	file
		.read_to_string(&mut content)
		.await
		.map_err(ParseTomlConfigError::ReadConfigError)
		?;

	let config: Config = toml::from_str(&content)
		.map_err(ParseTomlConfigError::DeserializeError)
		?;
	Ok(config)
}
