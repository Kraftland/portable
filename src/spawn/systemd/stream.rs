/**
	Setup the console to stream from remote Init.

	It will automatically allocate a pair of PTY file descriptors.

	The local console is set to RAW mode, restoration is handled by logging thread though.

	After which, streaming happens on different threads, until the console is exhausted
		and stop channel get a value sent.
*/
pub async fn setup(
	stop_tx:	tokio::sync::mpsc::Sender<crate::stop::StopLevel>,
) -> Result<crate::spawn::console::PtsName, StreamError> {
	let pty = crate::spawn::console::PtyPair::new()
		.map_err(StreamError::PtyAllocError)
		?;

	raw_mode()?;

	let (reader, writer) = {
		use std::os::fd::OwnedFd;
		let fd = OwnedFd::from(pty.master);
		let file = std::fs::File::from(fd);
		(file.try_clone().map_err(StreamError::CloneFdError)?, file)
	};

	// Output thread
	let output_thread = tokio::spawn(stream_out(reader));

	// Input thread
	let input_thread = tokio::spawn(stream_in(writer));
}

async fn stream_in(file: std::fs::File) -> Result<(), StreamError> {
	let mut buffer = [0u8, 4096];
	let mut stdin = {
		use std::os::fd::AsFd;
		let stdin = std::io::stdin()
			.as_fd()
			.try_clone_to_owned()
			.map_err(StreamError::StdinConvertError)
			?;
		tokio::fs::File::from_std(stdin.into())
	};

	let mut tokio_file = tokio::fs::File::from_std(file);
	use tokio::io::AsyncWriteExt;
	use tokio::io::AsyncReadExt;

	loop {
		match stdin.read(&mut buffer).await {
			Ok(0)	=> {return Ok(());}
			Ok(v)	=> {
				tokio_file.write(&buffer[..v])
					.await
					.map_err(StreamError::ConsoleIOError)
					?;
					tokio_file
						.flush()
						.await
						.map_err(StreamError::ConsoleIOError)
						?;
			}
			Err(e)	=> {
				return Err(StreamError::ConsoleIOError(e));
			}
		}
	}
}

async fn stream_out(file: std::fs::File) -> Result<(), StreamError> {
	let mut buffer = [0u8, 4096];
	let mut stdout = std::io::stdout();
	let mut tokio_file = tokio::fs::File::from_std(file);
	use tokio::io::AsyncReadExt;
	use std::io::{Write};
	loop {
		match tokio_file.read(&mut buffer).await {
			Ok(0)	=> {return Ok(())}
			Ok(v)	=> {
				stdout.write(&buffer[..v])
					.map_err(StreamError::ConsoleIOError)
					?;
				stdout
					.flush()
					.map_err(StreamError::ConsoleIOError)
					?;
			}
			Err(e)	=> {
				return Err(StreamError::ConsoleIOError(e));
			}
		}
	}
}

fn raw_mode() -> Result<(), StreamError> {
	let stdin = std::io::stdin();
	let mut termios = nix::sys::termios::tcgetattr(stdin)
		.map_err(StreamError::ObtainTermiosError)
		?;
	nix::sys::termios::cfmakeraw(&mut termios);
	Ok(())
}

#[derive(thiserror::Error, Debug)]
pub enum StreamError {
	#[error("Error obtaining termios: {0:#?}")]
	ObtainTermiosError(nix::Error),

	#[error("Error allocating PtyPair: {0:#?}")]
	PtyAllocError(crate::spawn::console::PtyError),

	#[error("Error cloning master fd: {0:#?}")]
	CloneFdError(std::io::Error),

	#[error("Error converting Stdin: {0:#?}")]
	StdinConvertError(std::io::Error),

	#[error("Error during console I/O: {0:#?}")]
	ConsoleIOError(std::io::Error),
}
