use portable_daemon::config;
use portable_daemon::logger;
use portable_daemon::stop;
use portable_daemon::consts;
use portable_daemon::xdg;
use portable_daemon::envs;
use portable_daemon::ipc;
use portable_daemon::pref;
// use portable_daemon::bind;

use thiserror::Error;

#[derive(Debug, Error)]
enum StartError {
	#[error("Could not contact logging thread: {0:#?}")]
	LogError(tokio::sync::mpsc::error::SendError<logger::LogMessage>),

	#[error("Could not contact stop worker: {0:#?}")]
	StopError(tokio::sync::mpsc::error::SendError<stop::StopLevel>),

	#[error("Could not read config: {0:#?}")]
	ConfigError(config::ConfigError),

	#[error("Could not populate XDG Base Directories: {0:#?}")]
	XdgError(xdg::XdgError),

	#[error("Could not spawn thread: {0:#?}")]
	SpawnError(tokio::task::JoinError),

	#[error("Could not register D-Bus service: {0:#?}")]
	BusError(ipc::register::RegisterError),

	#[error("Could not parse runtime options: {0:#?}")]
	RuntimeOptError(portable_daemon::pref::runtime::cmdline::RuntimeOptsError),

	#[error("Could not share files or directories: {0:#?}")]
	ShareError(portable_daemon::pref::runtime::cmdline::share_file::ShareError),

	#[error("Could not stop remote sandbox: {0:#?}")]
	StopControllerError(portable_daemon::ipc::controller::quit::StopError),
}

#[tokio::main]
async fn main() {
	let (stop_func_tx, stop_func_rx) = tokio::sync::mpsc::channel(5);
	let (stop_sig_tx, stop_sig_rx) = tokio::sync::mpsc::channel(1);
	let stop_worker = {
		tokio::spawn(stop::stop_worker(stop_func_rx, stop_sig_rx))
	};

	let log_tx = {
		let stop_clone = stop_sig_tx.clone();
		let (log_tx, log_rx) = tokio::sync::mpsc::channel(5);
		tokio::spawn(logger::logger(log_rx, stop_func_tx, stop_clone));
		log_tx
	};

	match run(log_tx.clone(), stop_sig_tx).await {
		Ok(_)	=> {}
		Err(e)	=> {
			log_tx.send(
				logger::LogMessage {
					level: logger::LogLevel::Fatal,
					message: format!("{e:#?}"),
				},
			).await.unwrap();
		}
	}
	let _ = stop_worker.await;
}

async fn run(
	log_tx:		logger::LogSender,
	stop_tx:	tokio::sync::mpsc::Sender<stop::StopLevel>,
) -> Result<(), StartError> {
	let runtime_opts_spawn = {
		tokio::spawn(portable_daemon::pref::runtime::cmdline::parse(log_tx.clone()))
	};

	let xdg_dirs_spawn = tokio::spawn(xdg::XdgDirs::get());
	let bus_spawn = {
		let stop_clone = stop_tx.clone();
		tokio::spawn(ipc::register::connect(stop_clone))
	};

	log_tx.send(
		logger::LogMessage {
			level: logger::LogLevel::Info,
			message: format!("Portable daemon version {}", consts::DAEMON_VERSION),
		},
	)
		.await
		.map_err(StartError::LogError)
		?;

	let envs_tx = {
		let log_clone = log_tx.clone();
		let (tx, rx) = envs::holder::new_channel().await;
		tokio::spawn(envs::holder::holder(rx, log_clone.to_owned()));
		tx
	};

	let xdg_dirs = xdg_dirs_spawn
		.await
		.map_err(StartError::SpawnError)?
		.map_err(StartError::XdgError)?;

	let config = {
		config::Config::get(
			log_tx.clone(),
			xdg_dirs.config_home.clone(),
		)
		.await
		.map_err(StartError::ConfigError)
	}
	?;


	let dbus_conn = bus_spawn
		.await
		.map_err(StartError::SpawnError)?
		.map_err(StartError::BusError)?;

	let bus_spawn = {
		let bus = dbus_conn.clone();
		tokio::spawn(
			ipc::register::register(
				config.metadata.sandbox_id.clone(),
				bus,
			)
		)
	};

	#[cfg(debug_assertions)]
	log_tx.send(
		logger::LogMessage {
			level: logger::LogLevel::Debug,
			message: format!("Resolved configuration: {config:#?}"),
		}
	)
		.await
		.map_err(StartError::LogError)
		?;

	#[cfg(debug_assertions)]
	log_tx.send(
		logger::LogMessage {
			level: logger::LogLevel::Debug,
			message: format!("Populated XDG Base Directories: {xdg_dirs:#?}"),
		}
	)
		.await
		.map_err(StartError::LogError)
		?;

	let runtime_opts = runtime_opts_spawn
		.await
		.map_err(StartError::SpawnError)?
		.map_err(StartError::RuntimeOptError)?;

	match runtime_opts.Action {
		pref::runtime::options::Action::Normal { debug_shell: _ }	=> {}
		pref::runtime::options::Action::ShareFile			=> {
			use portable_daemon::pref::runtime::cmdline::share_file;
			share_file::share_path_with_helper(
				&dbus_conn,
				false,
				&config.metadata.sandbox_id,
			).await
			.map_err(StartError::ShareError)
			?;

			stop_tx.send(stop::StopLevel::Normal)
				.await
				.map_err(StartError::StopError)
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

			stop_tx.send(stop::StopLevel::Normal)
				.await
				.map_err(StartError::StopError)
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

			stop_tx.send(stop::StopLevel::Normal)
				.await
				.map_err(StartError::StopError)
				?;
			return Ok(());
		}
		pref::runtime::options::Action::OpenHome			=> {
			unimplemented!();

			stop_tx.send(stop::StopLevel::Normal)
				.await
				.map_err(StartError::StopError)
				?;
			return Ok(());
		}
		pref::runtime::options::Action::ResetDocs			=> {
			unimplemented!();

			stop_tx.send(stop::StopLevel::Normal)
				.await
				.map_err(StartError::StopError)
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
			#[cfg(debug_assertions)]
			log_tx.send(
				logger::LogMessage {
					level: logger::LogLevel::Debug,
					message: format!("Entering auxiliary mode"),
				},
			)
			.await
			.map_err(StartError::LogError)
			?;

			stop_tx.send(stop::StopLevel::Normal);

			unimplemented!();

			return Ok(());
		}
	};

	log_tx.send(
		logger::LogMessage {
			level: logger::LogLevel::Debug,
			message: format!("Registered to session bus as primary"),
		}
	).await
	.map_err(StartError::LogError)?;

	/*
		Stop, or termination is handled by stop_worker, we sleep forever here to prevent bus being dropped
		TODO: remove this after implementing spawner
	*/
	std::future::pending::<()>().await;


	// stop_worker
	// 	.await
	// 	.map_err(StartError::StopWaitError)?;
	Ok(())
}
