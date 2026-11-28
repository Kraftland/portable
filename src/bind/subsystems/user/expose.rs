use crate::bind::types::BindRules;
use std::collections::HashMap;
mod validator;

#[derive(Debug, thiserror::Error)]
pub enum ExposeError {
	#[error("Could not contact PermissionStore: {0}")]
	PermissionStoreBusError(zbus::Error),

	#[error("Could not spawn zenity for user consent check: {0}")]
	ExposeZenityError(validator::file_exposure::question::QuestionError),
}

/**
	Takes a vector of FileExposurePreference and returns the following:

	a vector of bind rules to represent type MountPath, and a hashmap of (orig, dest) for D-Bus
*/
pub async fn forward_file(
	expose_list:	&Vec<crate::pref::runtime::options::FileExposurePreference>,
	dbus_conn:	&zbus::Connection,
	app_id:		&str,
	logger:		crate::logger::LogSender,

	// Whether file forwarding is allowed by the user
	file_forward:	&bool,
) -> Result<(BindRules, HashMap<String, String>), ExposeError> {
	let expose_list = validator::validate(
		dbus_conn,
		app_id,
		expose_list,
		file_forward,
	)
		.await
		?;


	use crate::pref::runtime::options::FileExposurePreference;
	use crate::bind::types::BindRule;
	let mut rules = vec![];

	let mut path_list = vec![];

	use crate::bind::types::BindType;

	for expose in expose_list {
		match expose {
			FileExposurePreference::MountPath { host, dest, class }	=> {
				rules.push(
					BindRule::Path {
						source:	host.to_path_buf(),
						dest:	dest.to_path_buf(),
						class:	{
							match class {
								BindType::Device	=> {
									BindType::Device
								}
								BindType::ReadOnly	=> {
									BindType::ReadOnly
								}
								BindType::ReadWrite	=> {
									BindType::ReadWrite
								}
							}
						},
					}
				);
			}
			FileExposurePreference::Passthrough { host }		=> {
				path_list.push(host);
			}
		}
	};

	let mut pass_map = HashMap::new();

	if path_list.len() > 0 {

		let doc_ids = crate::ipc::portals::documents::add_full(
			&path_list,
			dbus_conn,
			&app_id,
		)
			.await;
		let doc_ids = match doc_ids {
			Ok(v)	=> v,
			Err(e)	=> {
				use crate::logger::messages::warn::WarnMessage;
				let _ = logger.send(
					crate::logger::messages::LogMessage::Warn(
						WarnMessage::ForwardAddFullError(e)
					)
				).await;
				return Ok((rules, pass_map));
			}
		};
		for (doc, host_path) in doc_ids.iter().zip(path_list) {
			pass_map.insert(
				host_path.to_string_lossy().to_string(),
				doc.to_string_lossy().to_string(),
			);
		};
	};

	Ok((rules, pass_map))
}
