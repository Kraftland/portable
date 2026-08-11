/**
	Generate a passwd file

	shell is hard-coded to /usr/bin/bash, and it returns the passwd file path back
*/
pub async fn generate(
	portable_runtime:	std::path::PathBuf,
	state_directory:	std::path::PathBuf,
) -> Result<std::path::PathBuf, PasswdError> {


	let mut passwd = String::new();

	{
		/*
			Effectively user:x:1011:1011:User name:/home/user:/bin/bash
		*/
		let user = nix::unistd::User::from_uid(nix::unistd::Uid::current())
			.map_err(PasswdError::InfoError)
			?;

		let user = match user {
			Some(v)	=> {v}
			None	=> {
				return Err(PasswdError::EmptyInfo);
			}
		};

		passwd.push_str(&user.name);
		passwd.push_str(":x:");
		passwd.push_str(&user.uid.as_raw().to_string());
		passwd.push_str(":");
		passwd.push_str(&user.gid.as_raw().to_string());
		passwd.push_str(":");
		// We don't have Name here, using GECOS instead
		let gecos = user
			.gecos
			.into_string()
			.map_err(PasswdError::GecosStringError)
			?;
		passwd.push_str(&gecos);
		passwd.push_str(":");

		passwd.push_str(&state_directory.to_string_lossy());
		passwd.push_str(":");

		passwd.push_str("/usr/bin/bash");
		passwd.push_str("\n");
	};

	// Overflow user
	passwd.push_str("nobody:x:65534:65534:Kernel Overflow User:/:/usr/bin/nologin");
	passwd.push_str("\n");

	let passwd_path = {
		let mut path = portable_runtime;
		path.push("passwd");
		path
	};

	let mut file = tokio::fs::OpenOptions::new()
		.read(false)
		.write(true)
		.create_new(true)
		.mode(0o700)
		.open(&passwd_path)
		.await
		.map_err(PasswdError::CreatePasswdError)
		?;

	use tokio::io::AsyncWriteExt;

	file.write(
		passwd.as_bytes()
	)
		.await
		.map_err(PasswdError::IOError)
		?;

	Ok(passwd_path)

}

#[derive(thiserror::Error, Debug)]
pub enum PasswdError {
	#[error("Could not get user info: error calling getpwuid_r: {0:#?}")]
	InfoError(nix::Error),

	#[error("Could not get user info: empty return from getpwuid_r")]
	EmptyInfo,

	#[error("Could not convert GECOS to String: {0:#?}")]
	GecosStringError(std::ffi::c_str::IntoStringError),

	#[error("Could not create passwd: {0:#?}")]
	CreatePasswdError(std::io::Error),

	#[error("I/O error writing passwd: {0:#?}")]
	IOError(std::io::Error),
}
