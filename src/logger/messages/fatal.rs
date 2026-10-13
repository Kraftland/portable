pub enum FatalMessage {
	LogError(Box<tokio::sync::mpsc::error::SendError<crate::logger::messages::LogMessage>>),
	UnitStateRetrieve(zbus::Error),
}

impl std::fmt::Display for FatalMessage {
	fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
		match self {
			Self::UnitStateRetrieve(v)	=> {
				f.write_fmt(
					format_args!("Could not retrieve unit state: {v}")
				)
			}
			Self::LogError(v)		=> {
				f.write_fmt(format_args!("Could not send to logger: {v}"))
			}
		}
	}
}
