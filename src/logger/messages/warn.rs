pub enum WarnMessage {
	DebugBuild,
	DynamicPermissionStoreRetrieveError(
		crate::pref::permissions::consent::impls::store::portal::PortalPermissionStoreError,
	),
	UnknownArgument(String),
	ConsoleRawError(crate::spawn::stream::StreamError),

	#[cfg(feature = "flatpak")]
	InstanceIDCollided(u32),

	WindowSize(nix::Error),

	InvalidUnitState(String),

	UnitFailed,

	RestartNotifierError(zbus::Error),

	InitVersionMismatch,

	InitProxy(zbus::Error),

	InitVersionError(zbus::Error),
}

impl std::fmt::Display for WarnMessage {
	fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
		match self {
			Self::InitProxy(v)		=> {
				f.write_fmt(
					format_args!("Could not create proxy to sandbox Init: {v}")
				)
			}
			Self::InitVersionError(v)	=> {
				f.write_fmt(
					format_args!("Could not retrieve sandbox Init version: {v}")
				)
			}
			Self::InitVersionMismatch	=> {
				f.write_str("Sandbox is running a different major version.")
			}
			Self::RestartNotifierError(v)	=> {
				f.write_fmt(
					format_args!("Could not present restart notifier: {v}")
				)
			}
			Self::UnitFailed		=> {
				f.write_str("Unit has failed. Either the application is malfunctioning, or the system did not meet minium requirements.")
			}
			Self::InvalidUnitState(v)	=> {
				f.write_fmt(
					format_args!(
						"Unexpected systemd unit state: {v}"
					)
				)
			}
			Self::WindowSize(v)		=> {
				f.write_fmt(
					format_args!("Could not get console window size: {v}")
				)
			}
			Self::InstanceIDCollided(v)	=> {
				f.write_fmt(
					format_args!("Generated instance ID collided with Flatpak: {v}")
				)
			}
			WarnMessage::ConsoleRawError(v)	=> {
				f.write_fmt(
					format_args!(
						"Could not enable RAW console mode: {v}"
					)
				)
			}
			WarnMessage::DebugBuild		=> {
				f.write_str("Build type is debug, expect slowdown and verbose logging")
			}
			WarnMessage::DynamicPermissionStoreRetrieveError(v)
			=> {
				f.write_fmt(
					format_args!("Could not retrieve stored permissions: {v}"),
				)
			}
			WarnMessage::UnknownArgument(v)
			=> {
				f.write_fmt(format_args!("Unrecognised argument: {v}"))
			}
		}
	}
}
