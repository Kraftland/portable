use crate::bind::types::BindRules;
use std::collections::HashMap;

/**
	Takes a vector of FileExposurePreference and returns the following:

	a vector of bind rules to represent type MountPath, and a hashmap of (orig, dest) for D-Bus
*/
pub async fn forward_file(
	expose_list:	Vec<crate::pref::runtime::options::FileExposurePreference>,

	dbus_conn:	&zbus::Connection,
	app_id:		&str,

	logger:		crate::logger::LogSender,
) -> (BindRules, HashMap<String, String>) {
	use crate::pref::runtime::options::FileExposurePreference;
	use crate::bind::types::BindRule;
	let mut rules = vec![];

	let mut path_list = vec![];

	for expose in expose_list {
		match expose {
			FileExposurePreference::MountPath { host, dest, class }	=> {
				rules.push(
					BindRule::Path {
						source:	host,
						dest:	dest,
						class:	class,
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
				let _ = logger.send(
					crate::logger::LogMessage {
						level: crate::logger::LogLevel::Warn,
						message: format!("Could not forward files: {e:#?}"),
					},
				).await;
				return (rules, pass_map);
			}
		};
		for (doc, host_path) in doc_ids.iter().zip(path_list) {
			pass_map.insert(
				host_path.to_string_lossy().to_string(),
				doc.to_string_lossy().to_string(),
			);
		};
	};

	(rules, pass_map)
}
