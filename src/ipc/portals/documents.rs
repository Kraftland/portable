/**
	List the document storage of a specific application

	Returns a hash map of document IDs and PathBuf, paired in order.
*/
pub async fn list(
	dbus_conn:	&zbus::Connection,
	app_id:		&str,
) -> Result<std::collections::HashMap<String, std::path::PathBuf>, DocumentError> {
	let proxy = DocumentsPortalProxy::new(&dbus_conn)
		.await
		.map_err(DocumentError::ProxyError)
		?;

	let documents = proxy
		.list(app_id)
		.await
		.map_err(DocumentError::ListError)
		?;

	let mut map = std::collections::HashMap::with_capacity(documents.len());

	for (k, v) in documents {
		let path_string = String::try_from(v)
			.map_err(DocumentError::PathNotUTF8)
			?;
		map.insert(k, std::path::PathBuf::from(path_string.trim_end_matches("\0")));
	};

	Ok(map)
}

/**
	Delete files in the document Store.

	The file itself is not deleted.
*/
pub async fn delete(
	dbus_conn:	&zbus::Connection,
	doc_ids:	Vec<String>,
) -> Result<(), DocumentError> {
	let proxy = DocumentsPortalProxy::new(&dbus_conn)
		.await
		.map_err(DocumentError::ProxyError)
		?;

	for file in doc_ids {
		proxy
			.delete(&file)
			.await
			.map_err(DocumentError::DeleteError)
			?;
	};

	Ok(())
}

/**
	Adds every file into the document storage, returns a series of dest paths

	Caller can use .zip to iterate and make a connection between original paths and new paths
*/
pub async fn add_full(
	paths:		&Vec<&std::path::PathBuf>,
	dbus_conn:	&zbus::Connection,
	app_id:		&str,
) -> Result<Vec<std::path::PathBuf>, DocumentError> {
	let mut fds = vec![];

	for path in paths {
		let is_file = path.is_file();
		let is_dir = path.is_dir();

		if ! is_file && ! is_dir {
			continue;
		}

		let file = tokio::fs::OpenOptions::new()
			.read(true)
			.write(is_dir)
			.mode(0o700)
			.open(path)
			.await
			.map_err(DocumentError::OpenError)
			?;
		fds.push(
			zbus::zvariant::OwnedFd::from(
					std::os::fd::OwnedFd::from(file.into_std().await)
				),
		);
	};

	let proxy = DocumentsPortalProxy::new(&dbus_conn)
		.await
		.map_err(DocumentError::ProxyError)
		?;

	let permissions = vec![
		"read".to_string(),
		"write".to_string(),
		"grant-permissions".to_string(),
	];

	let (doc_ids, extra_info) = proxy
		.add_full(
			fds,
			1,
			&app_id,
			permissions,
		)
		.await
		.map_err(DocumentError::CallError)
		?;

	let prefix = {
		let raw_val = match extra_info.get("mountpoint") {
			Some(v)	=> {v}
			None	=> {
				return Err(DocumentError::MissingMountError);
			}
		};
		let bytes: Vec<u8> = raw_val
			.to_owned()
			.try_into()
			.map_err(DocumentError::MountTypeError)
			?;

		let path = String::from_utf8(bytes)
			.map_err(DocumentError::MountNotUTF8)
			?;
		path.trim_end_matches('\0').to_string()
	};

	let mut ret = vec![];

	for (doc, original) in doc_ids.iter().zip(paths) {
		let mut path = std::path::PathBuf::from(&prefix);
		path.push(doc.trim_end_matches('\0'));

		{
			match original.file_name() {
				Some(v)	=> {
					path.push(v);
				}
				None	=> {}
			};
		};

		ret.push(path);
	};

	Ok(ret)

}

#[derive(thiserror::Error, Debug)]
pub enum DocumentError {
	#[error("Could not open file: {0:#?}")]
	OpenError(std::io::Error),

	#[error("Could not create bus proxy: {0:#?}")]
	ProxyError(zbus::Error),

	#[error("Could not request passthrough: {0:#?}")]
	CallError(zbus::Error),

	#[error("Missing mountpoint in Portal reply, you may need a system upgrade")]
	MissingMountError,

	#[error("Type of mountpoint is not [u8]: {0:#?}")]
	MountTypeError(zbus::zvariant::Error),

	#[error("Type of mountpoint is not UTF-8: {0:#?}")]
	MountNotUTF8(std::string::FromUtf8Error),

	#[error("List error: {0:#?}")]
	ListError(zbus::Error),

	#[error("Path is not UTF-8: {0:#?}")]
	PathNotUTF8(std::string::FromUtf8Error),

	#[error("Error deleting a file from document storage: {0:#?}")]
	DeleteError(zbus::Error),
}

#[zbus::proxy(
	interface	= "org.freedesktop.portal.Documents",
	default_service	= "org.freedesktop.portal.Documents",
	default_path	= "/org/freedesktop/portal/documents",
)]
trait DocumentsPortal {
	#[zbus(name = "AddFull")]
	async fn add_full(
		&self,
		fds:		Vec<zbus::zvariant::OwnedFd>,
		flags:		u32, // 1 == reuse_existing, 2 == persistent, 4 == as-needed-by-app, 8 = export-directory
		app_id:		&str,
		permissions:	Vec<String>,
	) -> zbus::Result<(Vec<String>, std::collections::HashMap<String, zbus::zvariant::OwnedValue>)>;

	#[zbus(name = "List")]
	async fn list(
		&self,
		app_id:		&str,
	) -> zbus::Result<std::collections::HashMap<String, Vec<u8>>>;

	#[zbus(name = "Delete")]
	async fn delete(
		&self,
		doc_id:		&str,
	) -> zbus::Result<()>;
}
