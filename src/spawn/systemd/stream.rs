/**
	Setup the console to stream from remote Init.

	It will automatically allocate a pair of PTY file descriptors.

	The local console is set to RAW mode, restoration is handled by logging thread though.

	After which, streaming happens on different threads, until
*/
pub async fn setup() {}

async fn raw_mode() -> Result<(), StreamError> {
	let stdin = std::io::stdin();
	let mut termios = nix::sys::termios::tcgetattr(stdin)
		.map_err(StreamError::ObtainTermiosError)
		?;
	nix::sys::termios::cfmakeraw(&mut termios);
	Ok(())
}

#[derive(thiserror::Error, Debug)]
pub enum StreamError {
	#[error("Error obtaining termios")]
	ObtainTermiosError(nix::Error),
}
