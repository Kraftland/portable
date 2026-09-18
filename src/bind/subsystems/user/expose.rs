use crate::bind::types::BindRules;
use std::collections::HashMap;
mod contain;

/**
	Validates if the permission can be restored without calling user consent.

	It does not append new permissions, but just overwrites them in place.

	Also returns Ok(true) when the list is empty

	If not then update the permission store and return false.
*/
async fn validate_stored_permission(
	dbus_conn:	&zbus::Connection,
	logger:		&crate::logger::LogSender,
	app_id:		&str,
	expose_list:	&Vec<crate::pref::runtime::options::FileExposurePreference>,
) -> Result<bool, zbus::Error> {
	if expose_list.len() == 0 {
		return Ok(true);
	};

	let (rw, ro, device) = crate::ipc::portals::permission_store::expose::get(
		app_id,
		dbus_conn,
		logger,
	).await?;

	if contain::permission_contained(&rw, &ro, &device, &expose_list) {
		Ok(true)
	} else {
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

/**
	Takes a vector of FileExposurePreference and returns the following:

	a vector of bind rules to represent type MountPath, and a hashmap of (orig, dest) for D-Bus
*/
pub async fn forward_file(
	expose_list:	&Vec<crate::pref::runtime::options::FileExposurePreference>,
	dbus_conn:	&zbus::Connection,
	app_id:		&str,
	logger:		crate::logger::LogSender,
) -> (BindRules, HashMap<String, String>) {
	match validate_stored_permission(dbus_conn, &logger, app_id, expose_list).await {
		Ok(true)	=> {}
		Ok(false)	=> {
			match question(&expose_list).await {
				Ok(true)	=> {}
				Ok(false)	=> {
					let _ = logger.send(
						crate::logger::LogMessage {
							level: crate::logger::LogLevel::Info,
							message: format!("User denied exposing files"),
						},
					).await;

					return (vec![], std::collections::HashMap::new());
				}
				Err(e)		=> {
					let _ = logger.send(
						crate::logger::LogMessage {
							level: crate::logger::LogLevel::Warn,
							message: format!("Could not ask for consent: {e:#?}"),
						},
					).await;
					return (vec![], std::collections::HashMap::new());
				}
			}
		}
		Err(e)		=> {
			let _ = logger.send(
				crate::logger::LogMessage {
					level:		crate::logger::LogLevel::Warn,
					message:	format!(
						"Could not retrieve stored permissions: {e:#?}",
					),
				}
			).await;
			match question(&expose_list).await {
				Ok(true)	=> {}
				Ok(false)	=> {
					let _ = logger.send(
						crate::logger::LogMessage {
							level: crate::logger::LogLevel::Info,
							message: format!("User denied exposing files"),
						},
					).await;

					return (vec![], std::collections::HashMap::new());
				}
				Err(e)		=> {
					let _ = logger.send(
						crate::logger::LogMessage {
							level: crate::logger::LogLevel::Warn,
							message: format!("Could not ask for consent: {e:#?}"),
						},
					).await;
					return (vec![], std::collections::HashMap::new());
				}
			}
		}
	}


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
		#[cfg(debug_assertions)]
		let _ = logger.send(
			crate::logger::LogMessage {
				level:		crate::logger::LogLevel::Debug,
				message:	format!("Forwarding file via Portals: {:?}", &path_list),
			}
		).await;

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

/**
	Ask the user for consent of exposing files. Currently uses Zenity but we want to natively
		do that in the future.
*/
async fn question(paths: &Vec<crate::pref::runtime::options::FileExposurePreference>)
-> Result<bool, super::UserBindError> {
	let mut cmd_args = vec![
		"--title",
		"Permission Control",
		"--icon=folder-open-symbolic",
		"--question",
		"--default-cancel",
	];

	let text = {
		use crate::pref::runtime::options::FileExposurePreference;
		let mut string = String::new();
		string.push_str("--text=Exposing the following path: \n");
		for path in paths {
			match path {
				FileExposurePreference::Passthrough { host }	=> {
					string.push_str(&host.to_string_lossy());
				}
				FileExposurePreference::MountPath { host, dest: _, class: _ }
										=> {
					string.push_str(&host.to_string_lossy());
				}
			}
		};
		string
	};

	cmd_args.push(&text);

	let mut command = tokio::process::Command::new("zenity");

	let mut command = command.kill_on_drop(true);
	command = command.args(cmd_args);

	let mut child = match command.spawn() {
		Ok(v)	=> v,
		Err(e)	=> {
			return Err(
				super::UserBindError::ZenitySpawnError(e)
			);
		}
	};

	if child.wait().await.map_err(super::UserBindError::ZenitySpawnError)?.success() {
		Ok(true)
	} else {
		Ok(false)
	}
}
