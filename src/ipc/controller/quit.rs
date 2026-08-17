/**
	Call the controller to stop the sandbox

	It does not guarantee that the specified cgroup is killed before returning
*/
pub async fn stop_app(app_id: &str, bus: &zbus::Connection) -> Result<(), StopError> {
	let name = {
		let mut name = String::new();
		name.push_str("top.kimiblock.portable.");
		name.push_str(app_id);
		name
	};

	let proxy = IPCProxy::new(bus, name)
		.await
		.map_err(StopError::ProxyError)
		?;

	proxy
		.stop()
		.await
		.map_err(StopError::ControllerError)
}

#[derive(thiserror::Error, Debug)]
pub enum StopError {
	#[error("Could not contact remote controller: {0:#?}")]
	ControllerError(zbus::Error),

	#[error("Could not create proxy: {0:#?}")]
	ProxyError(zbus::Error),
}

#[zbus::proxy(
	interface	= "top.kimiblock.Portable.Controller",
	default_path	= "/top/kimiblock/portable/daemon",
)]
trait IPC {
	#[zbus(
		name	= "Stop",
		no_reply,
		no_autostart,
	)]
	async fn stop(&self) -> zbus::Result<()>;
}
