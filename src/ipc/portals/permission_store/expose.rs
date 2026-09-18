/*
	This module implements the Permission Store of Portable --expose switches.

	Permissions are stored as follows:
		Root
		|- table: top.kimiblock.Portable
		|--- id: expose-(rw, ro, device)
		|------ app: sandbox_id
		|--------- permissions: array of Strings (must be absolute paths!)
*/


/**
	Retrieve the permission types as PathBufs, in order of rw,ro,device.
*/
pub async fn get(
	sandbox_id:	&str,
	bus:		&zbus::Connection,
	logger:		&crate::logger::LogSender,
)
	-> Result<(Vec<std::path::PathBuf>, Vec<std::path::PathBuf>, Vec<std::path::PathBuf>), zbus::Error>
{

	let proxy = super::PermissionStoreProxy::new(bus)
		.await
		?;

	let rw = {
		let raw_rw_permissions = match proxy
			.get_permission(
				"top.kimiblock.Portable",
				"expose-rw",
				sandbox_id,
			)
			.await
		{
			Ok(v)	=> v,
			Err(e)	=> {
				let _ = logger.send(
					crate::logger::LogMessage {
						level:	crate::logger::LogLevel::Warn,
						message: format!(
							"Could not retrieve saved permission: {e:#?}"
						),
					}
				).await;
				vec![]
			}
		};

		let mut ret = vec![];

		for perm in raw_rw_permissions {
			let path = std::path::PathBuf::from(perm);

			if ! path.is_absolute() {
				return Err(
					zbus::Error::Failure(
						format!("Expected an absolute path, found {path:?}")
					)
				);
			}

			ret.push(
				path
			);
		};
		ret
	};

	let ro = {
		let raw_rw_permissions = match proxy
			.get_permission(
				"top.kimiblock.Portable",
				"expose-ro",
				sandbox_id,
			)
			.await
		{
			Ok(v)	=> v,
			Err(e)	=> {
				let _ = logger.send(
					crate::logger::LogMessage {
						level:	crate::logger::LogLevel::Warn,
						message: format!(
							"Could not retrieve saved permission: {e:#?}"
						),
					}
				).await;
				vec![]
			}
		};

		let mut ret = vec![];

		for perm in raw_rw_permissions {
			let path = std::path::PathBuf::from(perm);

			if ! path.is_absolute() {
				return Err(
					zbus::Error::Failure(
						format!("Expected an absolute path, found {path:?}")
					)
				);
			}

			ret.push(
				path
			);
		};
		ret
	};

	let device = {
		let raw_rw_permissions = match proxy
			.get_permission(
				"top.kimiblock.Portable",
				"expose-device",
				sandbox_id,
			)
			.await
		{
			Ok(v)	=> v,
			Err(e)	=> {
				let _ = logger.send(
					crate::logger::LogMessage {
						level:	crate::logger::LogLevel::Warn,
						message: format!(
							"Could not retrieve saved permission: {e:#?}"
						),
					}
				).await;
				vec![]
			}
		};

		let mut ret = vec![];

		for perm in raw_rw_permissions {
			let path = std::path::PathBuf::from(perm);

			if ! path.is_absolute() {
				return Err(
					zbus::Error::Failure(
						format!("Expected an absolute path, found {path:?}")
					)
				);
			}

			ret.push(
				path
			);
		};
		ret
	};

	Ok(
		(
			rw,
			ro,
			device,
		)
	)


}
