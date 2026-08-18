pub mod rules;
pub mod proxies;
pub mod documents;

/**
	Note that proxy_socket's parent must exist before starting the D-Bus proxy

	The public struct proxy is used to define rules and sandboxing layer for xdg-dbus-proxy.

	To influence the core sandbox binding logic, the app_sandbox field can be
	implemented. It can affect environment variables and filesystem bindings.
*/
#[derive(Debug)]
pub struct Proxy {
	pub sandbox:		crate::bind::types::BindRules,
	pub bus_access:		Vec<rules::BusAccessLevel>,
	/**
		Maps to $DBUS_SESSION_ADDRESS format
	*/
	pub bus_address:	String,
	pub logger:		crate::logger::LogSender,
	/**
		Designates the socket for proxy to listen on.

		WARNING: you should expose the parent directory manually
	*/
	pub proxy_socket:	std::path::PathBuf,

	/**
		Internally maps to --sloppy-names, makes all unique names visible
	*/
	pub sloppy_names:	bool,

	/**
		Specify a cancel token to call when the proxy finishes.
	*/
	pub cancen_token:	Option<tokio_util::sync::CancellationToken>,
	pub json_status_file:	Option<std::os::fd::OwnedFd>,

	/**
		Whether or not to introduce addition to the application sandboxing rules
	*/
	pub app_sandbox:	Option<crate::bind::types::BindRules>,

	/**
		Whether or not to introduce addition to the application environment variables
	*/
	pub envs:		Option<std::collections::HashMap<String, String>>
}

/**
	The shared trait StartProxy dictates a common generation method for D-Bus proxies.
*/
pub trait StartProxy: Sized {
	/// See struct Proxy for more info
	fn new(
		self
	) -> impl std::future::Future<Output = Result<Proxy, Self::ProxyError>> + Send;

	type ProxyError;
}
