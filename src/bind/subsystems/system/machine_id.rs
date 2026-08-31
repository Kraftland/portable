/**
	Generates or retrieves the machine-id from a file

	The default location to store which file is at XDG_CONFIG_HOME/portable/sandbox_id/machine-id
*/
pub async fn bind(
	config:	std::sync::Arc<crate::config::Config>,
	xdg:	std::sync::Arc<crate::xdg::XdgDirs>,
)
-> Result<crate::bind::types::BindRules, super::SystemBindError> {
	let path = {
		let mut path = xdg.config_home.to_path_buf();
		path.push("portable");
		path.push(&config.metadata.sandbox_id);
		path.push("machine-id");
		path
	};

	exist_or_generate_id(&path)
		.await
		?;

	use crate::bind::types::BindRule;

	Ok(
		vec![
			BindRule::Path {
				source:	path,
				dest:	"/etc/machine-id".into(),
				class:	crate::bind::types::BindType::ReadOnly,
			}
		]
	)

}

/**
	This function accepts a give file path, and verifies that the file actually is present.

	If it does not exist,
		it is expected to generate machine-id then write to that specific file for future use.
*/
async fn exist_or_generate_id(path: &std::path::PathBuf) -> Result<(), super::SystemBindError> {
	if tokio::fs::try_exists(&path).await.map_err(super::SystemBindError::IOError)? {
		Ok(())
	} else {
		let uuid = generate_id();

		let mut file = {
			let parent = match path.parent() {
				Some(v)	=> v,
				None	=> {
					return Err(
						super::SystemBindError::NoneParentForMachineID,
					);
				}
			};

			tokio::fs::create_dir_all(&parent)
				.await
				.map_err(super::SystemBindError::IOError)
				?;

			tokio::fs::OpenOptions::new()
				.read(false)
				.write(true)
				.create_new(true)
				.mode(0o700)
				.open(&path)
				.await
				.map_err(super::SystemBindError::IOError)
				?
		};

		use tokio::io::AsyncWriteExt;

		file
			.write(
				uuid
					.as_bytes()
				)
			.await
			.map_err(super::SystemBindError::IOError)
			?;
		Ok(())
	}
}

/**
	Generates a version 4 UUID to be used as machine-id
*/
fn generate_id() -> String {
	let uuid = uuid::Uuid::new_v4();

	uuid.
		simple()
		.encode_lower(&mut uuid::Uuid::encode_buffer())
		.to_string()
}
