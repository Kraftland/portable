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
	interface	= "oorg.freedesktop.impl.portal.PermissionStore",
	default_service	= "org.freedesktop.portal.Desktop",
	default_path	= "/org/freedesktop/impl/portal/PermissionStore",
	gen_async	= true,
	gen_blocking	= false,
)]
trait PermissionStore {
	async fn list(&self, table: &str) -> zbus::fdo::Result<Vec<String>>;
}
