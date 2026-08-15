/**
	The wake function automatically wakes up the application via D-Bus StatusNotifier

	Note that this should work for all apps using StatusNotifierItem, though we prefer legacy ones.
*/
pub async fn wake(
	config:		std::sync::Arc<crate::config::Config>,
	conn:		&zbus::Connection,
) -> Result<(), WakeError> {
	let bus_name = {
		let mut name = String::from(&config.metadata.sandbox_id);
		name.push_str(".Portable.Helper");
		name
	};

	let proxy = IPCProxy::new(conn, bus_name)
		.await
		.map_err(WakeError::BusError)
		?;

	proxy
		.activate()
		.await
		.map_err(WakeError::BusError)
}

#[derive(Debug, thiserror::Error)]
pub enum WakeError {
	#[error("Bus Error: {0:#?}")]
	BusError(zbus::Error),
}

#[zbus::proxy(
	interface	= "top.kimiblock.Portable.Init",
	default_path	= "/top/kimiblock/portable/init",
)]
trait IPC {
	#[zbus(name = "ActivateTray")]
	async fn activate(&self) -> zbus::Result<()>;
}
