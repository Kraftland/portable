async fn get(conn: zbus::Connection) -> Result<std::path::PathBuf, DocumentsError> {
	let proxy = DocumentsProxy::new(&conn)
		.await
		.map_err(DocumentsError::ProxyError)
		?;
	let path = String::from_utf8(
		proxy
			.mount_point()
			.await
			.map_err(DocumentsError::GetMountError)
			?
	)
		.map_err(DocumentsError::InvalidUTF8Error)
		?;
	Ok(std::path::PathBuf::from(path))

}

#[derive(thiserror::Error, Debug)]
pub enum DocumentsError {
	#[error("Could not create Documents Portal proxy: {0:#?}")]
	ProxyError(zbus::Error),
	#[error("Could not get Documents Portal mount point: {0:#?}")]
	GetMountError(zbus::Error),
	#[error("Could not get Documents Portal: invalid UTF-8 charset: {0:#?}")]
	InvalidUTF8Error(std::string::FromUtf8Error),
}

use zbus::proxy;
#[proxy(
	default_service = "org.freedesktop.portal.Documents",
	default_path = "/org/freedesktop/portal/documents",
)]
trait Documents {
	#[zbus(name = "GetMountPoint")]
	async fn mount_point(&self) -> zbus::Result<Vec<u8>>;
}
