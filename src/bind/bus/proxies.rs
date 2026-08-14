mod session;
pub mod start;
mod at_spi;

/**
	Start the D-Bus session bus and a11y bus proxies.

	Failures to the a11y bus are not critical.
*/
pub async fn start_proxies(
	logger:		crate::logger::LogSender,
	config:		std::sync::Arc<crate::config::config_definition::Config>,
	stop_tx:	tokio::sync::mpsc::Sender<crate::stop::StopLevel>,
	portable_dir:	std::sync::Arc<crate::bind::subsystems::dirs::portable_runtime::PortableRuntime>,
	bus_conn:	zbus::Connection,

	env:		crate::envs::holder::HoldChannel,

	#[cfg(feature = "flatpak")]
	flatpak_dir:	std::sync::Arc<crate::bind::subsystems::dirs::flatpak::FlatpakRuntime>,
	#[cfg(feature = "flatpak")]
	flatpak_info:	std::sync::Arc<std::path::PathBuf>,
) -> Result<crate::bind::types::BindRules, StartProxyError> {
	use crate::bind::bus::StartProxy;

	let (session_bind, session_env, session_spawn) = {

		#[cfg(feature = "flatpak")]
		let status_fd = {
			use crate::bind::subsystems::dirs::RuntimePathsTrait;
			let mut path = flatpak_dir.path();
			path.push("bwrapinfo.json");
			let file = tokio::fs::OpenOptions::new()
				.write(true)
				.create_new(true)
				.open(path)
				.await
				.map_err(StartProxyError::BwInfoError)
				?;
			std::os::fd::OwnedFd::from(file.into_std().await)
		};

		let proxy = session::SessionProxy {
			logger:		logger.clone(),
			config:		config.clone(),
			stop_token:	Some(stop_tx.clone()),
			portable_dir:	portable_dir.clone(),

			#[cfg(feature = "flatpak")]
			status_fd:	Some(status_fd),
			#[cfg(not(feature = "flatpak"))]
			status_fd:	None,

			#[cfg(feature = "flatpak")]
			flatpak_info:	flatpak_info.to_path_buf(),
		};

		let mut proxy_object = proxy
			.new()
			.await
			.map_err(StartProxyError::SessionProxyObjectError)
			?;

		let session_bind = match proxy_object.app_sandbox {
			Some(v)	=> {
				proxy_object.app_sandbox = None;
				v
			}
			None	=> {vec![]}
		};

		(
			session_bind,
			proxy_object.envs.clone().unwrap_or(std::collections::HashMap::new()),
			tokio::spawn(
				proxy_object.start()
			),
		)
	};

	let (a11y_bind, a11y_spawn) = {
		let a11y_proxy = at_spi::AtspiProxy {
			zbus_connection:	bus_conn,
			portable_runtime:	portable_dir,
			logger:			logger.clone(),
			#[cfg(feature = "flatpak")]
			flatpak_info:		flatpak_info.to_path_buf(),
		};

		let mut proxy_obj = a11y_proxy
			.new()
			.await
			.map_err(StartProxyError::AtspiProxyObjectError)
			?;

		let bind = match proxy_obj.app_sandbox {
			Some(v)	=> {
				proxy_obj.app_sandbox = None;
				v
			}
			None	=> {vec![]}
		};

		(
			bind,
			tokio::spawn(
				proxy_obj.start()
			),
		)
	};

	let mut rules = vec![];

	match session_spawn.await.map_err(StartProxyError::SpawnError)? {
		Ok(_)	=> {
			rules.extend(session_bind);
			for (k, v) in session_env {
				env.send(
					crate::envs::holder::EnvMessage::Add {
						key: k,
						value: v,
					},
				)
					.await
					.map_err(StartProxyError::EnvError)
					?;
			};
		}
		Err(e)	=> {
			return Err(StartProxyError::SessionProxyStartError(e));
		}
	};

	match a11y_spawn.await.map_err(StartProxyError::SpawnError)? {
		Ok(_)	=> {
			rules.extend(a11y_bind);
		}
		Err(e)	=> {
			let _ = logger.send(
				crate::logger::LogMessage {
					level: crate::logger::LogLevel::Warn,
					message: format!("Could not start a11y bus proxy: {e:#?}"),
				},
			).await;
		}
	}

	Ok(rules)
}

#[derive(thiserror::Error, Debug)]
pub enum StartProxyError {
	#[error("I/O error: {0:#?}")]
	IOError(std::io::Error),

	#[error("Could not create bwrapinfo.json: {0:#?}")]
	BwInfoError(std::io::Error),

	#[error("Could not spawn task: {0:#?}")]
	SpawnError(tokio::task::JoinError),

	#[error("Could not send environment variable: {0:#?}")]
	EnvError(tokio::sync::mpsc::error::SendError<crate::envs::holder::EnvMessage>),

	#[error("Could not create session bus proxy object: {0:#?}")]
	SessionProxyObjectError(crate::bind::bus::proxies::session::ProxyError),

	#[error("Could not start session bus proxy: {0:#?}")]
	SessionProxyStartError(crate::bind::bus::proxies::start::StartProxyError),

	#[error("Could not create a11y bus proxy object: {0:#?}")]
	AtspiProxyObjectError(crate::bind::bus::proxies::at_spi::AtspiError),
}
