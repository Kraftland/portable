pub async fn share_path_with_helper(
	bus_conn:	&zbus::Connection,
	directory:	bool,
) -> Result<(), ShareError> {
	unimplemented!()
}

/**
	Call the NameHasOwner function to see if Init is alive
*/
async fn helper_is_alive(app_id: &str, conn: &zbus::Connection) -> Result<bool, ShareError> {
	let proxy = SessionBusProxy::new(conn)
		.await
		.map_err(ShareError::ProxyError)
		?;

	let mut name = String::from(app_id);
	name.push_str(".Portable.Helper");

	proxy.name_has_owner(name)
		.await
		.map_err(ShareError::OwnerError)
}

#[zbus::proxy(
	interface	= "org.freedesktop.DBus",
	default_path	= "/org/freedesktop/DBus",
	default_service	= "org.freedesktop.DBus"
)]
trait SessionBus {
	#[zbus(name = "NameHasOwner")]
	async fn name_has_owner(&self, name: String) -> zbus::Result<bool>;
}

#[zbus::proxy(
	interface	= "top.kimiblock.Portable.Init",
	default_path	= "/top/kimiblock/portable/init",
)]
trait IPC {
	#[zbus(
		name = "RequestFSAccess",
		no_autostart,
	)]
	async fn request_fs(&self, directory: bool) -> zbus::Result<()>;
}

#[derive(thiserror::Error, Debug)]
pub enum ShareError {
	#[error("Could not call NameHasOwner on bus: {0:#?}")]
	OwnerError(zbus::Error),

	#[error("Could not create proxy: {0:#?}")]
	ProxyError(zbus::Error),
}
