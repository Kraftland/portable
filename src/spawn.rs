#[cfg(feature = "systemd")]
pub mod systemd;

pub mod rng;

pub mod instance_id;

pub struct Spawn {
	fs_rules:	crate::bind::types::BindRules,
	logger:		crate::logger::LogSender,
}

#[derive(thiserror::Error, Debug)]
pub enum StartAppError {}

pub trait Start {
	async fn start(&self) -> Result<(), StartAppError>;
}
