/**
	Generate a nsswitch file

	userdb is stripped to drop the requirement of said Varlink service

	resolved is stripped to drop dependency on relevant socket
*/
async fn generate(
	portable_runtime:	std::path::PathBuf,
) -> Result<std::path::PathBuf, NsswitchError> {
	let content = {
		let mut file = String::new();

		file.push_str("passwd: files\n");
		file.push_str("group: files\n");
		file.push_str("shadow: files\n");
		file.push_str("gshadow: files\n");
		file.push_str("publickey: files\n");
		file.push_str("hosts: files myhostname dns\n");
		file.push_str("networks: files\n");
		file.push_str("protocols: files\n");
		file.push_str("services: files\n");
		file.push_str("ethers: files\n");
		file.push_str("rpc: files\n");
		file.push_str("netgroup: files\n");

		file
	};

	let nsswitch_path = {
		let mut path = portable_runtime;
		path.push("nsswitch");
		path
	};

	let mut file = tokio::fs::OpenOptions::new()
		.read(false)
		.write(true)
		.create_new(true)
		.mode(0o700)
		.open(&nsswitch_path)
		.await
		.map_err(NsswitchError::CreateError)
		?;

	use tokio::io::AsyncWriteExt;
	file.write(&content.into_bytes())
		.await
		.map_err(NsswitchError::IOError)
		?;

	Ok(nsswitch_path)
}

#[derive(thiserror::Error, Debug)]
pub enum NsswitchError {

	#[error("Could not create nsswitch: {0:#?}")]
	CreateError(std::io::Error),

	#[error("I/O error writing nsswitch: {0:#?}")]
	IOError(std::io::Error),
}
