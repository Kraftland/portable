/*
	The Portal backend stores permissions in a XDG Desktop Portal PermissionStore instance.

	It uses the defined table and permission ID in ipc/portals/permission_store/types.
	Serialising each permission to a String, and stores them as array of Strings in the Store.

	Allow status is stored within the same String, suffixing the original permission bits as :true
*/
pub struct Portal {
	pub bus:	zbus::Connection,
	pub logger:	crate::logger::LogSender,
}

#[derive(Debug, thiserror::Error)]
pub enum PortalPermissionStoreError {
	#[error("Could not convert permission {original} into DynamicPermission")]
	ConvertStrError{
		original:	String,
		error:		portable_config::definitions::consent::from_str::ConversionError,
	},

	#[error("Could not contact PermissionStore via D-Bus: {0:#?}")]
	BusError(zbus::Error),
}

impl crate::pref::permissions::consent::PermissionStore for Portal {
	type StoreError = PortalPermissionStoreError;

	async fn store(
		&self,
		perms:	&crate::pref::permissions::consent::DynamicPermissionsResult,
		app_id:	&str,
	) -> Result<(), Self::StoreError> {
		let permission_type = crate::ipc::portals::permission_store::PermissionType::DynamicPermissions;

		let proxy = crate::ipc::portals::permission_store::PermissionStoreProxy::new(&self.bus)
			.await
			.map_err(PortalPermissionStoreError::BusError)
			?;

		let permissions = {
			let mut as_ret = vec![];

			for perm in perms {
				let mut perm_str = String::new();

				perm_str.push_str(perm.0.id());

				match perm.1 {
					true	=> {
						perm_str.push_str(":true");
					}
					false	=> {}
				}

				as_ret.push(perm_str);
			};

			as_ret
		};

		proxy.set_permission(
			permission_type.table(),
			true,
			permission_type.id(),
			&app_id,
			permissions,
		)
			.await
			.map_err(PortalPermissionStoreError::BusError)
			?;

		Ok(())
	}

	async fn retrieve(
		&self,
		app_id:	&str,
	) -> Result<crate::pref::permissions::consent::DynamicPermissionsResult, Self::StoreError> {
		let permission_type = crate::ipc::portals::permission_store::PermissionType::DynamicPermissions;

		let proxy = crate::ipc::portals::permission_store::PermissionStoreProxy::new(&self.bus)
			.await
			.map_err(PortalPermissionStoreError::BusError)
			?;

		let retrieved_strs = match proxy.get_permission(
			permission_type.table(),
			permission_type.id(),
			&app_id,
		).await {
			Ok(v)	=> v,
			Err(e)	=> {
				let _ = self.logger.send(
					crate::logger::LogMessage {
						level:		crate::logger::LogLevel::Debug,
						message:	format!(
							"Could not retrieve permission from PermissionStore: {e}. Treating as empty.",
						)
					}
				).await;
				vec![]
			}
		};

		let mut ret = vec![];

		for permission in retrieved_strs {
			ret.push(
				str_to_permission(permission.as_str())?
			);
		};

		Ok(ret)
	}
}

fn str_to_permission(value: &str) -> Result<(portable_config::definitions::consent::DynamicPermission, bool), PortalPermissionStoreError> {
		use portable_config::definitions::consent::DynamicPermission;
		match value.strip_suffix(":true") {
			Some(v)	=> {
				let permission = match DynamicPermission::try_from(v) {
					Ok(v)	=> v,
					Err(e)	=> {
						return Err(
							PortalPermissionStoreError::ConvertStrError {
								original: value.to_string(),
								error: e,
							}
						);
					}
				};

				Ok(
					(
						permission,
						true,
					)
				)
			}
			None	=> {
				let permission = match DynamicPermission::try_from(value) {
					Ok(v)	=> v,
					Err(e)	=> {
						return Err(
							PortalPermissionStoreError::ConvertStrError {
								original: value.to_string(),
								error: e,
							}
						);
					}
				};

				Ok(
					(
						permission,
						false,
					)
				)
			}
		}
}
