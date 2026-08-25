/**
	List registered StatusNotifierItems' bus name and their path
*/
pub async fn list(conn: &zbus::Connection, app_id: &str, log: &crate::logger::LogSender)
-> zbus::fdo::Result<Vec<(String, String)>> {
	let mut ret = vec![];

	let proxy = StatusNotifierWatcherProxy::new(&conn)
		.await
		?;

	let proxy_fdo = zbus::fdo::DBusProxy::new(&conn)
		.await
		?;

	let names = proxy
		.get_registered()
		.await
		?;

	for name in names {
		match name.split_once("@") {
			Some((k, v))	=> {

				let bus_name = match zbus::names::BusName::try_from(k) {
					Ok(v)	=> v,
					Err(e)	=> {
						return Err(zbus::fdo::Error::InvalidArgs(format!("{e:#?}")));
					}
				};
				let creds = proxy_fdo
					.get_connection_credentials(bus_name)
					.await
					?;
				let pid = match creds.process_id() {
					Some(v)	=> v,
					None	=> {
						return Err(
							zbus::fdo::Error::InvalidArgs(
								format!("No PID for {k}"),
							),
						);
					}
				};

				let dir = {
					let mut name = String::from("top.kimiblock.portable.");
					name.push_str(&app_id);
					let mut path = std::path::PathBuf::from("/proc");
					path.push(&pid.to_string());
					path.push("root");
					path.push(&name);
					path
				};

				#[cfg(debug_assertions)]
				let _ = log.send(
					crate::logger::LogMessage {
						level:		crate::logger::LogLevel::Debug,
						message:	format!("Got D-Bus remote: {k} on {pid:?}"),
					}
				).await;

				if ! dir.exists() {
					#[cfg(debug_assertions)]
					let _ = log.send(
						crate::logger::LogMessage {
							level:		crate::logger::LogLevel::Debug,
							message:	format!(
								"D-Bus remote: {k} on {pid:?} does not match",
							),
						}
					).await;
					continue;
				}


				ret.push((k.to_string(), v.to_string()));
			}
			None		=> {
				// Legacy style
				ret.push(
					(
						name,
						"/StatusNotifierItem".to_string(),
					)
				);
			}
		};
	};

	Ok(ret)
}


#[zbus::proxy(
	default_service	= "org.kde.StatusNotifierWatcher",
	default_path	= "/StatusNotifierWatcher",
	interface	= "org.kde.StatusNotifierWatcher"
)]
trait StatusNotifierWatcher {
	#[zbus(
		name	= "RegisteredStatusNotifierItems",
		property,
	)]
	fn get_registered(&self) -> zbus::Result<Vec<String>>;
}


