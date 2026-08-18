
/**
	Wait for the unit to exit with D-Bus
*/
pub async fn wait(
	conn:		&zbus::Connection,
	escaped_name:	super::escape::UnitName,
	logger:		crate::logger::LogSender,
	cancel_token:	tokio_util::sync::CancellationToken,
) -> Result<(), zbus::Error> {
	let proxy = ManagerProxyProxy::new(&conn)
		.await
		?;

	tokio::spawn(
		async move {
			while let Ok(message) = proxy.receive_job_removed_with_args(
				&vec![(2, escaped_name.as_str())]
			).await {
				break;
			}
			cancel_token.cancel();
		}
	);

	Ok(())
}

#[zbus::proxy(
	default_service	= "org.freedesktop.systemd1",
	default_path	= "/org/freedesktop/systemd1",
	interface	= "org.freedesktop.systemd1.Manager"
)]
trait ManagerProxy {
	#[zbus(signal)]
	async fn job_removed(
		&self,
		id:	u32,
		job:	zbus::zvariant::OwnedObjectPath,
		unit:	String,
		result:	String,
	);
}
