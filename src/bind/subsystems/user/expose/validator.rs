pub mod file_exposure;

/**
	Validates if permission allows current file exposure intents.

	Returns owned references of FileExposurePreference.
*/
pub async fn validate(
	dbus_conn:	&zbus::Connection,
	app_id:		&str,
	expose_list:	&Vec<crate::pref::runtime::options::FileExposurePreference>,

	// Whether file forwarding is allowed by the user
	file_forward:	&bool,
) -> Result<Vec<crate::pref::runtime::options::FileExposurePreference>, super::ExposeError> {
	let mut ret = vec![];

	let (
		// Raw PathBufs
		file_forwarding_list,
		expose_rw,
		expose_ro,
		expose_dev,
	) = {
		use crate::pref::runtime::options::FileExposurePreference;

		let mut ff = vec![];
		let mut expose_rw = vec![];
		let mut expose_ro = vec![];
		let mut expose_dev = vec![];

		for item in expose_list {
			match item {
				FileExposurePreference::MountPath {
					host,
					dest,
					class,
				}
				=> {
					match class {
						crate::bind::types::BindType::Device	=> {
							expose_dev.push(
								(
									host.to_path_buf(),
									dest.to_path_buf(),
								)
							);
						}
						crate::bind::types::BindType::ReadOnly	=> {
							expose_ro.push(
								(
									host.to_path_buf(),
									dest.to_path_buf(),
								)
							)
						}
						crate::bind::types::BindType::ReadWrite	=> {
							expose_rw.push(
								(
									host.to_path_buf(),
									dest.to_path_buf(),
								)
							);
						}
					}
				}
				FileExposurePreference::Passthrough { host: v }
				=> {
					ff.push(v);
				}
			}
		};

		(
			ff,

			expose_rw,
			expose_ro,
			expose_dev,
		)
	};

	/*
		Append file forwarding if allowed
	*/
	if *file_forward && file_forwarding_list.len() > 0 {
		use crate::pref::runtime::options::FileExposurePreference;
		for file in file_forwarding_list {
			ret.push(
				FileExposurePreference::Passthrough { host: file.to_path_buf() }
			);
		};
	};

	file_exposure::validate(
		dbus_conn,
		expose_rw,
		expose_ro,
		expose_dev,
		app_id,
		&mut ret,
	)
		.await
		?;

	Ok(ret)
}

