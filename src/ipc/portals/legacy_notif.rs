/**
	Notifies the user with org.freedesktop.Notifications
*/
pub async fn notify(
	dbus_conn:	&zbus::Connection,
	icon:		&str,
	summary:	&str,
	body:		&str,
) -> Result<(), zbus::Error> {
	let proxy = NotifyProxy::new(dbus_conn)
		.await
		?;

	proxy.send(
		"Portable Daemon".to_string(),
		0,
		icon.into(),
		summary.into(),
		body.into(),
		vec![],
		std::collections::HashMap::new(),
		7,
	).await?;
	Ok(())
}

#[zbus::proxy(
	interface = "org.freedesktop.Notifications",
	default_service = "org.freedesktop.Notifications",
	default_path = "/org/freedesktop/Notifications",
)]
trait Notify {
	#[zbus(name = "Notify")]
	async fn send(
		&self,
		app_name:	String,
		// optional ID of an existing notification this notification is intended to replace
		replace_id:	u32,
		icon:		String,
		// a single line overview of the notification
		summary:	String,
		// a multi-line body of text
		body:		String,
		// actions send a request message back to the notification client when invoked
		actions:	Vec<String>,
		// usually empty, server may not support it
		hints:		std::collections::HashMap<String, zbus::zvariant::OwnedValue>,
		timeout:	i32,
	) -> zbus::Result<u32>;
}
