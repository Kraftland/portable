pub async fn generate_sandbox_rules(
	proxy_path:	std::path::PathBuf,

	#[cfg(feature = "flatpak")]
	info_path:	std::path::PathBuf,
) -> Result<crate::bind::types::BindRules, super::ProxyError> {
	use crate::bind::types::BindRule;

	let proxy_parent_dir = match proxy_path.parent() {
		Some(v)	=> {v.to_path_buf()}
		None	=> {
			return Err(super::ProxyError::NoParentDirectory);
		}
	};

	let mut rules = vec![
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
		// BindRule::Path {
		// 	source: "/usr/share".into(),
		// 	dest: "/usr/share".into(),
		// 	class: crate::bind::types::BindType::ReadOnly,
		// },
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
	];

	#[cfg(feature = "flatpak")]
	{
		rules.push(
			BindRule::Path {
				source: info_path,
				dest: std::path::PathBuf::from("/.flatpak-info"),
				class: crate::bind::types::BindType::ReadOnly,
			}
		);
	}

	{
		let env = std::env::var("DBUS_SESSION_BUS_ADDRESS")
			.map_err(super::ProxyError::AddressUnknownError)
			?;
		let path = env.strip_prefix("unix:path=");
		match path {
			Some(v)	=> {
				rules.push(
					BindRule::Path {
						source: std::path::PathBuf::from(v),
						dest: std::path::PathBuf::from(v),
						class: crate::bind::types::BindType::ReadWrite,
					}
				);
			}
			None	=> {}
		}
	};

	Ok(rules)
}
