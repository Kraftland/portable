/**
	Resets several permissions for a given application.

	Supported permissions are listed in super::PermissionType.

	Currently, Portable's expose permissions,
	along with ScreenShot, Location and Notifications permissions can be reset.

	This does not handle the Document Portal, because technically they are different.
*/
pub async fn reset_permissions(bus: zbus::Connection, sandbox_id: std::sync::Arc<String>) -> zbus::Result<()> {
	let proxy = super::PermissionStoreProxy::new(&bus)
		.await
		?;

	let reset_list = vec![
		super::PermissionType::XDPScreenShot,
		super::PermissionType::XDPLocation,
		super::PermissionType::XDPNotifications,
	];

	for item in reset_list {
		proxy.delete_permission(
			item.table(),
			item.id(),
			&sandbox_id,
		)
			.await
			?
	}

	Ok(())
}
