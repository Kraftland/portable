// use thiserror::Error;

use portable_daemon::bind;

#[tokio::main]
async fn main() {
	println!(
		"{}",
		bind::devices::gpu::gputest_print_all_devices().await
	)
}
