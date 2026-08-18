
/**
	Wait for the unit to exit with D-Bus
*/
pub async fn wait(
	conn:		&zbus::Connection,
	escaped_path:	zbus::zvariant::OwnedObjectPath,
	cancel_token:	tokio_util::sync::CancellationToken,
	logger:		crate::logger::LogSender,
) -> Result<(), zbus::Error> {
	#[cfg(debug_assertions)]
	let _ = logger.send(
		crate::logger::LogMessage {
			level:		crate::logger::LogLevel::Debug,
			message:	format!("Listening on object path: {escaped_path:?}"),
		}
	).await;

	let proxy = PropertiesProxy::new(conn, escaped_path)
		.await
		?;

	let mut stream = proxy.receive_active_state_changed().await;

	tokio::spawn(
		async move {
			let mut activated = false;

			use futures_util::stream::StreamExt;
			while let Some(v) = stream.next().await {
				let state = match v.get().await {
					Ok(v)	=> {ActiveState::from(v.as_str())}
					Err(e)	=> {
						let _ = logger.send(
							crate::logger::LogMessage {
								level:	crate::logger::LogLevel::Fatal,
								message: format!(
									"Could not get ActiveState: {e:#?}",
								),
							}
						).await;
						cancel_token.cancel();
						return;
					}
				};

				match state {
					ActiveState::Active	=> {
						activated = true;
						#[cfg(debug_assertions)]
						let _ = logger.send(
							crate::logger::LogMessage {
								level:	crate::logger::LogLevel::Debug,
								message: format!(
									"Unit is alive",
								),
							}
						).await;
					}
					ActiveState::Failed	=> {
						let _ = logger.send(
							crate::logger::LogMessage {
								level:	crate::logger::LogLevel::Warn,
								message: format!(
									"Unit has failed",
								),
							}
						).await;
						break;
					}
					ActiveState::Inactive	=> {

						#[cfg(debug_assertions)]
						let _ = logger.send(
							crate::logger::LogMessage {
								level:	crate::logger::LogLevel::Debug,
								message: format!(
									"Unit is inactive",
								),
							}
						).await;

						if activated {
							break;
						}
					}
					ActiveState::Others { state }
								=> {
						let _ = logger.send(
							crate::logger::LogMessage {
								level:	crate::logger::LogLevel::Warn,
								message: format!(
									"Unknown active state: {state}",
								),
							}
						).await;
					}
				}
			}

			cancel_token.cancel();
		}
	);

	Ok(())
}

#[derive(Debug)]
enum ActiveState {
	Active,
	/**
		The Inactive state includes inactive (dead) and deacivating
	*/
	Inactive,
	Failed,
	Others { state: String },
}

impl From<&str> for ActiveState {
	fn from(value: &str) -> Self {
		match value {
			"active"	=> {ActiveState::Active}
			"inactive"	=> {ActiveState::Inactive}
			"failed"	=> {ActiveState::Failed}
			"deactivating"	=> {ActiveState::Inactive}
			_v		=> {ActiveState::Others { state: _v.to_string() }}
		}
	}
}

#[zbus::proxy(
	interface	= "org.freedesktop.systemd1.Unit",
	default_service	= "org.freedesktop.systemd1",
)]
trait Properties {
	#[zbus(
		name	= "ActiveState",
		property
	)]
	fn active_state(&self) -> zbus::fdo::Result<String>;
}
