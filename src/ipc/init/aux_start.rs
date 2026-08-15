mod envs;

#[zbus::proxy(
	interface	= "top.kimiblock.Portable.Init",
	default_path	= "/top/kimiblock/portable/init",
)]
trait IPC {
	#[zbus(name = "AuxStart3")]
	async fn request_start(
		&self,
		custom_target:	bool,
		target_exec:	String,
		args_append:	bool,
		arguments:	Vec<String>,
		extra_files:	std::collections::HashMap<String, String>,
		envs:		std::collections::HashMap<String, String>,
		pty:		zbus::zvariant::OwnedFd,
	) -> zbus::Result<()>;
}

pub async fn start(
	runtime_opts:	std::sync::Arc<crate::pref::runtime::options::RuntimeOpts>,
	config:		std::sync::Arc<crate::config::Config>,
	bus:		&zbus::Connection,
	logger:		crate::logger::LogSender,
	stop_func:	tokio::sync::mpsc::Sender<crate::stop::StopFunc>,
	stop_tx:	tokio::sync::mpsc::Sender<crate::stop::StopLevel>,
) -> Result<(), AuxStartError> {
	let args = {
		let mut args = config.exec.arguments.to_owned();
		args.extend(
			runtime_opts.app_args.to_owned()
		);
		args
	};

	let init_name = {
		let mut name = String::from(config.metadata.sandbox_id.to_owned());
		name.push_str(".Portable.Helper");
		name
	};

	let exec = config.exec.target.to_owned();

	let proxy = IPCProxy::new(
		bus,
		init_name,
	)
		.await
		.map_err(AuxStartError::CreateProxyError)
		?;

	use crate::bind::subsystems::user;

	let (_expose_rules, forward_map) = {
		user::forward_file(
			&runtime_opts.file_expose,
			runtime_opts.bus_activation,
			&bus,
			&config.metadata.sandbox_id,
			logger.clone(),
		)
		.await
	};

	proxy.request_start(
		true,
		exec,
		false,
		args,
		forward_map,
		envs::get(),
		crate::spawn::stream::setup(logger, stop_func, stop_tx)
			.await
			.map_err(AuxStartError::ConsoleError)
			?
			.into(),
	)
		.await
		.map_err(AuxStartError::RemoteError)

}

#[derive(Debug, thiserror::Error)]
pub enum AuxStartError {
	#[error("Create Proxy error: {0:#?}")]
	CreateProxyError(zbus::Error),

	#[error("Create console error: {0:#?}")]
	ConsoleError(crate::spawn::stream::StreamError),

	#[error("Remote error: {0:#?}")]
	RemoteError(zbus::Error),
}
