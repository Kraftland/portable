#[derive(thiserror::Error, Debug)]
pub enum PulseError {
	#[error("Could not connect to PulseAudio socket: {0:#?}")]
	ConnectError(std::io::Error),

	#[error("Could not parse PulseAudio server address: {0:#?}")]
	PulseAddressError(String),

	#[error("Could not find the location for PulseAudio socket")]
	NoUseableSocketError(),

	#[error("Could not send environment variable: {0:#?}")]
	SendEnvError(tokio::sync::mpsc::error::SendError<crate::envs::holder::EnvMessage>),
}



/**
	This function is SLOW!
	Ideally it should run as early as possible.
*/

impl crate::bind::subsystems::GenerateBind for super::Audio {
	type BindError = PulseError;

	async fn bind(self) -> Result<crate::bind::types::BindRules, Self::BindError> {
		let path = get_server_address(&self.logger, self.runtime_dir).await;

		let path = match path {
			Some(v)	=> {
				v
			}
			None	=> {
				let _ = self.logger.send(
					crate::logger::LogMessage {
						level: crate::logger::LogLevel::Warn,
						message: format!("Could not find PulseAudio server"),
					},
				).await;
				return Ok(vec![]);
			}
		};
		generate_bind(self.env, path).await
	}
}

use crate::bind::types::BindRules;

async fn generate_bind(
	env:	crate::envs::holder::HoldChannel,
	path:	std::path::PathBuf,
) -> Result<BindRules, PulseError> {
	env.send(
		crate::envs::holder::EnvMessage::Add {
			key: "PULSE_SERVER".into(),
			value: "unix:/run/PulseAudio".into(),
		},
	).await
	.map_err(PulseError::SendEnvError)
	?;
	use crate::bind::types::BindRule;

	Ok(vec![
		BindRule::Path {
			source: path,
			dest: "/run/PulseAudio".into(),
			class: crate::bind::types::BindType::ReadOnly,
		}
	])
}

async fn get_server_address(
	logger:		&crate::logger::LogSender,
	runtime_dir:	std::path::PathBuf,
) -> Option<std::path::PathBuf> {
	let path = parse_address(
		std::env::var("PULSE_SERVER")
			.unwrap_or("".to_string()),
		runtime_dir,
	);
	let path = match path.await {
		Ok(v)	=> {v}
		Err(e)	=> {
			let _ = logger.send(
				crate::logger::LogMessage {
					level: crate::logger::LogLevel::Warn,
					message: format!("Could not find a valid PulseAudio address: {e:#?}"),
				},
			).await;
			return None;
		}
	};

	match activate_server(&path).await {
		Ok(_)	=> {
			Some(path)
		}
		Err(e)	=> {
			let _ = logger.send(
				crate::logger::LogMessage {
					level: crate::logger::LogLevel::Warn,
					message: format!("Could not activate PulseAudio server: {e:#?}"),
				},
			).await;
			None
		}
	}
}

async fn parse_address(
	path:		String,
	runtime_dir:	std::path::PathBuf,
) -> Result<std::path::PathBuf, PulseError> {
	for addr in path.split(" ") {
		let mut addr = addr;
		if addr.starts_with('{') {
			match addr.find('}') {
				Some(v)	=> {
					addr = &addr[v + 1..];
				}
				None	=> {
					return Err(
						PulseError::PulseAddressError(
							format!("unclosed {{"),
						),
					);
				}
			}
		};
		match addr.strip_prefix("unix:") {
			Some(v)	=> {
				return Ok(
					std::path::PathBuf::from(v)
				);
			}
			None	=> {}
		};
		match addr.starts_with("/") {
			true	=> {
				return Ok(
					std::path::PathBuf::from(addr)
				);
			}
			false	=> {}
		}
	};

	let mut default_path = runtime_dir;
	default_path.push("pulse");
	default_path.push("native");
	Ok(default_path)
}


async fn activate_server(path: &std::path::PathBuf) -> Result<(), PulseError> {
	std::os::unix::net::UnixStream::connect(path).map_err(PulseError::ConnectError)?;
	Ok(())
}
