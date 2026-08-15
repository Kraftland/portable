#[cfg(feature = "pulseaudio")]
pub mod pulse;
#[cfg(feature = "pulseaudio")]
pub use pulse::*;

pub struct Audio {
	pub logger:		crate::logger::LogSender,
	pub runtime_dir:	std::path::PathBuf,
	pub env:		crate::envs::holder::HoldChannel,
}


