/**
	Validates if the permission can be restored without calling user consent.

	It does not overwrite with new permissions, but appends in place.

	Also returns Ok(true) when the list is empty.

	It only validates a specific kind, MountPath. As Passthrough is handled via
	Dynamic Permission.

	If not then update the permission store and return false.
*/
pub async fn validate_stored_permission(
	dbus_conn:	&zbus::Connection,
	logger:		&crate::logger::LogSender,
	app_id:		&str,
	expose_list:	&Vec<crate::pref::runtime::options::FileExposurePreference>,

	// Whether file forwarding is allowed by the user
	file_forward:	&bool,
) -> Result<bool, zbus::Error> {
	if expose_list.len() == 0 {
		return Ok(true);
	};

	let (rw, ro, device) = crate::ipc::portals::permission_store::expose::get(
		app_id,
		dbus_conn,
		logger,
	).await?;

	if super::contain::permission_contained(&rw, &ro, &device, &expose_list) {
		Ok(true)
	} else {
		let (rw, ro, device) = {
			let mut rw = vec![];
			let mut ro = vec![];
			let mut dev = vec![];

			use crate::pref::runtime::options::FileExposurePreference;

			for expose in expose_list {
				match expose {
					FileExposurePreference::MountPath { host, dest: _, class }
					=> {
						match class {
							crate::bind::types::BindType::Device	=> {
								dev.push(host.to_path_buf());
							}
							crate::bind::types::BindType::ReadOnly	=> {
								ro.push(host.to_path_buf());
							}
							crate::bind::types::BindType::ReadWrite	=> {
								rw.push(host.to_path_buf());
							}
						}
					}
					FileExposurePreference::Passthrough { host: _ }		=> {
						continue;
					}
				}
			};

			(rw, ro, dev)
		};


		crate::ipc::portals::permission_store::expose::update(
			app_id,
			dbus_conn,
			Some(rw),
			Some(ro),
			Some(device),
		)
			.await
			?;
		Ok(false)
	}
}
