/**
	Read one key under a specific namespace
*/
pub async fn read_one(
	bus:		&zbus::Connection,
	namespace:	&str,
	key:		&str,
) -> zbus::Result<zbus::zvariant::OwnedValue> {
	let proxy = SettingsPortalProxy::new(&bus)
		.await
		?;

	proxy.read_one(namespace, key).await
}

#[zbus::proxy(
	interface	= "org.freedesktop.portal.Settings",
	default_service	= "org.freedesktop.portal.Desktop",
	default_path	= "/org/freedesktop/portal/desktop",
	gen_blocking	= false,
)]
trait SettingsPortal {
	async fn read_one(
		&self,
		namespace:	&str,
		key:		&str,
	) -> zbus::Result<zbus::zvariant::OwnedValue>;
}
