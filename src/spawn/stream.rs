/**
	Setup the console to stream from remote Init.

	It will automatically allocate a pair of PTY file descriptors.

	The local console is set to RAW mode, restoration is handled by logging thread though.

	After which, streaming happens on different threads.

	The cancel token is used to signal that console streaming is complete
*/
pub async fn setup(
	logger:		crate::logger::LogSender,
	cancel_token:	tokio_util::sync::CancellationToken,
	stop_obj:	std::sync::Arc<crate::stop::Stop>,
) -> Result<std::os::fd::OwnedFd, StreamError> {
	let pty = crate::spawn::console::PtyPair::new()
		.map_err(StreamError::PtyAllocError)
		?;

	#[cfg(debug_assertions)]
	{
		use std::os::fd::AsRawFd;
		let _ = logger.send(
		crate::logger::LogMessage {
			level: crate::logger::LogLevel::Debug,
			message: format!(
				"Allocated pty master (fd:{0:?}) and slave (fd:{1:?})",
					pty.master.as_raw_fd(),
					pty.slave.as_raw_fd(),
				),
			}
		).await;
	};

	match raw_mode(stop_obj).await {
		Ok(_)	=> {
			#[cfg(debug_assertions)]
			let _ = logger.send(
				crate::logger::LogMessage {
					level: crate::logger::LogLevel::Debug,
					message: format!("Successfully set console to raw mode"),
				}
			).await;
		}
		Err(e)	=> {
			let _ = logger.send(
				crate::logger::LogMessage {
					level: crate::logger::LogLevel::Warn,
					message: format!(
						"Could not put console into RAW mode: {e:#?}",
					),
				}
			).await;
		}
	}

	let (reader, writer) = {
		let fd = pty.master;
		let file = std::fs::File::from(fd);
		(file.try_clone().map_err(StreamError::CloneFdError)?, file)
	};

	// Output thread
	let mut output_thread = tokio::spawn(stream_out(reader));

	// Input thread
	let mut input_thread = tokio::spawn(stream_in(writer));

	tokio::spawn(
		async move {
			tokio::select! {
				out	= &mut output_thread		=> {
					#[cfg(debug_assertions)]
					println!("Output thread stopped: {out:#?}");
				}
				input	= &mut input_thread		=> {
					#[cfg(debug_assertions)]
					println!("Input thread stopped: {input:#?}");
				}
			}

			output_thread.abort();
			input_thread.abort();

			cancel_token.cancel();
		}
	);
	Ok(pty.slave)
}

async fn stream_in(file: std::fs::File) -> Result<(), StreamError> {
	use std::os::fd::AsRawFd;

	let sigwinch = tokio::signal::unix::signal(
		tokio::signal::unix::SignalKind::window_change(),
	);

	let mut sigwinch = match sigwinch {
		Ok(v)	=> {v}
		Err(e)	=> {return Err(StreamError::WinchError(e));}
	};

	nix::ioctl_read_bad!(ioctl_get_winsize, nix::libc::TIOCGWINSZ, nix::libc::winsize);
	nix::ioctl_write_ptr_bad!(ioctl_set_winsize, nix::libc::TIOCSWINSZ, nix::libc::winsize);


	let mut buffer = [0u8; 1024];
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
		let signal = tokio::select! {
			input	=	stdin.read(&mut buffer)	=> {input}
			_	=	sigwinch.recv()		=> {
				let winsize = unsafe {
					let mut size: nix::libc::winsize = std::mem::zeroed();
					match ioctl_get_winsize(nix::libc::STDIN_FILENO, &mut size) {
						Ok(_)	=> {(size.ws_col, size.ws_row)}
						Err(_)	=> {continue;}
					}
				};

				let size = nix::libc::winsize {
					ws_row:		winsize.1,
					ws_col:		winsize.0,
					ws_xpixel:	0,
					ws_ypixel:	0,
				};
				let result = unsafe {
					ioctl_set_winsize(stdin.as_raw_fd(), &size)
				};

				match result {
					Ok(_)	=> {}
					Err(e)	=> {
						return Err(
							StreamError::ConsoleIOError(e.into())
						);
					}
				};

				continue;
			}
		};

		match signal {
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
	let mut buffer = [0u8; 4096];
	let mut stdout = std::io::stdout();
	let mut tokio_file = tokio::fs::File::from_std(file);
	use tokio::io::AsyncReadExt;
	use std::io::{Write};
	loop {
		match tokio_file.read(&mut buffer).await {
			Ok(0)	=> {break;}
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
	};

	Ok(())
}

async fn raw_mode(
	stop:	std::sync::Arc<crate::stop::Stop>,
) -> Result<(), StreamError> {
	let stdin = std::io::stdin();
	let mut termios = nix::sys::termios::tcgetattr(&stdin)
		.map_err(StreamError::ObtainTermiosError)
		?;

	let termios_clone = termios.clone();

	nix::sys::termios::cfmakeraw(&mut termios);

	nix::sys::termios::tcsetattr(&stdin, nix::sys::termios::SetArg::TCSANOW, &termios)
		.map_err(StreamError::RawError)
		?;

	let cancel_token = stop.pre_parent.child_token();

	stop.stop_funcs.send(
		crate::stop::StopMessage::Prepare {
			task:	tokio::spawn(

				async move {
					cancel_token.cancelled().await;

					nix::sys::termios::tcsetattr(
						stdin,
						nix::sys::termios::SetArg::TCSANOW,
						&termios_clone,
					).map_err(crate::stop::StopError::RestoreConsoleError)
				},

			),
		}
	)
		.map_err(StreamError::StopError)
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

	#[error("Error subscribing to SIGWINCH signal: {0:#?}")]
	WinchError(std::io::Error),

	#[error("Error during console I/O: {0:#?}")]
	ConsoleIOError(std::io::Error),

	#[error("Error putting console into RAW mode: {0:#?}")]
	RawError(nix::Error),

	#[error("Could not contact stop worker: {0:#?}")]
	StopError(tokio::sync::mpsc::error::SendError<crate::stop::StopMessage>),
}
