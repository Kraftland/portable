/**
	Ask the user for consent of exposing files. Currently uses Zenity but we want to natively
		do that in the future.
*/
async fn question(
	rw:		&Vec<super::MountPathPreference>,
	ro:		&Vec<super::MountPathPreference>,
	device:		&Vec<super::MountPathPreference>,
)
-> Result<bool, QuestionError> {
	let title = gettextrs::gettext("Designated the following list of path to mount");

	let mut cmd_args = vec![
		"--title",
		title.as_str(),
		"--icon=folder-open-symbolic",
		"--question",
		"--default-cancel",
	];

	let text = {
		let mut string = String::new();
		string.push_str("--text=Exposing the following path: \n");
		for path in rw {
			string.push('\n');
			string.push_str(&path.0.to_string_lossy());
		};
		for path in ro {
			string.push('\n');
			string.push_str(&path.0.to_string_lossy());
		};
		for path in device {
			string.push('\n');
			string.push_str(&path.0.to_string_lossy());
		};
		string
	};

	cmd_args.push(&text);

	let mut command = tokio::process::Command::new("zenity");

	let mut command = command.kill_on_drop(true);
	command = command.args(cmd_args);

	let mut child = command
		.spawn()
		.map_err(QuestionError::SpawnError)
		?;

	if
		child
			.wait()
			.await
			.map_err(QuestionError::ExitError)
			?
			.success()
	{
		Ok(true)
	} else {
		Ok(false)
	}
}

#[derive(Debug, thiserror::Error)]
pub enum QuestionError {
	#[error("Could not spawn zenity: {0}")]
	SpawnError(std::io::Error),

	#[error("Could not wait for zenity: {0}")]
	ExitError(std::io::Error),
}
