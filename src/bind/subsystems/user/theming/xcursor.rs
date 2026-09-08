/**
	Read the current cursor theme from Portal if able, and send the envs over to channel

	The specific way of choice is querying the Settings Portal for "cursor-theme" under
		namespace "org.gnome.desktop.interface".
*/
pub async fn forward_xcursor(
	env:		crate::envs::holder::HoldChannel,
	logger:		crate::logger::LogSender,
	bus:		zbus::Connection,
) -> Result<(), crate::bind::subsystems::user::UserBindError> {
	let cursor_name = {
		let raw_value = match crate::ipc::portals::settings::read_one(
			&bus,
			"org.gnome.desktop.interface",
			"cursor-theme",
		).await {
			Ok(v)	=> v,
			Err(e)	=> {
				let _ = logger.send(
					crate::logger::LogMessage {
						level:		crate::logger::LogLevel::Warn,
						message:	format!("Could not retrieve cursor theme from Portal: {e:#?}"),
					}
				).await;
				return Ok(());
			}
		};

		match String::try_from(raw_value) {
			Ok(v)	=> v,
			Err(e)	=> {
				return Err(
					crate::bind::subsystems::user::UserBindError::CursorVariantStringError(e)
				);
			}
		}
	};

	env.send(
		crate::envs::holder::EnvMessage::Add {
			key:	"XCURSOR_THEME".into(),
			value:	cursor_name,
		}
	)
		.await
		.map_err(crate::bind::subsystems::user::UserBindError::ForwardEnvsError)
		?;

	Ok(())
}


