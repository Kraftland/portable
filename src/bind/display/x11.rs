#[derive(Debug, thiserror::Error)]
pub enum DisplayBindError {
	#[error("Could not bind X11 display: {0:#?}")]
	SpawnError(tokio::task::JoinError),
}

#[derive(Clone)]
pub struct X11 {
	pub logger:		crate::logger::LogSender,
	pub home:		std::path::PathBuf,
	pub env:		crate::envs::holder::HoldChannel,
}

pub mod xauth;

impl super::BindDisplay for X11 {
	async fn bind(self) -> Result<crate::bind::types::BindRules, Self::DisplayBindError> {
		let mut ret = vec![];

		let xauth_spawn = tokio::spawn(xauth::bind(self.home, self.env));



		match xauth_spawn.await.map_err(DisplayBindError::SpawnError)? {
			Ok(v)	=> {
				ret.extend(v);
			}
			Err(e)	=> {
				let _ = self.logger.send(
					crate::logger::LogMessage {
						level: crate::logger::LogLevel::Warn,
						message: format!("{e:#?}"),
					},
				).await;
			}
		};

		unimplemented!()
	}

	async fn ime(self) -> Result<crate::bind::types::BindRules, Self::DisplayBindError> {
		unimplemented!()
	}

	type DisplayBindError = DisplayBindError;
}
