#[derive(thiserror::Error, Debug)]
pub enum XAuthorityError {
	#[error("Invalid XAUTHORITY environment variable: {0:#?}")]
	InvalidEnv(std::env::VarError),

	#[error("I/O error: {0:#?}")]
	IOError(crate::bind::subsystems::display::ExistError),

	#[error("Could not find useable XAUTHORITY file")]
	NotExistError,

	#[error("Could not send XAUTHORITY environment variable: {0:#?}")]
	SendEnvError(tokio::sync::mpsc::error::SendError<crate::envs::holder::EnvMessage>),
}

pub async fn bind(
	home:	std::path::PathBuf,
	env:	crate::envs::holder::HoldChannel,
) -> Result<crate::bind::types::BindRules, XAuthorityError> {
	let path = get_authority_path(&home).await?;
	use crate::bind::types::BindRule;
	let ret = vec![
		BindRule::Path {
			source: path.clone(),
			dest: std::path::PathBuf::from("/run/XAuthority"),
			class: crate::bind::types::BindType::ReadOnly,
		},
	];

	env.send(
		crate::envs::holder::EnvMessage::Add {
			key: "XAUTHORITY".into(),
			value: "/run/XAuthority".into(),
		},
	)
		.await
		.map_err(XAuthorityError::SendEnvError)
		?;

	Ok(ret)
}

async fn get_authority_path(home: &std::path::PathBuf) -> Result<std::path::PathBuf, XAuthorityError> {
	let path = match std::env::var("XAUTHORITY") {
		Ok(v)	=> {std::path::PathBuf::from(v)}
		Err(e)	=> {
			match e {
				std::env::VarError::NotPresent	=> {}
				_				=> {
					return Err(XAuthorityError::InvalidEnv(e));
				}
			}

			let mut path = std::path::PathBuf::from(home);
			path.push(".Xauthority");
			path
		}
	};

	let exist = crate::bind::subsystems::display::exists(path.clone())
		.await
		.map_err(XAuthorityError::IOError)
		?;
	match exist {
		true	=> {
			Ok(path)
		}
		false	=> {
			Err(XAuthorityError::NotExistError)
		}
	}
}
