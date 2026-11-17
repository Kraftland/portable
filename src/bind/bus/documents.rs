/**
	The public type describes a path of Documents Portal

*/
pub type DocumentPath = std::path::PathBuf;

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
