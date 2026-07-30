/*
	RegisterStatus dictates which mode to operate on

	Primary means the sole instance to start and configure sandbox
	Secondary relies on another primary instance to execute a process
*/
pub enum RegisterStatus {
	Primary {connection: zbus::Connection},
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

pub async fn register(
	app_id:		&str,
	stop_tx:	tokio::sync::mpsc::Sender<crate::stop::StopLevel>,
) -> Result<RegisterStatus, RegisterError> {
	let builder = zbus::connection::Builder::
		session()
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
			super::objects::AuxStart {
				started: std::sync::atomic::AtomicBool::new(false)
			}
		)
		.map_err(RegisterError::ServeObjectError)
		?;
	let builder = builder
		.serve_at(
			"/top/kimiblock/portable/daemon",
			super::objects::Controller {stop_tx: stop_tx},
		)
		.map_err(RegisterError::ServeObjectError)
		?;
	let conn = builder
		.build()
		.await
		.map_err(RegisterError::BuildConnectionError)
		?;

	let mut name = String::from("top.kimiblock.portable.");
	name.push_str(&app_id);

	match conn.request_name(name).await {
		Ok(_)	=> {
			Ok(RegisterStatus::Primary { connection: conn })
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
