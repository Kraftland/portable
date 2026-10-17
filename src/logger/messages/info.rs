pub enum InfoMessage {
	PermissionRevoked {
		config:	std::sync::Arc<crate::config::Config>,
	},

	DynamicPermissionStoreError(zbus::Error),
	ExposeDenied,

	DaemonVersion,
}

impl std::fmt::Display for InfoMessage {
	fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
		match self {
			Self::DaemonVersion			=> {
				f.write_fmt(
					format_args!(
						"Portable daemon, version {}",
						env!("CARGO_PKG_VERSION_MAJOR"),
					)
				)
			}
			Self::ExposeDenied			=> {
				f.write_str("Exposing was denied by the user")
			}
			Self::PermissionRevoked { config }	=> {
				f.write_str("Permissions for ")?;
				f.write_str(config.metadata.sandbox_id.as_str())?;
				f.write_str(" has been revoked")
			}
			Self::DynamicPermissionStoreError(v)	=> {
				f.write_fmt(format_args!("Could not retrieve Dynamic Permissions from PermissionStore: {v}"))
			}
		}
	}
}
