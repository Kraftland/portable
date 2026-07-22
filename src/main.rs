mod config_definition;
mod legacy_config;
mod logger;
mod stop;

#[tokio::main]
async fn main() -> std::process::ExitCode {
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






	stop_worker.await;
	std::process::ExitCode::SUCCESS
}
