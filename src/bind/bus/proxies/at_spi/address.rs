/**
	Get the accessibility bus's address, errors if not accessible

	Errors out if not UNIX socket to provide isolation. As network filtering does not work.
*/
pub async fn get_address(conn: zbus::Connection) -> Result<String, super::AtspiError> {
	let proxy = GetAddressProxy::new(&conn)
		.await
		.map_err(super::AtspiError::AddressError)
		?;

	let address = proxy.get()
		.await
		.map_err(super::AtspiError::AddressError)
		?;

	if address.starts_with("unix:path=") {
		Ok(address)
	} else {
		Err(super::AtspiError::NotSocketError)
	}
}

#[zbus::proxy(
	interface = "org.a11y.Bus",
	default_service = "org.a11y.Bus",
	default_path = "/org/a11y/bus",
	gen_async = true,
)]
trait GetAddress {
	async fn get(&self) -> zbus::Result<String>;
}
