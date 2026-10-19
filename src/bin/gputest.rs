// use thiserror::Error;

use portable_daemon::bind;

#[tokio::main]
async fn main() {
	let (stop_object, _stop_drainer) = portable_daemon::stop::Stop::new().await;
	let (log_tx, handle) = {
		let (log_tx, log_rx) = tokio::sync::mpsc::channel(5);
		(log_tx, tokio::spawn(portable_daemon::logger::logger(log_rx, stop_object)))
	};
	println!(
		"{}",
		bind::subsystems::devices::gpu::gputest_print_all_devices(&log_tx.clone()).await
	);
	handle.await.unwrap();
}
