pub mod debug;
pub mod fatal;
pub mod info;
pub mod warn;

/**
	The central LogMessage implementation.

	It implements the std::fmt::Display trait for presentation
*/
pub enum LogMessage {
	#[cfg(debug_assertions)]
	Debug(debug::DebugMessage),

	Info(info::InfoMessage),

	Warn(warn::WarnMessage),

	Fatal(fatal::FatalMessage),
}


