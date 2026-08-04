#[cfg(feature = "pulseaudio")]
pub mod pulse;
#[cfg(feature = "pulseaudio")]
pub use pulse::*;

pub struct Audio;

pub trait GetAddress {
	fn get(
		logger:		crate::logger::LogSender,
		runtime_dir:	std::path::PathBuf,
	)	-> impl std::future::Future<Output = Option<std::path::PathBuf>> + Send;
}
