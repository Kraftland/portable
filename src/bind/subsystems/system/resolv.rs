pub async fn mount(
	logger:	crate::logger::LogSender,
) -> Vec<crate::bind::types::BindRule> {
	let attrs = match tokio::fs::symlink_metadata("/etc/resolv.conf").await {
		Ok(v)	=> v,
		Err(e)	=> {
			let _ = logger.send(
				crate::logger::LogMessage {
					level:		crate::logger::LogLevel::Warn,
					message:	format!("Could not read status of resolv.conf: {e:#?}"),
				}
			).await;
			return vec![];
		}
	};

	if attrs.is_symlink() {
		#[cfg(debug_assertions)]
		let _ = logger.send(
			crate::logger::LogMessage {
				level: crate::logger::LogLevel::Debug,
				message: format!("resolv.conf is a symlink"),
			}
		).await;

		let dest = match tokio::fs::read_link("/etc/resolv.conf").await {
			Ok(v)	=> {v}
			Err(e)	=> {
				let _ = logger.send(
					crate::logger::LogMessage {
					level:		crate::logger::LogLevel::Warn,
					message:	format!("Could not read dest of resolv.conf: {e:#?}"),
					}
				).await;
				return vec![];
			}
		};

		vec![
			crate::bind::types::BindRule::Path {
				source:	dest,
				dest:	"/etc/resolv.conf".into(),
				class:	crate::bind::types::BindType::ReadOnly,
			}
		]
	} else if attrs.is_dir() || attrs.is_file() {
		vec![
			crate::bind::types::BindRule::Path {
				source:	"/etc/resolv.conf".into(),
				dest:	"/etc/resolv.conf".into(),
				class:	crate::bind::types::BindType::ReadOnly,
			}
		]
	} else {
		let _ = logger.send(
			crate::logger::LogMessage {
				level: crate::logger::LogLevel::Warn,
				message: format!("Unknown type of resolv.conf"),
			}
		).await;
		vec![]
	}
}
