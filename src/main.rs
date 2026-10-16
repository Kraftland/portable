use portable_daemon::config;
use portable_daemon::logger;
use portable_daemon::stop;
use portable_daemon::xdg;
use portable_daemon::envs;
use portable_daemon::ipc;
use portable_daemon::pref;
use portable_daemon::bind;
use portable_daemon::spawn;

use portable_daemon::errors::StartError;

pub use portable_daemon::errors;

#[tokio::main]
async fn main() {
	let (stop_object, stop_drainer) = stop::Stop::new().await;



	let (log_tx, log_task) = {
		let (log_tx, log_rx) = tokio::sync::mpsc::channel(16);
		(log_tx, tokio::spawn(logger::logger(log_rx, stop_object.clone())))
	};

	let success = match run(log_tx.clone(), stop_object.clone()).await {
		Ok(_)	=> {true}
		Err(e)	=> {
			log_tx.send(
				logger::messages::LogMessage::Fatal(
					logger::messages::fatal::FatalMessage::StartError(Box::new(e))
				)
			).await.unwrap();
			false
		}
	};

	stop::worker::stop(
		stop_drainer,
		stop_object.pre_parent.clone(),
		stop_object.post_cancel.clone(),
		success,
	).await;
	log_task.await.unwrap();
}

async fn run(
	log_tx:		logger::LogSender,
	stop_obj:	std::sync::Arc<stop::Stop>,
) -> Result<(), StartError> {
	let runtime_opts_spawn = {
		tokio::spawn(portable_daemon::pref::runtime::cmdline::parse(log_tx.clone()))
	};

	let xdg_dirs = tokio::spawn(xdg::XdgDirs::get());
	let bus_spawn = {
		let token = tokio_util::sync::CancellationToken::new();
		(tokio::spawn(ipc::register::connect(token.clone())), token)
	};

	log_tx.send(
		logger::messages::LogMessage::Info(
			logger::messages::info::InfoMessage::DaemonVersion
		)
	)
		.await
		.map_err(StartError::LogError)
		?;

	#[cfg(debug_assertions)]
	log_tx.send(
		logger::messages::LogMessage::Warn(logger::messages::warn::WarnMessage::DebugBuild)
	)
		.await
		.map_err(StartError::LogError)
		?;

	let xdg_dirs = std::sync::Arc::new(
		xdg_dirs
			.await
			.map_err(StartError::SpawnError)?
			.map_err(StartError::XdgError)?
	);

	let config = std::sync::Arc::new(
		config::get(
			log_tx.clone(),
			xdg_dirs.config_home.to_path_buf(),
		)
		.await
		.map_err(StartError::ConfigError)
		?
	);

	#[cfg(feature = "desktop-file")]
	let desktop_file = tokio::spawn(
		pref::runtime::desktop_file::get(
			stop_obj.clone(),
			config.clone(),
			xdg_dirs.clone(),
		)
	);

	let envs_tx = {
		let xdg = xdg_dirs.clone();
		let config = config.clone();
		let log_clone = log_tx.clone();
		let (tx, rx) = envs::holder::new_channel().await;
		tokio::spawn(
			envs::holder::holder(
				rx,
				log_clone.to_owned(),
				xdg,
				config,
			),
		);
		tx
	};

	let dbus_conn = bus_spawn.0
		.await
		.map_err(StartError::SpawnError)?
		.map_err(StartError::BusError)?;

	let bus_cancel = bus_spawn.1;

	let bus_spawn = {
		let bus = dbus_conn.clone();
		tokio::spawn(
			ipc::register::register(
				config.metadata.sandbox_id.to_string(),
				bus,
			)
		)
	};

	let runtime_opts = std::sync::Arc::new(
		runtime_opts_spawn
			.await
			.map_err(StartError::SpawnError)?
			.map_err(StartError::RuntimeOptError)?
	);

	match &runtime_opts.action {
		pref::runtime::options::Action::Normal				=> {}
		pref::runtime::options::Action::ShareFile			=> {
			use portable_daemon::pref::runtime::cmdline::share_file;
			share_file::share_path_with_helper(
				&dbus_conn,
				false,
				&config.metadata.sandbox_id,
			).await
			.map_err(StartError::ShareError)
			?;
			return Ok(());
		}
		pref::runtime::options::Action::ShareDir			=> {
			use portable_daemon::pref::runtime::cmdline::share_file;
			share_file::share_path_with_helper(
				&dbus_conn,
				true,
				&config.metadata.sandbox_id,
			).await
			.map_err(StartError::ShareError)
			?;
			return Ok(());
		}
		pref::runtime::options::Action::Quit				=> {
			use portable_daemon::ipc::controller::quit;

			quit::stop_app(
				&config.metadata.sandbox_id,
				&dbus_conn,
			)
				.await
				.map_err(StartError::StopControllerError)
				?;
			return Ok(());
		}
		pref::runtime::options::Action::OpenHome			=> {
			pref::runtime::cmdline::open_home::open(
				xdg_dirs,
				&config.metadata.state_directory,
				&dbus_conn,
			)
				.await
				.map_err(StartError::OpenHomeError)
				?;
			return Ok(());
		}
		pref::runtime::options::Action::ResetDocs			=> {

			pref::runtime::cmdline::reset(
				std::sync::Arc::new(config.metadata.sandbox_id.to_owned()),
				&dbus_conn,
				log_tx.clone(),
			)
				.await
				.map_err(StartError::ResetError)
				?;

			log_tx.send(
				logger::messages::LogMessage::Info(
					logger::messages::info::InfoMessage::PermissionRevoked { config }
				)
			)
			.await
			.map_err(StartError::LogError)
			?;

			return Ok(());
		}
	}

	match bus_spawn
		.await
		.map_err(StartError::SpawnError)?
		.map_err(StartError::BusError)?
	{
		ipc::register::RegisterStatus::Primary		=> {}
		ipc::register::RegisterStatus::Secondary	=> {
			if ! config.advanced.tray_wake {
				ipc::init::aux_start::start(
					runtime_opts,
					config,
					&dbus_conn,
					log_tx.clone(),
					stop_obj.clone(),
				)
					.await
					.map_err(StartError::AuxStartError)
					?;

				#[cfg(debug_assertions)]
				log_tx.send(
					logger::messages::LogMessage::Debug(
						logger::messages::debug::DebugMessage::AuxRequested
					)
				)
				.await
				.map_err(StartError::LogError)
				?;
			} else {
				ipc::init::tray::wake(config, &dbus_conn)
					.await
					.map_err(StartError::TrayError)
					?;
			}

			return Ok(());
		}
	};

	#[cfg(feature = "desktop-file")]
	let desktop_file: std::sync::Arc<freedesktop_desktop_entry::DesktopEntry> = desktop_file
		.await
		.map_err(StartError::SpawnError)
		?
		.map_err(StartError::DesktopFileError)
		?
		.into();

	// Fire the consent checker right away, to avoid delaying startup
	let dynamic_permissions = {
		tokio::spawn(
			pref::permissions::consent::get(
				log_tx.clone(),
				config.clone(),
				dbus_conn.clone(),

				#[cfg(feature = "desktop-file")]
				desktop_file,
			)
		)
	};

	let instance_id = std::sync::Arc::new(
		spawn::instance_id::generate_instance_id(
			&xdg_dirs.runtime,
			log_tx.clone(),
		)
			.await
			.map_err(StartError::InstanceIDError)
			?
	);

	let dynamic_permissions = match
		dynamic_permissions
			.await
			.map_err(StartError::SpawnError)
			?
	{
		Ok(v)	=> v,
		Err(e)	=> {
			let _ = log_tx.send(
				logger::messages::LogMessage::Warn(
					logger::messages::warn::WarnMessage::DynamicPermissionsError(e)
				)
			)
				.await;

			std::sync::Arc::new(std::collections::HashMap::new())
		}
	};

	#[cfg(feature = "flatpak")]
	let flatpak_runtime_spawn = {
		use bind::subsystems::dirs::RuntimePathsTrait;
		use bind::subsystems::dirs::flatpak::FlatpakRuntime;
		let config_clone = config.clone();
		let xdg_clone = xdg_dirs.clone();
		let instance_id_clone = instance_id.clone();
		let stop_clone = stop_obj.clone();
		tokio::spawn(
				async move {
					let runtime = FlatpakRuntime::new(
						config_clone,
						xdg_clone,
						instance_id_clone,
					);
					runtime
						.create_path(stop_clone)
						.await
						.map_err(StartError::FlatpakRuntimePathError)
						?;
					Ok(runtime)
				}
		)
	};

	let (
		portable_runtime_spawn,
		document_spawn,
	) = {
		let stop_clone = stop_obj.clone();
		use bind::subsystems::dirs::RuntimePathsTrait;
		use bind::subsystems::dirs::portable_runtime::PortableRuntime;
		use bind::subsystems::dirs::documents;

		let config_clone = config.clone();
		let config_clone_2 = config.clone();
		let xdg_clone = xdg_dirs.clone();
		let instance_id_clone = instance_id.clone();
		let bus_clone = dbus_conn.clone();

		(
			tokio::spawn(
				async move {
					let runtime = PortableRuntime::new(
						config_clone,
						xdg_clone,
						instance_id_clone,
					);
					runtime
						.create_path(stop_clone.clone())
						.await
						.map_err(StartError::PortableRuntimePathError)
						?;
					Ok(runtime)
				},
			),
			tokio::spawn(
				async move {
					let runtime = documents::DocumentsMountPoint::new(
						config_clone_2,
						bus_clone,
					)
					.await
					.map_err(StartError::DocumentRuntimePathError)
					?;

					runtime.create_path()
						.await
						.map_err(StartError::DocumentRuntimePathError)
						?;
					Ok(runtime)
				}
			),
		)
	};

	let (portable_runtime, document) = {
		(
			std::sync::Arc::new(
				portable_runtime_spawn
					.await
					.map_err(StartError::SpawnError)
					?
					?
			),
			document_spawn.await.map_err(StartError::SpawnError)??,
		)
	};

	#[cfg(feature = "flatpak")]
	let flatpak_runtime = std::sync::Arc::new(
		flatpak_runtime_spawn
			.await
			.map_err(StartError::SpawnError)
			?
			?
	);

	let flatpak_info_spawn = {
		let portable_clone = portable_runtime.clone();
		let config_clone = config.clone();
		let instance_clone = instance_id.clone();
		let xdg_clone = xdg_dirs.clone();
		let flatpak_runtime = flatpak_runtime.clone();
		tokio::spawn(
				async {
					Ok(
						bind::flatpak_info::create(
							config_clone,
							instance_clone,
							xdg_clone,
							portable_clone,

							#[cfg(feature = "flatpak")]
							flatpak_runtime,
						)
						.await
						.map_err(StartError::FlatpakInfoError)?
				)
			}
		)
	};

	let flatpak_info = std::sync::Arc::new(
		flatpak_info_spawn
			.await
			.map_err(StartError::SpawnError)
			?
			?
	);

	let proxy_cancel = tokio_util::sync::CancellationToken::new();
	let bus_binds = tokio::spawn(
		bind::bus::proxies::start_proxies(
			log_tx.clone(),
			config.clone(),
			proxy_cancel.clone(),
			portable_runtime.clone(),
			dbus_conn.clone(),
			envs_tx.clone(),
			runtime_opts.clone(),
			dynamic_permissions.clone(),
			#[cfg(feature = "flatpak")]
			flatpak_runtime,
			#[cfg(feature = "flatpak")]
			flatpak_info.clone(),
		)
	);

	let (mut bind_rules, init_info, _subsystem_cancel) = bind::subsystems::generate_bindrules(
		portable_runtime,
		document,
		xdg_dirs.clone(),
		config.clone(),
		dynamic_permissions.clone(),
		log_tx.clone(),
		stop_obj.clone(),
		envs_tx.clone(),
		instance_id.to_string(),
		flatpak_info.clone(),
		runtime_opts,
		dbus_conn.clone(),
	)
		.await
		.map_err(StartError::BindError)
		?;

	let init_info = tokio::spawn(
		init_info.publish(
			dbus_conn.clone(),
		),
	);

	bind_rules.extend(
		bus_binds
			.await
			.map_err(StartError::SpawnError)
			?
			.map_err(StartError::ProxyError)
			?,
	);
	init_info
		.await
		.map_err(StartError::SpawnError)
		?
		.map_err(StartError::InitPublishError)
		?;


	let spawn_cancel = {
		let cancel_token = tokio_util::sync::CancellationToken::new();

		let spawn_struct = spawn::Spawn {
			config:		config.clone(),
			uid:		instance_id.to_string(),
			fs_rules:	bind_rules,
			logger:		log_tx.clone(),
			stop:		stop_obj.clone(),
			envs:		envs_tx,
			cancen_token:	cancel_token.clone(),
			sandbox_home:	{
				let mut state_dir = xdg_dirs.data_home.to_path_buf();
				state_dir.push(&config.metadata.state_directory);
				state_dir
			}
		};
		use spawn::Start;
		spawn_struct
			.start(&dbus_conn)
			.await
			.map_err(StartError::SpawnSandboxError)
			?;
		cancel_token
	};

	tokio::select! {
		_	=	bus_cancel.cancelled()	=> {
			#[cfg(debug_assertions)]
			log_tx.send(
				logger::messages::LogMessage::Debug(
					logger::messages::debug::DebugMessage::QuitController
				)
			)
				.await
				.map_err(StartError::LogError)
				?;
		}

		/*
			This is problematic. Should the first invocation be finished and
				other instances remaining, this will cause them to be killed too
		*/
		// _	=	subsystem_cancel.cancelled()	=> {
		// 	#[cfg(debug_assertions)]
		// 	log_tx.send(
		// 		logger::LogMessage {
		// 			level:		logger::LogLevel::Debug,
		// 			message:	format!("Quit requested: Console stream ended"),
		// 		}
		// 	)
		// 		.await
		// 		.map_err(StartError::LogError)
		// 		?;
		// }
		_	=	proxy_cancel.cancelled()	=> {
			#[cfg(debug_assertions)]
			log_tx.send(
				logger::messages::LogMessage::Debug(
					logger::messages::debug::DebugMessage::QuitProxy
				)
			)
				.await
				.map_err(StartError::LogError)
				?;
		}

		_	=	spawn_cancel.cancelled()	=> {
			#[cfg(debug_assertions)]
			log_tx.send(
				logger::messages::LogMessage::Debug(
					logger::messages::debug::DebugMessage::QuitSandbox
				)
			)
				.await
				.map_err(StartError::LogError)
				?;
		}
	}
	Ok(())
}
