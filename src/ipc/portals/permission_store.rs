mod types;
mod reset;
pub mod expose;

pub use reset::reset_permissions;

/**
	PermissionType designates a specific permission.

	Various useful functions are implemented in the types module.
	Such as the table and type method.

	XDP means Portal permissions, while others are Portable permissions.
*/
pub enum PermissionType {
	XDPScreenShot,
	XDPLocation,
	XDPNotifications,
}

/**
	Given a table ID, list enabled permissions.

	The method of storing permissions are backend-specific,
	but it should always be array of strings.
*/
pub async fn list(bus: &zbus::Connection, table: &str) -> zbus::Result<Vec<String>> {
	let proxy = PermissionStoreProxy::new(&bus)
		.await
		?;

	let vector = proxy
		.list(table)
		.await
		?;

	Ok(vector)
}


#[zbus::proxy(
	interface	= "org.freedesktop.impl.portal.PermissionStore",
	default_service	= "org.freedesktop.impl.portal.PermissionStore",
	default_path	= "/org/freedesktop/impl/portal/PermissionStore",
	gen_async	= true,
	gen_blocking	= false,
)]
pub trait PermissionStore {
	#[zbus(
		name	= "List"
	)]
	async fn list(&self, table: &str) -> zbus::fdo::Result<Vec<String>>;
	#[zbus(
		name	= "DeletePermission"
	)]
	async fn delete_permission(&self, table: &str, id: &str, app: &str) -> zbus::fdo::Result<()>;

	#[zbus(
		name	= "SetPermission"
	)]
	/// Sets the permissions for an application and a resource in the given table.
	async fn set_permission(
		&self,
		table:	&str,
		create:	bool,
		id:	&str,
		app_id:	&str,
		perms:	Vec<String>,
	) -> zbus::Result<()>;

	#[zbus(
		name	= "GetPermission"
	)]
	/// Gets the entry for an application and a resource in the given table.
	async fn get_permission(
		&self,
		table:	&str,
		id:	&str,
		app_id:	&str,
	) -> zbus::Result<Vec<String>>;
}
