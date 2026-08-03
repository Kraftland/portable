pub mod rules;
pub mod proxies;
pub mod documents;

/**
	The public struct proxy is used to define rules and sandboxing layer for xdg-dbus-proxy
*/
pub struct Proxy {
	sandbox:	crate::bind::types::BindRules,
	bus_access:	Vec<rules::BusAccessLevel>,
	bus_address:	String,
	logger:		crate::logger::LogSender,
	proxy_address:	String,
}

#[derive(thiserror::Error, Debug)]
pub enum BusProxyError {

}

/**
	The shared trait StartProxy dictates a common start method for D-Bus proxies.
*/
pub trait StartProxy: Sized {
	fn start(
		proxy: Proxy,
	) -> impl std::future::Future<Output = Result<(), Self::ProxyError>> + Send;
	type ProxyError;

	/// See struct Proxy for more info
	/**
		Note that proxy_path internally translates to:
			unix:path=<proxy_path>/bus
		To workaround an issue with bind-mounting non-existent files
	*/
	fn new(
		logger:		crate::logger::LogSender,
		proxy_path:	std::path::PathBuf,

		#[cfg(feature = "flatpak")]
		info_path:	std::path::PathBuf,
		#[cfg(feature = "flatpak")]
		runtime_dir:	std::path::PathBuf,
	) -> impl std::future::Future<Output = Result<Self, Self::ProxyError>> + Send;
}
