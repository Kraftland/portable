/**
	The per-app instance of document portal

	The struct does not implement shared trait due to the async and IPC nature.
*/
pub struct DocumentsMountPoint {
	inner:	std::path::PathBuf,
}

impl DocumentsMountPoint {
	async fn new(
			config:		&crate::config::Config,
			bus:		&zbus::Connection
		)	->
			Result<Self, DocumentError>
	{
		let proxy = DocumentsProxy::new(bus)
			.await
			.map_err(DocumentError::ProxyError)
			?;

		Ok(
			DocumentsMountPoint {
				inner: {
					let bytes = proxy
						.mount_point()
						.await
						.map_err(DocumentError::CallError)
						?;
					let path = String::from_utf8(bytes)
						.map_err(DocumentError::TranslateError)
						?;
					let mut pathbuf = std::path::PathBuf::from(path);
					pathbuf.push(&config.metadata.sandbox_id);
					pathbuf
				},
			}
		)
	}

	fn path(&self)	-> std::path::PathBuf {
		self.inner.clone()
	}

	fn path_ref(&self) -> &std::path::PathBuf {
		&self.inner
	}

	async fn create_path(&self) -> Result<(), DocumentError> {
		tokio::fs::create_dir_all(self.path_ref())
			.await
			.map_err(DocumentError::CreateError)
	}
}

#[zbus::proxy(
	interface	= "org.freedesktop.portal.Documents",
	default_service	= "org.freedesktop.portal.Documents",
	default_path	= "/org/freedesktop/portal/documents",
)]
trait Documents {
	#[zbus(
		name	= "GetMountPoint"
	)]
	fn mount_point(&self) -> zbus::Result<Vec<u8>>;
}

#[derive(Debug, thiserror::Error)]
pub enum DocumentError {
	#[error("I/O error creating documents path: {0:#?}")]
	CreateError(std::io::Error),

	#[error("Error creating proxy: {0:#?}")]
	ProxyError(zbus::Error),

	#[error("Error calling proxy: {0:#?}")]
	CallError(zbus::Error),

	#[error("Error translating bytes to String: {0:#?}")]
	TranslateError(std::string::FromUtf8Error),
}
