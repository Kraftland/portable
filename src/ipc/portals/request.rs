

#[zbus::proxy(
	interface	= "org.freedesktop.portal.Request",
	default_service	= "org.freedesktop.portal.Desktop",
)]
trait Requests {
	#[zbus(
		signal,
		name	= "Response",
	)]
	async fn response(&self)	-> zbus::Result<(u32, zbus::zvariant::OwnedValue)>;

	async fn close(&self)		-> zbus::Result<()>;
}
