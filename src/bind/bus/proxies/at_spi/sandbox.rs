

/**
	Publish rules for the D-Bus proxy
*/
pub async fn get_sandbox(
	proxy_socket:	&std::path::PathBuf,
	host_socket:	std::path::PathBuf,
	flatpak_info:	std::path::PathBuf,
) -> Result<crate::bind::types::BindRules, super::AtspiError> {
	use crate::bind::types::BindRule;

	let proxy_parent_dir = match proxy_socket.parent() {
		Some(v)	=> {v.to_path_buf()}
		None	=> {
			return Err(super::AtspiError::NoParentDirectory);
		}
	};

	let ret = vec![
		BindRule::Symlink {
			source: "/usr/lib64".into(),
			dest: "/lib64".into(),
		},
		BindRule::Path {
			source: "/usr/lib".into(),
			dest: "/usr/lib".into(),
			class: crate::bind::types::BindType::ReadOnly,
		},
		BindRule::Path {
			source: "/usr/lib64".into(),
			dest: "/usr/lib64".into(),
			class: crate::bind::types::BindType::ReadOnly,
		},
		BindRule::Path {
			source: "/usr/bin".into(),
			dest: "/usr/bin".into(),
			class: crate::bind::types::BindType::ReadOnly,
		},
		BindRule::Path {
			source: "/usr/bin".into(),
			dest: "/usr/bin".into(),
			class: crate::bind::types::BindType::ReadOnly,
		},

		BindRule::Path {
			source: proxy_parent_dir.to_path_buf(),
			dest: proxy_parent_dir,
			class: crate::bind::types::BindType::ReadWrite,
		},
		BindRule::Path {
			source: host_socket.clone(),
			dest: host_socket,
			class: crate::bind::types::BindType::ReadWrite,
		},

		#[cfg(feature = "flatpak")]
		BindRule::Path {
			source: flatpak_info,
			dest: std::path::PathBuf::from("/.flatpak-info"),
			class: crate::bind::types::BindType::ReadOnly,
		}
	];

	Ok(ret)
}
