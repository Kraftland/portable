/**
	Resets several permissions for a given application.

	Supported permissions are listed in super::PermissionType.

	Currently, the ScreenShot, Location and Notifications permission can be reset.

	This does not handle the Document Portal, because technically they are different.
*/
pub async fn reset_permissions(bus: zbus::Connection, sandbox_id: std::sync::Arc<String>) -> zbus::Result<()> {
	let proxy = super::PermissionStoreProxy::new(&bus)
		.await
		?;

	let reset_list = vec![
		super::PermissionType::ScreenShot,
		super::PermissionType::Location,
		super::PermissionType::Notifications,
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
