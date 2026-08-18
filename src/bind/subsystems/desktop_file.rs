/**
	Install a desktop file if missing

	Utilises fixed metadata, and removes the file on shutdown.
*/
pub async fn install_desktop_file(
	stop:		std::sync::Arc<crate::stop::Stop>,
	logger:		crate::logger::LogSender,

	app_id:		String,

	data_home:	std::path::PathBuf,
	data_dirs:	Vec<std::path::PathBuf>,
) {
	match has_desktop_file(data_dirs, &app_id).await {
		Ok(true)	=> {}
		Ok(false)	=> {}
		Err(e)		=> {
			let _ = logger.send(
				crate::logger::LogMessage {
					level: crate::logger::LogLevel::Warn,
					message: format!("Could not detect .desktop file: {e:#?}"),
				},
			).await;
			return;
		}
	};

	let mut file_name = String::from(&app_id);
	file_name.push_str(".desktop");

	let mut dir_path = {
		let mut path = data_home;
		path.push("applications");

		path
	};


	let file_path = match std::fs::create_dir_all(&dir_path) {
		Ok(_)	=> {
			dir_path.push(file_name);
			dir_path
		}
		Err(e)	=> {
			let _ = logger.send(
				crate::logger::LogMessage {
					level: crate::logger::LogLevel::Warn,
					message: format!("Could not create applications directory: {e:#?}"),
				}
			).await;
			return;
		}
	};

	let file = tokio::fs::OpenOptions::new()
		.read(false)
		.write(true)
		.create_new(true)
		.mode(0o700)
		.open(&file_path)
		.await;

	let mut file = match file {
		Ok(v)	=> v,
		Err(e)	=> {
			let _ = logger.send(
				crate::logger::LogMessage {
					level: crate::logger::LogLevel::Warn,
					message: format!("Could not create .desktop file: {e:#?}"),
				}
			).await;
			return;
		}
	};

	{
		let token = stop.pre_cancel.clone();

		let res = stop.stop_funcs.send(
			crate::stop::StopMessage::Prepare {
				task:	tokio::spawn(
					async move {
						token.cancelled().await;

						tokio::fs::remove_file(file_path)
							.await
							.map_err(crate::stop::StopError::RemoveFsError)
					}
				),
			}
		);

		match res {
			Ok(_)	=> {}
			Err(e)	=> {
				let _ = logger.send(
					crate::logger::LogMessage {
						level: crate::logger::LogLevel::Warn,
						message: format!("Could not send to stop channel: {e:#?}"),
					}
				).await;
				return;
			}
		}
	};

	use tokio::io::AsyncWriteExt;

	match file.write(generate_file_content(&app_id).await.as_bytes()).await {
		Ok(_)	=> {}
		Err(e)	=> {
			let _ = logger.send(
				crate::logger::LogMessage {
					level: crate::logger::LogLevel::Warn,
					message: format!("Could not write .desktop file: {e:#?}"),
				}
			).await;
			return;
		}
	}
}

async fn generate_file_content(app_id: &str) -> String {
	let mut content = String::new();

	content.push_str("[Desktop Entry]");
	content.push_str("\n");

	content.push_str("Name=");
	content.push_str("Unknown app:");
	content.push_str(app_id);
	content.push_str("\n");

	content.push_str("Exec=");
	content.push_str("true");
	content.push_str("\n");

	content.push_str("Type=Application");
	content.push_str("\n");

	content.push_str("Icon=image-missing");
	content.push_str("\n");

	content.push_str("Comment=Application info missing");
	content.push_str("\n");

	content
}

async fn has_desktop_file(
	data_dirs:	Vec<std::path::PathBuf>,
	app_id:		&str,
) -> Result<bool, InstallDesktopFileError> {
	for path in data_dirs {
		let mut file_path = std::path::PathBuf::from(path);
		file_path.push("applications");
		let mut file_name = String::from(app_id);
		file_name.push_str(".desktop");
		file_path.push(file_name);

		if exists(file_path).await? {
			return Ok(true);
		}
	};
	Ok(false)
}

/**
	Whether the socket or file exists on filesystem
*/
pub async fn exists(path: std::path::PathBuf) -> Result<bool, InstallDesktopFileError> {
	tokio::task::spawn_blocking(|| {
		std::fs::exists(path).map_err(InstallDesktopFileError::IOError)
	}).await.map_err(InstallDesktopFileError::SpawnError)?
}

#[derive(thiserror::Error, Debug)]
pub enum InstallDesktopFileError {
	#[error("Could not determine if path exists")]
	IOError(std::io::Error),

	#[error("Could not determine if path exists: error spawning task: {0:#?}")]
	SpawnError(tokio::task::JoinError),

}
