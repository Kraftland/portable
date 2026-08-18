/**
	Determine if the init version is too low, send a notification.

	Init must be the same as Daemon.
*/
pub async fn check(
	conn:		&zbus::Connection,
	init_name:	&str,
	logger:		crate::logger::LogSender,
) {
	let version = env!("CARGO_PKG_VERSION_MAJOR");

	let proxy = match IPCProxy::new(conn, init_name).await {
		Ok(v)	=> v,
		Err(e)	=> {
			let _ = logger.send(
				crate::logger::LogMessage {
					level:	crate::logger::LogLevel::Warn,
					message: format!("Could not create proxy to Init: {e:#?}"),
				}
			).await;
			return;
		}
	};

	let init_version = match proxy.version().await {
		Ok(v)	=> {v}
		Err(e)	=> {
			let _ = logger.send(
				crate::logger::LogMessage {
					level:	crate::logger::LogLevel::Warn,
					message: format!("Could not get Init version: {e:#?}"),
				}
			).await;
			return;
		}
	};

	if version == init_version {
		return;
	}

	match crate::ipc::portals::legacy_notif::notify(
		conn,
		"dialog-warning-symbolic",
		"Restart required",
		"Application is running a different major version of Portable",
	).await {
		Ok(_)	=> {}
		Err(e)	=> {
			let _ = logger.send(
				crate::logger::LogMessage {
					level:	crate::logger::LogLevel::Warn,
					message: format!("Could not notify restart: {e:#?}"),
				}
			).await;
		}
	}
}

#[zbus::proxy(
	interface	= "top.kimiblock.Portable.Init",
	default_path	= "/top/kimiblock/portable/init",
)]
trait IPC {
	#[zbus(
		name	= "Version",
		property(
			emits_changed_signal	= "const"
		)
	)]
	fn version(
		&self,
	) -> zbus::Result<String>;
}
