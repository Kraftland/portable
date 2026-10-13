/**
	Resets several permissions for a given application.

	Supported permissions are listed in super::PermissionType.

	Currently, Portable's expose permissions,
	along with ScreenShot, Location and Notifications permissions can be reset.

	This does not handle the Document Portal, because technically they are different.
*/
pub async fn reset_permissions(
	bus:		zbus::Connection,
	sandbox_id:	std::sync::Arc<String>,
	logger:		crate::logger::LogSender,
) -> zbus::Result<()> {
	let proxy = super::PermissionStoreProxy::new(&bus)
		.await
		?;

	let reset_list = vec![
		super::PermissionType::XDPScreenShot,
		super::PermissionType::XDPLocation,
		super::PermissionType::XDPNotifications,
		super::PermissionType::ExposeRW,
		super::PermissionType::ExposeRO,
		super::PermissionType::ExposeDevice,
		super::PermissionType::DynamicPermissions,
	];

	for item in reset_list {
		match proxy.delete_permission(
			item.table(),
			item.id(),
			&sandbox_id,
		)
			.await
		{
			Ok(_)	=> {
				#[cfg(debug_assertions)]
				let _ = logger.send(
					crate::logger::messages::LogMessage::Debug(
						crate::logger::messages::debug::DebugMessage::PermissionReset {
							table:	item.table().to_string(),
							id:	item.id().to_string(),
						}
					)
				).await;
			}
			Err(e)	=> {
				use crate::logger::messages::warn::WarnMessage;
				let _ = logger.send(
					crate::logger::messages::LogMessage::Warn(
						WarnMessage::PermissionStoreResetError {
							table:	item.table().to_string(),
							id:	item.id().to_string(),
							err:	e,
						}
					)
				).await;
			}
		}
	}

	Ok(())
}
