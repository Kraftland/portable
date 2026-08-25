pub async fn wake_name(conn: &zbus::Connection, name: &str, path: &str) -> zbus::Result<()> {
	let proxy = StatusNotifierItemProxy::new(conn, name, path)
		.await
		?;
	proxy
		.activate(1, 18)
		.await
}

#[zbus::proxy(
	interface	= "org.kde.StatusNotifierItem",
)]
trait StatusNotifierItem {
	#[zbus(name = "Activate")]
	async fn activate(&self, x: i32, y: i32) -> zbus::Result<()>;
}
