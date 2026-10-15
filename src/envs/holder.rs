use thiserror::Error;

#[derive(Error, Debug)]
pub enum EnvError {
	#[error("Could not sync environment variables: channel error: {0:#?}")]
	ChannelError(tokio::sync::mpsc::error::SendError<EnvMessage>),

	#[error("Could not acquire environment variables: channel error: {0:#?}")]
	RecvError(tokio::sync::oneshot::error::RecvError),
}

#[derive(Debug)]
pub enum EnvMessage {
	Add		{key: String, value: String},
	Collect		{chan: CollectChannel},
}

pub type HoldChannel = tokio::sync::mpsc::Sender<EnvMessage>;
pub type CollectChannel = tokio::sync::oneshot::Sender<
	std::collections::HashMap<String, String>
>;
type HoldChannelRx = tokio::sync::mpsc::Receiver<EnvMessage>;

pub async fn new_channel() -> (HoldChannel, HoldChannelRx) {
	tokio::sync::mpsc::channel(24)
}

pub async fn holder(
	mut rx: HoldChannelRx,
	log_tx: crate::logger::LogSender,
	xdg:	std::sync::Arc<crate::xdg::XdgDirs>,
	config:	std::sync::Arc<crate::config::Config>,
) {
	use crate::logger::messages::LogMessage;

	let mut envs_map = super::forward::get();

	match super::user::load_user_envs(&mut envs_map, xdg.clone(), config.clone(), &log_tx).await {
		Ok(_)	=> {}
		Err(e)	=> {
			let _ = log_tx.send(
				LogMessage::Warn(
					crate::logger::messages::warn::WarnMessage::UserEnvLoadError(e)
				)
			).await;
		}
	};

	match super::xdg::generate_xdg_envs(&mut envs_map, xdg, config).await {
		Ok(_)	=> {}
		Err(e)	=> {
			let _ = log_tx.send(
				LogMessage::Warn(
					crate::logger::messages::warn::WarnMessage::XdgEnvLoadError(e)
				)
			).await;
		}
	};

	loop {
		let msg = tokio::select! {
			msg	= rx.recv()	=> {msg}
		};
		let msg = match msg {
			Some(v)	=> {v}
			None	=> {return;}
		};
		match msg {
			EnvMessage::Add { key, value }	=> {
				match envs_map.contains_key(&key) {
					true	=> {continue;}
					false	=> {}
				};

				#[cfg(debug_assertions)]
				let _ = log_tx.send(
					LogMessage::Debug(
						crate::logger::messages::debug::DebugMessage::EnvSet {
							key:	key.clone(),
							val:	value.clone(),
						}
					)
				).await;

				envs_map.insert(key, value);
			}
			EnvMessage::Collect { chan }	=> {
				let result = chan.send(envs_map);
				match result {
					Ok(_)	=> {}
					Err(_)	=> {
						let _ = log_tx.send(
							LogMessage::Fatal(
								crate::logger::messages::fatal::FatalMessage::SendEnvs
							)
						).await;
					}
				}
				rx.close();
				return;
			}
		}
	}

}

pub async fn retrieve(ch: HoldChannel) -> Result<std::collections::HashMap<String, String>, EnvError> {
	let (tx, rx) = tokio::sync::oneshot::channel();
	ch.send(
		EnvMessage::Collect { chan: tx }
	)
		.await
		.map_err(EnvError::ChannelError)?;
	rx
		.await
		.map_err(EnvError::RecvError)
}
