/**
	Checks if a .desktop file is installed, if not then generate a stub.

	Returns a `freedesktop_desktop_entry::DesktopEntry` for extracting display names etc.

	Caller should not treat errors as non-fatal, as XDP and various other IPC services expect
	a valid desktop file.
*/
pub async fn get(
	stop:		std::sync::Arc<crate::stop::Stop>,

	config:		std::sync::Arc<crate::config::Config>,

	xdg:		std::sync::Arc<crate::xdg::XdgDirs>,
) -> Result<freedesktop_desktop_entry::DesktopEntry, InstallDesktopFileError> {
	/*
		Use desktop file if already available
	*/

	match search_desktop_file(&xdg.data_dirs, config.metadata.sandbox_id.as_str()).await {
		Ok(Some(v))	=> {
			let content = {
				use tokio::io::AsyncReadExt;

				let mut file = tokio::fs::OpenOptions::new()
					.read(true)
					.write(false)
					.create(false)
					.open(&v)
					.await
					.map_err(InstallDesktopFileError::OpenIOError)
					?;

				let mut buffer = String::new();

				file
					.read_to_string(&mut buffer)
					.await
					.map_err(InstallDesktopFileError::ReadIOError)
					?;

				buffer
			};

			let desktop_entry = freedesktop_desktop_entry::DesktopEntry::from_str(
				v,
				content.as_str(),
				None::<&[&str]>,
			)
				.map_err(InstallDesktopFileError::ParseError)
				?;

			return Ok(desktop_entry);
		}
		Ok(None)	=> {}
		Err(e)		=> {
			return Err(e);
		}
	};

	let stub_desktop_file_path = {
		let mut file_name = String::from(&config.metadata.sandbox_id);
		file_name.push_str(".desktop");

		let mut path = xdg.data_home.to_path_buf();
		path.push("applications");

		match tokio::fs::create_dir_all(&path).await {
			Ok(_)	=> {}
			Err(e)	=> {
				return Err(InstallDesktopFileError::InstallIOError(e));
			}
		}

		path.push(&file_name);

		path
	};

	let mut stub_file = tokio::fs::OpenOptions::new()
		.read(false)
		.write(true)
		.create_new(true)
		.mode(0o700)
		.open(&stub_desktop_file_path)
		.await
		.map_err(InstallDesktopFileError::InstallIOError)
		?;

	// Prepare stop cleaner
	{
		let child_token = stop.pre_parent.child_token();

		let path = stub_desktop_file_path.to_path_buf();

		stop.stop_funcs.send(
			crate::stop::StopMessage::Prepare {
				task:	tokio::spawn(
					async move {
						child_token.cancelled().await;

						tokio::fs::remove_file(path)
							.await
							.map_err(crate::stop::StopError::RemoveFsError)
					}
				),
			}
		)
			.map_err(InstallDesktopFileError::StopError)
			?;
	};

	let stub_file_content = generate_file_content(&config.metadata.sandbox_id)
				.await;

	{
		use tokio::io::AsyncWriteExt;

		stub_file.write(
			stub_file_content
				.as_bytes()
		)
			.await
			.map_err(InstallDesktopFileError::InstallIOError)
			?
	};

	Ok(
		freedesktop_desktop_entry::DesktopEntry::from_str(
			stub_desktop_file_path,
			stub_file_content.as_str(),
			None::<&[&str]>,
		)
			.map_err(InstallDesktopFileError::ParseError)
			?
	)
}

async fn generate_file_content(app_id: &str) -> String {
	let mut content = String::new();

	content.push_str("[Desktop Entry]");
	content.push_str("\n");

	content.push_str("Name=");
	content.push_str("Unknown:");
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

async fn search_desktop_file(
	data_dirs:	&Vec<std::path::PathBuf>,
	app_id:		&str,
) -> Result<Option<std::path::PathBuf>, InstallDesktopFileError> {
	for path in data_dirs {
		let mut file_path = std::path::PathBuf::from(path);
		file_path.push("applications");
		let mut file_name = String::from(app_id);
		file_name.push_str(".desktop");
		file_path.push(file_name);

		if tokio::fs::try_exists(&file_path).await.map_err(InstallDesktopFileError::ExistIOError)? {
			return Ok(Some(file_path));
		}
	};
	Ok(None)
}

#[derive(thiserror::Error, Debug)]
pub enum InstallDesktopFileError {
	#[error("Could not determine if path exists: {0}")]
	ExistIOError(std::io::Error),

	#[error("Could not open desktop file: {0}")]
	OpenIOError(std::io::Error),

	#[error("Could not read desktop file: {0}")]
	ReadIOError(std::io::Error),

	#[error("Could not parse desktop file: {0}")]
	ParseError(freedesktop_desktop_entry::DecodeError),

	#[error("Could not install desktop file: {0}")]
	InstallIOError(std::io::Error),

	#[error("Could not initiate stop task: {0}")]
	StopError(tokio::sync::mpsc::error::SendError<crate::stop::StopMessage>),

}
