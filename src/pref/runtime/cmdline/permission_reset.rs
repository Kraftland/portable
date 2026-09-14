/**
	Reset permission for an app

	Currently it only removes file permission.
*/
pub async fn reset(
	app_id:	std::sync::Arc<String>,
	bus:	&zbus::Connection,
) -> Result<(), ResetError> {

	let reset = {
		let bus = bus.clone();
		let id = app_id.clone();
		tokio::spawn(
			crate::ipc::portals::permission_store::reset_permissions(
				bus,
				id,
			)
		)
	};

	{
		use crate::ipc::portals::documents;

		let list = documents::list(&bus, &app_id)
			.await
			.map_err(ResetError::DocumentError)
			?;

		let mut doc_ids = vec![];

		for (k, _v) in list {
			doc_ids.push(k);
		};

		documents::delete(&bus, doc_ids)
			.await
			.map_err(ResetError::DocumentError)
			?;
	};

	reset
		.await
		.map_err(ResetError::SpawnError)
		?
		.map_err(ResetError::BusIPCError)
		?;

	Ok(())
}

#[derive(Debug, thiserror::Error)]
pub enum ResetError {
	#[error("Error resetting documents permission: {0:#?}")]
	DocumentError(crate::ipc::portals::documents::DocumentError),

	#[error("Error spawning task: {0:#?}")]
	SpawnError(tokio::task::JoinError),

	#[error("Error doing IPC: {0:#?}")]
	BusIPCError(zbus::Error),
}
