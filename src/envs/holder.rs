// use thiserror::Error;

#[derive(Debug)]
pub enum EnvMessage {
	Add		{key: String, value: String},
	Collect		{},
}

pub type HoldChannel = tokio::sync::mpsc::Sender<EnvMessage>;
pub type CollectChannel = tokio::sync::oneshot::Receiver<
	std::collections::HashMap<String, String>
>;
type HoldChannelRx = tokio::sync::mpsc::Receiver<EnvMessage>;

pub async fn new_channel() -> (HoldChannel, HoldChannelRx) {
	tokio::sync::mpsc::channel(24)
}

pub async fn holder(
	mut rx: HoldChannelRx,
) {

}
