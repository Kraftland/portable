#[derive(Debug, thiserror::Error)]
pub enum DisplayBindError {
	#[error("Could not bind X11 display: {0:#?}")]
	SpawnError(tokio::task::JoinError),

	#[error("I/O error: {0:#?}")]
	IOError(crate::bind::display::ExistError),

	#[error("Could not send environment variable: {0:#?}")]
	SendEnvError(tokio::sync::mpsc::error::SendError<crate::envs::holder::EnvMessage>),
}

#[derive(Clone)]
pub struct X11 {
	pub logger:		crate::logger::LogSender,
	pub home:		std::path::PathBuf,
	pub env:		crate::envs::holder::HoldChannel,
}

pub mod xauth;
pub mod im;

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

		match super::exists("/tmp/.X11-unix".into()).await.map_err(DisplayBindError::IOError)? {
			true	=> {
				ret.push(
					crate::bind::types::BindRule::Path {
						source: "/tmp/.X11-unix".into(),
						dest: "/tmp/.X11-unix".into(),
						class: crate::bind::types::BindType::ReadWrite,
					},
				);
			}
			false	=> {}
		};

		Ok(ret)
	}

	async fn ime(self) -> Result<crate::bind::types::BindRules, Self::DisplayBindError> {
		self.env.send(
			crate::envs::holder::EnvMessage::Add {
				key: "IBUS_USE_PORTAL".into(),
				value: "1".into(),
			}
		)
			.await
			.map_err(DisplayBindError::SendEnvError)
			?;

		use std::collections::HashMap;

		let envs: HashMap<&str, &str> = match im::detect_kind().await {
			im::InputMethodKind::IBus	=> {
				let mut map = HashMap::new();
				map.insert(
					"QT_IM_MODULES",
					"wayland;ibus",
				);
				map.insert(
					"QT_IM_MODULE",
					"ibus",
				);
				map
			}
			im::InputMethodKind::Fcitx	=> {
				let mut map = HashMap::new();

				map.insert("QT_IM_MODULES", "wayland;fcitx");
				map.insert("QT_IM_MODULE", "fcitx");

				map
			}
			im::InputMethodKind::Gcin	=> {
				let mut map = HashMap::new();

				map.insert("QT_IM_MODULES", "wayland;ibus");
				map.insert("QT_IM_MODULE", "ibus");

				map
			}
			im::InputMethodKind::Unknown	=> {
				let _ = self.logger.send(
					crate::logger::LogMessage {
						level: crate::logger::LogLevel::Warn,
						message: format!("Could not determine Input Method type"),
					},
				).await;

				HashMap::new()
			}
		};

		for (k, v) in envs {
			self.env.send(
				crate::envs::holder::EnvMessage::Add {
					key: k.into(),
					value: v.into(),
				},
			).await.map_err(DisplayBindError::SendEnvError)?
		};

		match super::exists("/tmp/.XIM-unix".into()).await.map_err(DisplayBindError::IOError)? {
			true	=> {
				Ok(vec![
					crate::bind::types::BindRule::Path {
						source: "/tmp/.XIM-unix".into(),
						dest: "/tmp/.XIM-unix".into(),
						class: crate::bind::types::BindType::ReadWrite,
					}
				])
			}
			false	=> {
				Ok(vec![])
			}
		}
	}

	type DisplayBindError = DisplayBindError;
}
