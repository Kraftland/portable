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

	ExposePermissionRetrieve(zbus::Error),

	PermissionStoreResetError{
		table:	String,
		id:	String,
		err:	zbus::fdo::Error,
	},

	InvalidUserEnvironment(String),

	ResolvUnknown,
	ResolvLinkError(std::io::Error),
	ResolvStatError(std::io::Error),
	ForwardAddFullError(crate::ipc::portals::documents::DocumentError),

	BindConsentError(crate::bind::subsystems::user::UserBindError),

	XCursorRetrieve(zbus::Error),

	X11SessionType,

	UnknownSessionType,

	WpSecurityContextManagerV1(crate::bind::subsystems::display::wayland::security_context::SecurityContextError),

	X11UnknownInputMethod,

	X11DisplayUnknown(std::env::VarError),

	X11AuthorityBindError(crate::bind::subsystems::display::x11::xauth::XAuthorityError),

	EnumerateGPUError(crate::bind::subsystems::devices::gpu::GPUError),
	GPUSpawnError(tokio::task::JoinError),
	UnknownGPUVendor,
	UnknownNVIDIADriver(Option<std::string::String>),
}

impl std::fmt::Display for WarnMessage {
	fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
		match self {
			Self::UnknownGPUVendor		=> {
				f.write_str("Unknown GPU vendor, please report a bug with PCI vendor ID")
			}
			Self::UnknownNVIDIADriver(v)	=> {
				f.write_fmt(
					format_args!("Unknown driver for NVIDIA card: {v:?}")
				)
			}
			Self::GPUSpawnError(v)		=> {
				f.write_fmt(
					format_args!("Could not spawn GPU task: {v}")
				)
			}
			Self::EnumerateGPUError(v)	=> {
				f.write_fmt(
					format_args!("Could not enumerate GPUs: {v}")
				)
			}
			Self::X11AuthorityBindError(v)	=> {
				f.write_fmt(
					format_args!("Could not bind XAUTHORITY: {v}")
				)
			}
			Self::X11DisplayUnknown(v)	=> {
				f.write_fmt(
					format_args!("Could not retrieve DISPLAY for X11: {v}")
				)
			}
			Self::X11UnknownInputMethod	=> {
				f.write_str("Could not determine input method for X11")
			}
			Self::WpSecurityContextManagerV1(v)
			=> {
				f.write_fmt(
					format_args!("Could not enable security context v1: {v}")
				)
			}
			Self::X11SessionType		=> {
				f.write_str("Detected session type as X11.")?;
				f.write_str("This is insecure: ")?;
				f.write_str("X11 allows client to access keyboard input and capture screen silently.")
			}
			Self::UnknownSessionType	=> {
				f.write_str("Could not determine session type. Did you set XDG_SESSION_TYPE?")
			}
			Self::XCursorRetrieve(v)	=> {
				f.write_fmt(
					format_args!(
						"Could not retrieve X Cursor information from Portal: {v}"
					)
				)
			}
			Self::BindConsentError(v)	=> {
				f.write_fmt(
					format_args!(
						"Could not ask for file exposure consent: {v}"
					)
				)
			}
			Self::ForwardAddFullError(v)	=> {
				f.write_fmt(
					format_args!(
						"Could not forward files: AddFull failed: {v}"
					)
				)
			}
			Self::ResolvStatError(v)	=> {
				f.write_fmt(
					format_args!(
						"Could not read status of resolv.conf: {v}"
					)
				)
			}
			Self::ResolvLinkError(v)	=> {
				f.write_fmt(
					format_args!("Could not canonicalise resolv.conf: {v}")
				)
			}
			Self::ResolvUnknown		=> {
				f.write_str("Unknown type of resolv.conf")
			}
			Self::InvalidUserEnvironment(v)	=> {
				f.write_fmt(
					format_args!("Invalid user environment: {v}")
				)
			}
			Self::PermissionStoreResetError { table, id, err }
			=> {
				f.write_fmt(
					format_args!("Could not reset permissions: table {table} ID {id}: {err}")
				)
			}

			Self::ExposePermissionRetrieve(v)
			=> {
				f.write_fmt(
					format_args!("Could not retrieve expose permissions: {v}")
				)
			}
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
