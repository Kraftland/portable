pub async fn unbind(
	env:	&crate::envs::holder::HoldChannel,
) -> Result<(), super::DisplayBindError> {
	env.send(
		crate::envs::holder::EnvMessage::Add {
			key:	"WAYLAND_DISPLAY".into(),
			value:	"".into(),
		}
	)
		.await
		.map_err(super::DisplayBindError::SendEnvError)
		?;

	Ok(())
}
