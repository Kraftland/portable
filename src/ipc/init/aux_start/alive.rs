
/**
	Wait if the Init is not alive yet
*/
pub async fn wait(
	conn:		&zbus::Connection,
	init_name:	&str,
) -> Result<(), super::AuxStartError> {
	let proxy = zbus::fdo::DBusProxy::new(conn)
		.await
		.map_err(super::AuxStartError::AliveError)
		?;

	let bus_name = zbus::names::BusName::try_from(init_name)
		.map_err(super::AuxStartError::InitNameError)
		?;

	let mut stream = proxy
		.receive_name_owner_changed_with_args(
			&vec![(0, init_name)]
		)
		.await
		.map_err(super::AuxStartError::AliveError)
		?;

	if proxy.name_has_owner(bus_name).await.map_err(super::AuxStartError::AliveFdoError)? {
		return Ok(());
	};

	{
		use futures_util::stream::StreamExt;
		while let Some(v) = stream.next().await {
			let args = v
				.args()
				.map_err(super::AuxStartError::AliveError)
				?;

			if args.new_owner.is_some() {
				break;
			} else {
				return Err(super::AuxStartError::RemoteDiedError);
			}
		}
	};

	Ok(())
}
