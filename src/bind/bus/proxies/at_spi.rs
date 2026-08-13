/**
	The accessibility bus proxy, errors should not be critical
*/
pub struct AtspiProxy {
	pub zbus_connection:	zbus::Connection,
	pub portable_runtime:	std::sync::Arc<crate::bind::subsystems::dirs::portable_runtime::PortableRuntime>,
	pub logger:		crate::logger::LogSender,

	#[cfg(feature = "flatpak")]
	pub flatpak_info:	std::path::PathBuf,
}

mod rules;
mod address;
mod sandbox;

impl crate::bind::bus::StartProxy for AtspiProxy {
	async fn new(
			self
		) -> Result<crate::bind::bus::Proxy, Self::ProxyError>
	{
		let bus_rules = tokio::spawn(rules::generate_rules());

		let host_address = address::get_address(self.zbus_connection)
			.await
			?;

		let host_address_formatted = {
			let mut name = String::from("unix:path=");
			name.push_str(&host_address.as_os_str().to_string_lossy());
			name
		};

		let proxy_socket_path = {
			use crate::bind::subsystems::dirs::RuntimePathsTrait;
			let mut path = self.portable_runtime.path();
			path.push("at-spi");
			path
		};

		let proxy_address = {
			let mut name = String::from("unix:path=");
			name.push_str(&proxy_socket_path.as_os_str().to_string_lossy());
			name
		};

		let sandbox_rules = sandbox::get_sandbox(
			&proxy_socket_path,
			host_address,
			self.flatpak_info,
		)
			.await
			?;

		Ok(
			crate::bind::bus::Proxy {
				sandbox:		sandbox_rules,
				bus_access:		bus_rules.await.map_err(AtspiError::SpawnError)??,
				bus_address:		host_address_formatted,
				logger:			self.logger,
				proxy_address:		proxy_address,
				sloppy_names:		true,
				bind_lifetime:		None,
				json_status_file:	None,
				app_sandbox:		Some(sandbox::app(&proxy_socket_path).await),
				envs:			None,
			}
		)
	}

	type ProxyError = AtspiError;
}

#[derive(Debug, thiserror::Error)]
pub enum AtspiError {
	#[error("Could not start D-Bus proxy for a11y bus: invalid bus name: {0:#?}")]
	InvalidBusNameError(crate::bind::bus::rules::BusNameError),

	#[error("Could not start D-Bus proxy for a11y bus: error obtaining address: {0:#?}")]
	AddressError(zbus::Error),

	#[error("Could not start D-Bus proxy for a11y bus: address not socket")]
	NotSocketError,

	#[error("Could not start D-Bus proxy for a11y bus: thread spawn error: {0:#?}")]
	SpawnError(tokio::task::JoinError),
}

