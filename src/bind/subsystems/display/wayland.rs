#[derive(Debug, thiserror::Error)]
pub enum DisplayBindError {
	#[error("I/O error: {0:#?}")]
	IOError(crate::bind::subsystems::display::ExistError),

	#[error("Could not use Wayland socket: path does not exist")]
	NonExistentError,

	#[error("Could not send environment variable: {0:#?}")]
	SendEnvError(tokio::sync::mpsc::error::SendError<crate::envs::holder::EnvMessage>),

	#[error("Error creating security context: {0:#?}")]
	WpSecurityContextV1Error(security_context::SecurityContextError),
}

mod find_socket;
mod security_context;

pub struct Wayland {
	pub runtime_dir:	std::path::PathBuf,
	pub env:		crate::envs::holder::HoldChannel,
	pub portable_runtime:	std::sync::Arc<crate::bind::subsystems::dirs::portable_runtime::PortableRuntime>,
	pub logger:		crate::logger::LogSender,
	pub app_id:		String,
	pub instance_id:	String,
}

impl super::BindDisplay for Wayland {
	async fn bind(self) -> Result<crate::bind::types::BindRules, Self::DisplayBindError> {
		let display = {
			let host_socket = find_socket::find(self.runtime_dir)
				.await
				?;
			security_context::create_context(
				host_socket,
				self.portable_runtime,
				self.logger,
				self.app_id,
				self.instance_id,
			)
				.await
				.map_err(DisplayBindError::WpSecurityContextV1Error)
				?
		};

		let mut ret: crate::bind::types::BindRules = vec![];
		ret.push(
			crate::bind::types::BindRule::Path {
				source: display,
				dest: "/run/wayland".into(),
				class: crate::bind::types::BindType::ReadOnly,
			}
		);

		self.env.send(
			crate::envs::holder::EnvMessage::Add {
				key: "WAYLAND_DISPLAY".into(),
				value: "/run/wayland".into(),
			},
		)
			.await
			.map_err(DisplayBindError::SendEnvError)
			?;

		Ok(ret)
	}

	async fn ime(self) -> Result<crate::bind::types::BindRules, Self::DisplayBindError> {
		unimplemented!()
	}

	type DisplayBindError = DisplayBindError;
}
