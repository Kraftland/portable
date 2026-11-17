/**
	The per-app instance of document portal

	The struct does not implement shared trait due to the async and IPC nature.

	It is later bound by the system subsystem, with inner_per_app acting as source
		and inner as dest.
*/
pub struct DocumentsMountPoint {
	inner:		std::path::PathBuf,
	inner_per_app:	std::path::PathBuf,
}

impl DocumentsMountPoint {
	pub async fn new(
			config:		std::sync::Arc<crate::config::Config>,
			bus:		zbus::Connection
		)	->
			Result<Self, DocumentError>
	{
		let shared_path = crate::ipc::portals::documents::get_mount_point(&bus)
			.await
			.map_err(DocumentError::DocumentGetError)
			?;

		let per_app_path = {
			let mut shared_path = shared_path.to_path_buf();
			shared_path.push("by-app");
			shared_path.push(&config.metadata.sandbox_id);
			shared_path
		};

		Ok(
			DocumentsMountPoint {
				inner:		shared_path,
				inner_per_app:	per_app_path,
			}
		)
	}

	pub fn path(&self)	-> std::path::PathBuf {
		self.inner.to_path_buf()
	}

	pub fn path_per_app(&self) -> std::path::PathBuf {
		self.inner_per_app.to_path_buf()
	}

	pub fn path_ref(&self)	-> &std::path::PathBuf {
		&self.inner
	}

	pub async fn create_path(&self) -> Result<(), DocumentError> {
		tokio::fs::create_dir_all(self.path_ref())
			.await
			.map_err(DocumentError::CreateError)
	}
}

#[derive(Debug, thiserror::Error)]
pub enum DocumentError {
	#[error("I/O error creating documents path: {0}")]
	CreateError(std::io::Error),

	#[error("Could not get Document Portal mount point: {0}")]
	DocumentGetError(crate::ipc::portals::documents::DocumentError),
}
