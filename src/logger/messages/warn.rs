pub enum WarnMessage {
	DebugBuild,
	DynamicPermissionStoreRetrieveError(
		crate::pref::permissions::consent::impls::store::portal::PortalPermissionStoreError,
	),
	UnknownArgument(String),
}

impl std::fmt::Display for WarnMessage {
	fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
		match self {
			WarnMessage::DebugBuild	=> {
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
