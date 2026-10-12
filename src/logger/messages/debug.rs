/**
	A collection of debugging messages
*/
#[cfg(debug_assertions)]
pub enum DebugMessage {
	DecodedConfig(std::sync::Arc<crate::config::Config>),
}

impl std::fmt::Display for DebugMessage {
	fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
		match self {
			DebugMessage::DecodedConfig(v)	=> {
				f.write_fmt(format_args!("Resolved configuration: {:#?}", v))
			}
		}
	}
}
