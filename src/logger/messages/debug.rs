/**
	A collection of debugging messages
*/
#[cfg(debug_assertions)]
pub enum DebugMessage {
	FoundConfig(std::path::PathBuf),
	DecodedConfig(std::sync::Arc<crate::config::Config>),
	DynamicPermissionsRetrieved(String),
	PtyAllocated {
		master:	std::os::fd::RawFd,
		slave:	std::os::fd::RawFd,
	},
}

impl std::fmt::Display for DebugMessage {
	fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
		match self {
			Self::PtyAllocated { master, slave }
			=> {
				f.write_fmt(
					format_args!(
						"Allocated pty pair for console streaming: {:?}:{:?}",
						master,
						slave,
					)
				)
			}
			DebugMessage::FoundConfig(v)	=> {
				f.write_fmt(format_args!("Selected configuration: {v:?}"))
			}
			DebugMessage::DecodedConfig(v)	=> {
				f.write_fmt(format_args!("Resolved configuration: {:#?}", v))
			}
			DebugMessage::DynamicPermissionsRetrieved(v)
			=> {
				f.write_fmt(
					format_args!(
						"Retrieved Dynamic Permissions from PermissionStore: {}",
						v,
					),
				)
			}
		}
	}
}
