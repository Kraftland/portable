// use thiserror::Error;

use portable_daemon::bind;

#[tokio::main]
async fn main() {
	let (_stop_func_tx, stop_func_rx) = tokio::sync::mpsc::channel(5);
	let (stop_sig_tx, stop_sig_rx) = tokio::sync::mpsc::channel(1);

	let _stop_worker = {
		tokio::spawn(portable_daemon::stop::stop_worker(stop_func_rx, stop_sig_rx))
	};
	let log_tx = {
		let stop_clone = stop_sig_tx.clone();
		let (log_tx, log_rx) = tokio::sync::mpsc::channel(5);
		tokio::spawn(portable_daemon::logger::logger(log_rx, stop_clone));
		log_tx
	};
	println!(
		"{}",
		bind::subsystems::devices::gpu::gputest_print_all_devices(&log_tx.clone()).await
	);
	std::thread::sleep(std::time::Duration::from_secs(5));
}
