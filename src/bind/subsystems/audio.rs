#[cfg(feature = "pulseaudio")]
pub mod pulse;
#[cfg(feature = "pulseaudio")]
pub use pulse::*;

pub struct Audio {
	logger:		crate::logger::LogSender,
	runtime_dir:	std::path::PathBuf,
	env:		crate::envs::holder::HoldChannel,
}


