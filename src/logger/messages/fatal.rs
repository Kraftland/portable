pub enum FatalMessage {
	LogError(Box<tokio::sync::mpsc::error::SendError<crate::logger::messages::LogMessage>>),
	UnitStateRetrieve(zbus::Error),
	ProxyFailed(std::io::Error),
	SendEnvs,
	StartError(crate::errors::StartError),
}

impl std::fmt::Display for FatalMessage {
	fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
		match self {
			Self::StartError(v)	=> {
				f.write_fmt(
					format_args!("{v}")
				)
			}
			Self::SendEnvs		=> {
				f.write_str(
					"Could not send environment variables: An unsuccessful send would be one where the corresponding receiver has already been deallocated."
				)
			}
			Self::ProxyFailed(v)	=> {
				f.write_fmt(
					format_args!("Could not start D-Bus proxy: {v}")
				)
			}
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
