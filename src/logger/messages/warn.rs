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
		table:	Box<String>,
		id:	Box<String>,
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
	DevNVIDIAScanError(std::io::Error),
	BootVGADeprecated(
		std::path::PathBuf,
	),
	MissingIDPath(std::ffi::OsString),
	UnknownDeviceGPUType(std::ffi::OsString),
	CardRendererNodeMissing(std::ffi::OsString),

	PulseAddressError(crate::bind::subsystems::audio::PulseError),
	ActivatePulseError(crate::bind::subsystems::audio::PulseError),

	A11yBusSpawnError(crate::bind::bus::proxies::start::StartProxyError),

	A11yBusBuildError(crate::bind::bus::proxies::StartProxyError),

	UserEnvLoadError(crate::envs::user::LoadError),
	XdgEnvLoadError(crate::envs::xdg::XDGEnvError),

	DynamicPermissionsError(crate::pref::permissions::consent::ConsentError),
}

impl std::fmt::Display for WarnMessage {
	fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
		match self {
			Self::DynamicPermissionsError(v)
			=> {
				f.write_fmt(
					format_args!("Could not retrieve dynamic permissions: {v}")
				)
			}
			Self::XdgEnvLoadError(v)	=> {
				f.write_fmt(
					format_args!("Could not load XDG environment variables: {v}")
				)
			}
			Self::UserEnvLoadError(v)	=> {
				f.write_fmt(
					format_args!(
						"Could not load user environment variables: {v}"
					)
				)
			}
			Self::A11yBusBuildError(v)	=> {
				f.write_fmt(
					format_args!(
						"Could not build accessibility proxy: {v}"
					)
				)
			}
			Self::A11yBusSpawnError(v)	=> {
				f.write_fmt(
					format_args!(
						"Could not spawn accessibility proxy: {v}"
					)
				)
			}
			Self::PulseAddressError(v)	=> {
				f.write_fmt(
					format_args!("Could not parse PulseAudio address: {v}")
				)
			}
			Self::ActivatePulseError(v)	=> {
				f.write_fmt(
					format_args!(
						"Could not activate the PulseAudio server: {v}. "
					)
				)?;

				f.write_str("You may experience audio issues if PulseAudio is not running")
			}
			Self::CardRendererNodeMissing(v)
			=> {
				f.write_fmt(
					format_args!(
						"{v:?} is missing the card / renderer node"
					)
				)
			}
			Self::UnknownDeviceGPUType(v)	=> {
				f.write_fmt(
					format_args!("Device {v:?} has unknown type")
				)
			}
			Self::MissingIDPath(v)		=> {
				f.write_fmt(
					format_args!("Device {v:?} is missing ID_PATH property")
				)
			}
			Self::BootVGADeprecated(v)	=> {
				f.write_fmt(
					format_args!(
						"Device {v:?} is using deprecated boot_vga property"
					)
				)
			}
			Self::DevNVIDIAScanError(v)	=> {
				f.write_fmt(
					format_args!("Could not scan /dev for NVIDIA devices: {v}")
				)
			}
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
