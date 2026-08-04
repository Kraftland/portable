pub mod rules;
pub mod proxies;
pub mod documents;

/**
	The public struct proxy is used to define rules and sandboxing layer for xdg-dbus-proxy
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
}
/**
	The shared trait StartProxy dictates a common start method for D-Bus proxies.
*/
pub trait StartProxy: Sized {
	/// See struct Proxy for more info
	/**
		Note that proxy_path internally translates to:
			unix:path=<proxy_path>/bus
		To workaround an issue with bind-mounting non-existent files

		The proxy directory must exist beforehand.
	*/
	fn new(
		logger:		crate::logger::LogSender,
		proxy_path:	std::path::PathBuf,
		mpris_names:	Vec<String>,
		stop_token:	Option<tokio::sync::mpsc::Sender<crate::stop::StopLevel>>,

		app_id:		String,
		kde_status:	bool,
		classic_notif:	bool,
		inhibit:	bool,
		status_fd:	Option<std::os::fd::OwnedFd>,

		#[cfg(feature = "flatpak")]
		info_path:	std::path::PathBuf,
		#[cfg(feature = "flatpak")]
		runtime_dir:	std::path::PathBuf,
	) -> impl std::future::Future<Output = Result<Self, Self::ProxyError>> + Send;

	type ProxyError;
}
