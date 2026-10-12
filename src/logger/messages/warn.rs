pub enum WarnMessage {
	DebugBuild,
}

impl std::fmt::Display for WarnMessage {
	fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
		match self {
			WarnMessage::DebugBuild	=> {
				f.write_str("Build type is debug, expect slowdown and verbose logging")
			}
		}
	}
}
