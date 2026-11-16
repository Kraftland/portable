/**
	Generate a passwd and a group file

	shell is hard-coded to /usr/bin/bash, and it returns 2 bind rules back
*/
pub async fn generate(
	portable_runtime:	std::path::PathBuf,
	state_directory:	std::path::PathBuf,
) -> Result<crate::bind::types::BindRules, PasswdError> {
	let user = nix::unistd::User::from_uid(nix::unistd::Uid::current())
		.map_err(PasswdError::InfoError)
		?;

	let user = match user {
		Some(v)	=> {v}
		None	=> {
			return Err(PasswdError::EmptyInfo);
		}
	};

	//
	let overflow_user = match nix::unistd::User::from_uid(
			nix::unistd::Uid::from_raw(
				// https://docs.kernel.org/admin-guide/sysctl/kernel.html#id54
				u32_from_file(
					"/proc/sys/kernel/overflowuid"
				)
					.await
					?
			)
		)
		.map_err(PasswdError::InfoError)
		?
	{
		Some(v)	=> v,
		None	=> {
			return Err(PasswdError::OverflowEmptyUser);
		}
	};

	let overflow_group = match nix::unistd::Group::from_gid(
			nix::unistd::Gid::from_raw(
				// https://docs.kernel.org/admin-guide/sysctl/kernel.html#id54
				u32_from_file(
					"/proc/sys/kernel/overflowgid"
				)
					.await
					?
			)
		)
		.map_err(PasswdError::InfoError)
		?
	{
		Some(v)	=> v,
		None	=> {
			return Err(PasswdError::OverflowEmptyGroup);
		}
	};

	let group = {
		/*
			The /etc/group file is a text file that defines the groups on the
			system.  There is one entry per line, with the following format:
			group_name:password:GID:user_list


			Since we unshare user namespace, only nobody and current user is required.
		*/

		let mut group = String::new();

		let unistd_group = match nix::unistd::Group::from_gid(
				nix::unistd::Gid::current()
			)
				.map_err(PasswdError::InfoError)
				?
		{
			Some(v)	=> v,
			None	=> {
				return Err(PasswdError::EmptyInfo);
			}
		};

		group.push_str(
			format!(
				"{}:{}:{}:{}\n",
				unistd_group.name,
				"x",
				unistd_group.gid.as_raw(),
				user.name
			)
				.as_str()
		);

		group.push_str(
			format!(
				"{}:{}:{}:{}\n",
				overflow_group.name,
				"x",
				overflow_group.gid.as_raw(),
				overflow_user.name
			)
				.as_str()
		);

		group
	};

	let passwd = {
		let mut passwd = String::new();

		/*
			Effectively name:password:UID:GID:GECOS:directory:shell
		*/

		passwd.push_str(
			format!(
				"{}:{}:{}:{}:{}:{}:{}\n",
				user.name,
				"x",
				user.uid.as_raw(),
				user.gid.as_raw(),
				user.gecos.to_string_lossy(),
				state_directory.to_string_lossy(),
				user.shell.to_string_lossy(),
			)
			.as_str()
		);

		passwd.push_str(
			format!(
				"{}:{}:{}:{}:{}:{}:{}\n",
				overflow_user.name,
				"x",
				overflow_user.uid.as_raw(),
				overflow_user.gid.as_raw(),
				overflow_user.gecos.to_string_lossy(),
				overflow_user.dir.to_string_lossy(),
				overflow_user.shell.to_string_lossy(),
			)
			.as_str()
		);

		passwd
	};

	let passwd_path = {
		let mut path = portable_runtime.to_path_buf();
		path.push("passwd");
		path
	};

	let group_path = {
		let mut path = portable_runtime;
		path.push("group");
		path
	};

	{
		let mut file = tokio::fs::OpenOptions::new()
			.read(false)
			.write(true)
			.create_new(true)
			.mode(0o700)
			.open(&group_path)
			.await
			.map_err(PasswdError::CreateGroupError)
			?;

		use tokio::io::AsyncWriteExt;

		file.write(
			group.as_bytes()
		)
			.await
			.map_err(PasswdError::IOError)
			?;
	}

	{
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
	}

	Ok(
		vec![
			crate::bind::types::BindRule::Path {
				source:	passwd_path,
				dest:	"/etc/passwd".into(),
				class:	crate::bind::types::BindType::ReadOnly,
			},
			crate::bind::types::BindRule::Path {
				source:	group_path,
				dest:	"/etc/group".into(),
				class:	crate::bind::types::BindType::ReadOnly,
			}
		]
	)

}

/**
	Parse a 32-bit unsigned integer from a file.
*/
async fn u32_from_file(path: impl AsRef<std::path::Path>) -> Result<u32, PasswdError> {
	let mut file = tokio::fs::OpenOptions::new()
		.read(true)
		.write(false)
		.open(path)
		.await
		.map_err(PasswdError::IOParseI32)
		?;

	use tokio::io::AsyncReadExt;

	let mut buffer = String::new();

	file
		.read_to_string(&mut buffer)
		.await
		.map_err(PasswdError::IOParseI32)
		?;

	Ok(
		buffer.parse::<u32>()
			.map_err(PasswdError::IntegerOverflow)
			?
	)
}

#[derive(thiserror::Error, Debug)]
pub enum PasswdError {
	#[error("Integer overflow parsing String: {0}")]
	IntegerOverflow(std::num::ParseIntError),

	#[error("I/O error parsing integer from file: {0}")]
	IOParseI32(std::io::Error),

	#[error("Could not get user / group info: error calling getpwuid_r: {0}")]
	InfoError(nix::Error),

	#[error("Could not get user / group info: empty return")]
	EmptyInfo,

	#[error("Could not convert GECOS to String: {0}")]
	GecosStringError(std::ffi::c_str::IntoStringError),

	#[error("Could not create passwd: {0}")]
	CreatePasswdError(std::io::Error),

	#[error("Could not create group: {0}")]
	CreateGroupError(std::io::Error),

	#[error("I/O error writing passwd: {0}")]
	IOError(std::io::Error),

	#[error("Could not obtain info about the overflow user 65534: empty info")]
	OverflowEmptyUser,

	#[error("Could not obtain info about the overflow group 65534: empty info")]
	OverflowEmptyGroup,
}
