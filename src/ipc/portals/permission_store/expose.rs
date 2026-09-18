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
	Retrieve the permission types as PathBufs, in order of rw,ro,device
*/
pub async fn get(
	sandbox_id:	&str,
	bus:		&zbus::Connection,
)
	-> Result<(Vec<std::path::PathBuf>, Vec<std::path::PathBuf>, Vec<std::path::PathBuf>), zbus::Error>
{

	let proxy = super::PermissionStoreProxy::new(bus)
		.await
		?;

	let rw = {
		let raw_rw_permissions = proxy
			.get_permission(
				"top.kimiblock.Portable",
				"expose-rw",
				sandbox_id,
			)
			.await
			?;

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
		let raw_rw_permissions = proxy
			.get_permission(
				"top.kimiblock.Portable",
				"expose-ro",
				sandbox_id,
			)
			.await
			?;

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
		let raw_rw_permissions = proxy
			.get_permission(
				"top.kimiblock.Portable",
				"expose-device",
				sandbox_id,
			)
			.await
			?;

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
