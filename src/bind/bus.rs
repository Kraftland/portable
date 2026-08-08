pub mod rules;
pub mod proxies;
pub mod documents;

/**
	Note that proxy_path is in form of:
		unix:path=<proxy_path>/bus
	to workaround an issue with bind-mounting non-existent files

	The proxy directory must exist beforehand.

	The public struct proxy is used to define rules and sandboxing layer for xdg-dbus-proxy.

	To influence the core sandbox binding logic, the app_sandbox field can be
	implemented. It can affect environment variables and filesystem bindings.
*/
pub struct Proxy {
	pub sandbox:		crate::bind::types::BindRules,
	pub bus_access:		Vec<rules::BusAccessLevel>,
	pub bus_address:	String,
	pub logger:		crate::logger::LogSender,
	pub proxy_address:	String,

	/**
		Internally maps to --sloppy-names, makes all unique names visible
	*/
	pub sloppy_names:	bool,

	/**
		Whether to kill the sandbox once the proxy dies

		Specifies a stop token when needed
	*/
	pub bind_lifetime:	Option<tokio::sync::mpsc::Sender<crate::stop::StopLevel>>,
	pub json_status_file:	Option<std::os::fd::OwnedFd>,

	/**
		Whether or not to introduce addition to the application sandboxing rules
	*/
	pub app_sandbox:	Option<crate::bind::types::BindRules>,
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
