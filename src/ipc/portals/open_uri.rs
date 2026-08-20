

#[zbus::proxy(
	interface	= "org.freedesktop.portal.OpenURI",
	default_service	= "org.freedesktop.portal.Desktop",
	default_path	= "/org/freedesktop/portal/desktop",
)]
trait OpenURI {
	#[zbus(name = "OpenDirectory")]
	async fn directory(
		&self,
		parent_window:	String,
		fd:		zbus::zvariant::OwnedFd,
		options:	std::collections::HashMap<String, zbus::zvariant::OwnedValue>,
	) -> zbus::Result<zbus::zvariant::OwnedObjectPath>;
}
