// use thiserror::Error;

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
) {
	use std::collections::HashMap;
	use crate::logger::LogMessage;

	let mut envs_map: HashMap<String, String> = HashMap::new();

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
				let _ = log_tx.send(
					LogMessage {
						level: crate::logger::LogLevel::Debug,
						message: format!("Environment variable set: {} {}", key, value),
					}
				).await;
				envs_map.insert(key, value);
			}
			EnvMessage::Collect { chan }	=> {
				chan.send(envs_map.to_owned())
					.expect("Could not send variables");
			}
		}
	}

}
