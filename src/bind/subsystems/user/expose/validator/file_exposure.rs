mod contains;

/**
	This async function validates if current permissions allow exposing user selected files,
	via --expose switches.

	Do not call this validate function if user did not select --expose, as it will have
	additional D-Bus IPC overhead.

	If permissions allow, append them into the append_dest vector.
*/
pub async fn validate(
	dbus_conn:	&zbus::Connection,
	rw:		Vec<MountPathPreference>,
	ro:		Vec<MountPathPreference>,
	device:		Vec<MountPathPreference>,
	app_id:		&str,

	append_dest:	&mut Vec<crate::pref::runtime::options::FileExposurePreference>,
) -> Result<(), crate::bind::subsystems::user::expose::ExposeError> {
	use crate::bind::subsystems::user::expose::ExposeError;


	let (
		rw_stored,
		ro_stored,
		device_stored,
	) = crate::ipc::portals::permission_store::expose::get(
		app_id,
		dbus_conn,
	)
		.await
		.map_err(ExposeError::PermissionStoreBusError)
		?;

	if contains::permission_contained(
		&rw,
		&ro,
		&device,
		&rw_stored,
		&ro_stored,
		&device_stored,
	) {
		for item in rw {
			append_dest.push(
				crate::pref::runtime::options::FileExposurePreference::MountPath {
					host:	item.0,
					dest:	item.1,
					class:	crate::bind::types::BindType::ReadWrite,
				}
			);
		};

		for item in ro {
			append_dest.push(
				crate::pref::runtime::options::FileExposurePreference::MountPath {
					host:	item.0,
					dest:	item.1,
					class:	crate::bind::types::BindType::ReadOnly,
				}
			);
		};

		for item in device {
			append_dest.push(
				crate::pref::runtime::options::FileExposurePreference::MountPath {
					host:	item.0,
					dest:	item.1,
					class:	crate::bind::types::BindType::Device,
				}
			);
		};
	};

	Ok(())

}

/**
	The MountPathPreference is an alias of (std::path::PathBuf, std::path::PathBuf).

	It is expressed as (source, destination)
*/
pub type MountPathPreference = (std::path::PathBuf, std::path::PathBuf);
