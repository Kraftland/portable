use portable_daemon::config_definition;
// use portable_daemon::config_legacy;
// use portable_daemon::config_toml;
use portable_daemon::config;
use portable_daemon::logger;
use portable_daemon::stop;
use portable_daemon::consts;
use portable_daemon::xdg;
// use portable_daemon::bind;

use thiserror::Error;

#[derive(Debug, Error)]
enum StartError {
	#[error("Could not contact logging thread: {0:#?}")]
	LogError(tokio::sync::mpsc::error::SendError<logger::LogMessage>),

	#[error("Could not wait on stop worker: {0:#?}")]
	StopWaitError(tokio::task::JoinError),

	#[error("Could not read config: {0:#?}")]
	ConfigError(config::ConfigError),

	#[error("Could not populate XDG Base Directories: {0:#?}")]
	XdgError(xdg::XdgError),

	#[error("Could not spawn thread: {0:#?}")]
	SpawnError(tokio::task::JoinError),
}

#[tokio::main]
async fn main() -> Result<(), StartError> {
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

	log_tx.send(
		logger::LogMessage {
			level: logger::LogLevel::Info,
			message: format!("Portable daemon version {}", consts::DAEMON_VERSION),
		},
	)
		.await
		.map_err(StartError::LogError)
		?;

	let xdg_dirs_spawn = tokio::spawn(xdg::XdgDirs::get());

	let config = config_definition::Config::get()
		.await
		.map_err(StartError::ConfigError)
		?;

	log_tx.send(
		logger::LogMessage {
			level: logger::LogLevel::Debug,
			message: format!("Resolved configuration: {config:#?}"),
		}
	)
		.await
		.map_err(StartError::LogError)
		?;


	let xdg_dirs = xdg_dirs_spawn
		.await
		.map_err(StartError::SpawnError)?
		.map_err(StartError::XdgError)?;
	log_tx.send(
		logger::LogMessage {
			level: logger::LogLevel::Debug,
			message: format!("Populated XDG Base Directories: {xdg_dirs:#?}"),
		}
	)
		.await
		.map_err(StartError::LogError)
		?;




	stop_worker
		.await
		.map_err(StartError::StopWaitError)?;
	Ok(())
}
