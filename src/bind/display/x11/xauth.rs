#[derive(thiserror::Error, Debug)]
pub enum XAuthorityError {
	#[error("Invalid XAUTHORITY environment variable: {0:#?}")]
	InvalidEnv(std::env::VarError),

	#[error("I/O error: {0:#?}")]
	IOError(crate::bind::display::ExistError),

	#[error("Could not find useable XAUTHORITY file")]
	NotExistError,
}

pub async fn get_authority_path(home: &std::path::PathBuf) -> Result<std::path::PathBuf, XAuthorityError> {
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

	let exist = crate::bind::display::exists(path.clone())
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
