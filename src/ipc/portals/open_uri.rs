/**
	Opens a directory.
*/
pub async fn open_directory(
	bus:		&zbus::Connection,
	parent_window:	Option<&str>,
	path:		std::path::PathBuf,
) -> zbus::Result<()> {
	let token = super::request::generate_token().await;
	let sender = match bus.unique_name() {
		Some(v)	=> {v}
		None	=> {
			return Err(
				zbus::Error::Failure("Missing a bus unique name".to_string())
			);
		}
	};
	let fd = tokio::fs::OpenOptions::new()
		.read(true)
		.write(false)
		.create(false)
		.open(path)
		.await;

	let fd = match fd {
		Ok(v)	=> v,
		Err(e)	=> {
			return Err(
				zbus::Error::Failure(format!("Could not open path: {e:#?}"))
			);
		}
	};

	let response = crate::ipc::portals::request::get_response(&bus, sender, &token)
		.await
		?;

	let proxy = OpenURIProxy::new(&bus)
		.await
		?;

	use std::collections::HashMap;

	let mut options: HashMap<String, zbus::zvariant::OwnedValue> = HashMap::new();

	{
		use zbus::zvariant::Str;
		use zbus::zvariant::OwnedValue;

		options.insert(
			"handle_token".to_string(),
			OwnedValue::from(
				Str::from(token)
			)
		);
	};

	let fd = {
		use std::os::fd::OwnedFd;

		OwnedFd::from(
			fd
				.into_std()
				.await
		)

	};



	proxy
		.directory(
			parent_window.unwrap_or("parent_window"),
			zbus::zvariant::OwnedFd::from(fd),
			options,
		)
		.await
		?;

	let result = match response.await {
		Ok(v)	=> v?,
		Err(e)	=> {
			return Err(
				zbus::Error::Failure(format!("{e:#?}"))
			);
		}
	};

	use crate::ipc::portals::request::PortalResponse;
	match result.0 {
		PortalResponse::Success			=> {
			Ok(())
		}
		PortalResponse::Cancelled		=> {
			Err(
				zbus::Error::Failure(String::from("Request denied"))
			)
		}
		PortalResponse::Failed			=> {
			Err(
				zbus::Error::Failure(String::from("Request failed"))
			)
		}
		PortalResponse::Unknown { code }	=> {
			Err(
				zbus::Error::Failure(format!("Unknown response: {code:?}"))
			)
		}
	}
}

#[zbus::proxy(
	interface	= "org.freedesktop.portal.OpenURI",
	default_service	= "org.freedesktop.portal.Desktop",
	default_path	= "/org/freedesktop/portal/desktop",
)]
trait OpenURI {
	#[zbus(name = "OpenDirectory")]
	async fn directory(
		&self,
		parent_window:	&str,
		fd:		zbus::zvariant::OwnedFd,
		options:	std::collections::HashMap<String, zbus::zvariant::OwnedValue>,
	) -> zbus::Result<zbus::zvariant::OwnedObjectPath>;
}
