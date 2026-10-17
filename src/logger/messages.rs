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

impl std::fmt::Display for LogMessage {
	fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
		match self {
			#[cfg(debug_assertions)]
			LogMessage::Debug(v)	=> {
				v.fmt(f)
			}
			LogMessage::Info(v)	=> {
				v.fmt(f)
			}
			LogMessage::Warn(v)	=> {
				v.fmt(f)
			}
			LogMessage::Fatal(v)	=> {
				v.fmt(f)
			}
		}
	}
}
