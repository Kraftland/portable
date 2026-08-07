
/**
	A generic trait for other subsystems to implement binding generation

	Portable's bind rule generation system is divided to multiple subsystems. Each of them may
	implement different functions and are generally controlled via Cargo feature switches.

	Every subsystem has a unique struct to pass along information.
*/
pub trait GenerateBind {
	fn bind(self) -> impl std::future::Future<Output = Result<super::types::BindRules, Self::BindError>> + Send;

	type BindError;
}

#[cfg(feature = "audio")]
pub mod audio;

#[cfg(feature = "devices")]
pub mod devices;

pub mod dirs;

#[cfg(feature = "display")]
pub mod display;
