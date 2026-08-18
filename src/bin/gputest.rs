// use thiserror::Error;

use portable_daemon::bind;

#[tokio::main]
async fn main() {
	let token = tokio_util::sync::CancellationToken::new();
	let (log_tx, handle) = {
		let (log_tx, log_rx) = tokio::sync::mpsc::channel(5);
		let child = token.child_token();
		(log_tx, tokio::spawn(portable_daemon::logger::logger(log_rx, child)))
	};
	println!(
		"{}",
		bind::subsystems::devices::gpu::gputest_print_all_devices(&log_tx.clone()).await
	);
	token.cancel();
	handle.await.unwrap();
}
