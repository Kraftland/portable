/**
	Reset permission for an app

	Currently it only removes file permission.
*/
pub async fn reset(
	app_id:	&str,
	bus:	&zbus::Connection,
) -> Result<(), ResetError> {
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

	Ok(())
}

#[derive(Debug, thiserror::Error)]
pub enum ResetError {
	#[error("Error resetting documents permission: {0:#?}")]
	DocumentError(crate::ipc::portals::documents::DocumentError),
}
