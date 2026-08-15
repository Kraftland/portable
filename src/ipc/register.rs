/**
	RegisterStatus dictates which mode to operate on

	Primary means the sole instance to start and configure sandbox
	Secondary relies on another primary instance to execute a process
*/
pub enum RegisterStatus {
	Primary,
	Secondary,
}

#[derive(thiserror::Error, Debug)]
pub enum RegisterError {
	#[error("Could not build a D-Bus connection: {0:#?}")]
	BuildConnectionError(zbus::Error),

	#[error("Could not serve a D-Bus Object Server: {0:#?}")]
	ServeObjectError(zbus::Error),

	#[error("Could not request well-known name on D-Bus: {0:#?}")]
	RequestNameError(zbus::Error),
}

/**
	Connect to the session bus and publish services

	Must be done before register to handle multi-instance and certain commamdline flags
*/
pub async fn connect(stop_tx: tokio::sync::mpsc::Sender<crate::stop::StopLevel>)
-> Result<zbus::Connection, RegisterError> {
	let builder = zbus::connection::Builder::session()
		.map_err(RegisterError::BuildConnectionError)
		?;
	let builder = builder
		.allow_name_replacements(false);
	let builder = builder
		.replace_existing_names(false);
	let builder = builder
		.serve_at("/top/kimiblock/portable/daemon", super::objects::Info)
		.map_err(RegisterError::ServeObjectError)
		?;
	let builder = builder
		.serve_at(
			"/top/kimiblock/portable/daemon",
			super::objects::Controller {stop_tx: stop_tx},
		)
		.map_err(RegisterError::ServeObjectError)
		?;
	builder
		.build()
		.await
		.map_err(RegisterError::BuildConnectionError)
}


/**
	Register as the primary instance on the session bus, if possible

	Otherwise returns the secondary status
*/
pub async fn register(
	app_id:		String,
	conn:		zbus::Connection,
) -> Result<RegisterStatus, RegisterError> {

	let mut name = String::from("top.kimiblock.portable.");
	name.push_str(&app_id);

	match conn.request_name_with_flags(name, zbus::fdo::RequestNameFlags::DoNotQueue.into()).await {
		Ok(_)	=> {
			Ok(RegisterStatus::Primary)
		}
		Err(e)	=> {
			match e {
				zbus::Error::NameTaken	=> {
					Ok(RegisterStatus::Secondary)
				}
				_			=> {
					Err(RegisterError::RequestNameError(e))
				}
			}
		}
	}
}
