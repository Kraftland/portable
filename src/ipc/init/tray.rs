mod list;
mod wake;

/**
	The wake function automatically wakes up the application via D-Bus StatusNotifier

	Note that this should work for all apps using StatusNotifierItem, though we prefer legacy ones.
*/
pub async fn wake(
	config:		std::sync::Arc<crate::config::Config>,
	conn:		&zbus::Connection,
	log:		crate::logger::LogSender,
) -> Result<(), WakeError> {
	let pairs = list::list(&conn, &config.metadata.sandbox_id, &log)
		.await
		.map_err(WakeError::BusFdoError)
		?;

	for (name, path) in pairs {
		#[cfg(debug_assertions)]
		let _ = log.send(
			crate::logger::LogMessage {
				level:		crate::logger::LogLevel::Debug,
				message:	format!("Waking D-Bus remote: {name} with path {path}"),
			}
		).await;
		wake::wake_name(&conn, &name, &path)
			.await
			.map_err(WakeError::BusError)
			?;

		#[cfg(debug_assertions)]
		let _ = log.send(
			crate::logger::LogMessage {
				level:		crate::logger::LogLevel::Debug,
				message:	format!("Woke D-Bus remote: {name} with path {path}"),
			}
		).await;

	};
	Ok(())
}

#[derive(Debug, thiserror::Error)]
pub enum WakeError {
	#[error("Bus Error: {0:#?}")]
	BusError(zbus::Error),

	#[error("Bus Error: {0:#?}")]
	BusFdoError(zbus::fdo::Error),
}
