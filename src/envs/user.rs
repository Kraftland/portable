#[derive(Debug, thiserror::Error)]
pub enum LoadError {
	#[error("I/O error opening file: {0:#?}")]
	OpenError(std::io::Error),

	#[error("I/O error reading file: {0:#?}")]
	ReadError(std::io::Error),

	#[error("Is a symlink")]
	IsSymlink,
}

pub async fn load_user_envs(
	map:	&mut std::collections::HashMap<String, String>,
	xdg:	std::sync::Arc<crate::xdg::XdgDirs>,
	config:	std::sync::Arc<crate::config::Config>,
	log:	&crate::logger::LogSender,
) -> Result<(), LoadError> {
	let env_path = {
		let mut sandbox_home = {
			let mut path = xdg.data_home.to_path_buf();
			path.push(&config.metadata.state_directory);
			path
		};
		sandbox_home.push("portable.env");
		sandbox_home
	};

	match tokio::fs::try_exists(&env_path).await.map_err(LoadError::OpenError)? {
		true	=> {}
		false	=> {
			return Ok(());
		}
	};

	{
		let metadata = tokio::fs::symlink_metadata(&env_path)
			.await
			.map_err(LoadError::OpenError)
			?;

		if metadata.is_symlink() {
			return Err(LoadError::IsSymlink);
		};
	};

	let mut file = tokio::fs::OpenOptions::new()
		.read(true)
		.write(false)
		.create(false)
		.mode(0o700)
		.open(env_path)
		.await
		.map_err(LoadError::ReadError)
		?;

	let content = {
		let mut content = String::new();

		use tokio::io::AsyncReadExt;

		file
			.read_to_string(&mut content)
			.await
			.map_err(LoadError::ReadError)
			?;

		content
	};

	for line in content.split("\n") {
		match line.split_once("=") {
			Some((k, v))	=> {
				map.insert(k.to_string(), v.to_string());
			}
			None		=> {
				let _ = log.send(
					crate::logger::LogMessage {
						level:	crate::logger::LogLevel::Warn,
						message: format!("Malformed environment: {line}"),
					}
				).await;
				continue;
			}
		}
	};

	Ok(())
}
