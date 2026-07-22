mod config_definition;
mod legacy_config;
mod logger;
mod stop;

#[tokio::main]
async fn main() -> std::process::ExitCode {
	let (stop_tx, stop_rx) = tokio::sync::mpsc::channel(5);
	let cancel_token = tokio_util::sync::CancellationToken::new();

	let stop_worker = {
		let token_clone = cancel_token.clone();
		tokio::spawn(stop::stop_worker(stop_rx, token_clone))
	};

	let (log_tx, log_rx) = tokio::sync::mpsc::channel(5);
	tokio::spawn(logger::logger(log_rx, stop_tx));



	stop_worker.await;
	std::process::ExitCode::SUCCESS
}
