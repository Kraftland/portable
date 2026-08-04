#[cfg(feature = "pulseaudio")]
pub mod pulse;
#[cfg(feature = "pulseaudio")]
pub use pulse::*;

pub struct Audio;

pub trait GetAddress {
	async fn get(
		logger:		crate::logger::LogSender,
		runtime_dir:	std::path::PathBuf,
	)	-> Option<std::path::PathBuf>;
}
