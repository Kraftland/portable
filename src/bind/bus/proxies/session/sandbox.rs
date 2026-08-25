pub async fn generate_sandbox_rules(
	proxy_path:	std::path::PathBuf,

	app_id:		String,

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
		BindRule::VirtualFS {
			dest:	{
					let mut name = String::from("top.kimiblock.portable.");
					name.push_str(&app_id);
					name
				}.into(),
			class:	crate::bind::types::VirtualFS::Tmpfs {
				size_mb:	Some(1),
				perms:		None,
			},
		},

		BindRule::Path {
			source:	"/usr".into(),
			dest:	"/usr".into(),
			class:	crate::bind::types::BindType::ReadOnly,
		},
		BindRule::Symlink {
			source:	"/usr/lib".into(),
			dest:	"/lib".into(),
		},
		BindRule::Symlink {
			source:	"/usr/lib".into(),
			dest:	"/lib64".into(),
		},
		BindRule::Symlink {
			source:	"/usr/bin".into(),
			dest:	"/bin".into(),
		},
		BindRule::Symlink {
			source:	"/usr/bin".into(),
			dest:	"/sbin".into(),
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
