// use thiserror::Error;

use portable_daemon::bind;

#[tokio::main]
async fn main() {
	let log_tx = {
		let (log_tx, log_rx) = tokio::sync::mpsc::channel(5);
		tokio::spawn(portable_daemon::logger::logger(log_rx));
		log_tx
	};
	println!(
		"{}",
		bind::subsystems::devices::gpu::gputest_print_all_devices(&log_tx.clone()).await
	);
	std::thread::sleep(std::time::Duration::from_secs(5));
}
