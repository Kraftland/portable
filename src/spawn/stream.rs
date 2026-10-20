/**
	Setup the console to stream from remote Init.

	It will automatically allocate a pair of PTY file descriptors.

	The local console is set to RAW mode, restoration is handled by logging thread though.

	After which, streaming happens on different threads.

	The cancel token is used to signal that console streaming is complete

	Title is a terminal title to set. Application cannot override because it is set every time
	an output emerges.
*/
pub async fn setup(
	logger:		crate::logger::LogSender,
	cancel_token:	tokio_util::sync::CancellationToken,
	stop_obj:	std::sync::Arc<crate::stop::Stop>,
	config:		&crate::config::Config,
) -> Result<std::os::fd::OwnedFd, StreamError> {
	let (column, row) = match get_winsize() {
		Ok((column, row))	=> {
			(column, row)
		}
		Err(e)			=> {
			let _ = logger.send(
				crate::logger::messages::LogMessage::Warn(
					crate::logger::messages::warn::WarnMessage::WindowSize(
						e
					)
				)
			).await;
			(80, 24)
		}
	};

	let pty = crate::spawn::console::PtyPair::new(column, row)
		.map_err(StreamError::PtyAllocError)
		?;

	#[cfg(debug_assertions)]
	{
		use std::os::fd::AsRawFd;
		let _ = logger.send(
			crate::logger::messages::LogMessage::Debug(
				crate::logger::messages::debug::DebugMessage::PtyAllocated {
					master:	pty.master.as_raw_fd(),
					slave:	pty.slave.as_raw_fd(),
				}
			)
		).await;
	};

	match raw_mode(stop_obj).await {
		Ok(_)	=> {}
		Err(e)	=> {
			let _ = logger.send(
				crate::logger::messages::LogMessage::Warn(
					crate::logger::messages::warn::WarnMessage::ConsoleRawError(e)
				)
			).await;
		}
	}

	let (reader, writer, resizer) = {
		let fd = pty.master;
		let file = std::fs::File::from(fd);
		(
			file.try_clone().map_err(StreamError::CloneFdError)?,
			file.try_clone().map_err(StreamError::CloneFdError)?,
			file,
		)
	};

	let console_title = {
		let mut title = String::new();

		title.push_str("Portable app: ");
		title.push_str(&config.metadata.sandbox_id);
		title
	};

	// Output thread
	let mut output_thread = tokio::spawn(stream_out(reader, console_title));

	// Input thread
	let mut input_thread = tokio::spawn(stream_in(writer));

	// Resize thread
	let mut resize_thread = tokio::spawn(stream_resize(resizer.into()));

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
				resize	= &mut resize_thread		=> {
					#[cfg(debug_assertions)]
					println!("Resize thread stopped: {resize:#?}");
				}
			}

			output_thread.abort();
			input_thread.abort();
			resize_thread.abort();

			cancel_token.cancel();
		}
	);
	Ok(pty.slave)
}

/**
	Get the window size of stdin

	Result is in (col, row)
*/
fn get_winsize() -> Result<(u16, u16), nix::Error> {
	nix::ioctl_read_bad!(ioctl_get_winsize, nix::libc::TIOCGWINSZ, nix::libc::winsize);
	nix::ioctl_write_ptr_bad!(ioctl_set_winsize, nix::libc::TIOCSWINSZ, nix::libc::winsize);

	unsafe {
			let mut size: nix::libc::winsize = std::mem::zeroed();
			match ioctl_get_winsize(nix::libc::STDIN_FILENO, &mut size) {
				Ok(_)	=> {
					#[cfg(feature = "term-resize-debug")]
					println!(
						"Window size changed: {} col, {} row",
						&size.ws_col,
						&size.ws_row,
					);
					Ok((size.ws_col, size.ws_row))
				}
				Err(e)	=> {
					return Err(e);
				}
			}
	}
}

async fn stream_resize(master_fd: std::os::fd::OwnedFd)-> Result<(), StreamError> {
	use std::os::fd::AsRawFd;
	nix::ioctl_read_bad!(ioctl_get_winsize, nix::libc::TIOCGWINSZ, nix::libc::winsize);
	nix::ioctl_write_ptr_bad!(ioctl_set_winsize, nix::libc::TIOCSWINSZ, nix::libc::winsize);

	let sigwinch = tokio::signal::unix::signal(
		tokio::signal::unix::SignalKind::window_change(),
	);

	let mut sigwinch = match sigwinch {
		Ok(v)	=> {v}
		Err(e)	=> {return Err(StreamError::WinchError(e));}
	};

	loop {
		sigwinch.recv().await;
		let winsize = match get_winsize() {
			Ok(v)	=> v,
			Err(e)	=> {
				eprintln!("Could not get terminal dimentions: {e:#?}");
				tokio::time::sleep(std::time::Duration::from_secs(5)).await;
				continue;
			}
		};

		let size = nix::libc::winsize {
			ws_row:		winsize.1,
			ws_col:		winsize.0,
			ws_xpixel:	0,
			ws_ypixel:	0,
		};
		let result = unsafe {
			ioctl_set_winsize(master_fd.as_raw_fd(), &size)
		};
		match result {
			Ok(_)	=> {}
			Err(e)	=> {
				return Err(
					StreamError::ConsoleIOError(e.into())
				);
			}
		};
	}
}

async fn stream_in(file: std::fs::File) -> Result<(), StreamError> {
	let mut buffer = [0u8; 1024];
	let stdin = {
		tokio::io::unix::AsyncFd::new(
			make_nonblock(std::io::stdin())
				?
		)
			.map_err(StreamError::StdinSetupError)
			?
	};

	let mut tokio_file = tokio::fs::File::from_std(file);
	use tokio::io::AsyncWriteExt;

	loop {
		match
			stdin
			.readable()
			.await
			.map_err(StreamError::ConsoleIOError)
			?
			.try_io(|inner| {
				use std::io::Read;
				inner.get_ref().read(&mut buffer)
			})
		{
			Ok(_)	=> {
				tokio_file.write(&buffer)
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
				return Err(StreamError::ConsoleTryIOError(e));
			}
		}
	}
}

#[inline]
fn set_title(title: &str, stdout: &mut std::io::Stdout) -> Result<(), StreamError> {
	use std::io::Write;

	write!(stdout, "\x1b]0;{}\x07", title)
		.map_err(StreamError::ConsoleIOError)
		?;

	stdout
		.flush()
		.map_err(StreamError::ConsoleIOError)
		?;

	Ok(())
}

#[inline]
async fn stream_out(file: std::fs::File, title: String) -> Result<(), StreamError> {
	let mut buffer = [0u8; 4096];
	let mut stdout = std::io::stdout();
	let mut tokio_file = tokio::fs::File::from_std(file);
	use tokio::io::AsyncReadExt;
	use std::io::{Write};

	set_title(&title, &mut stdout)
		?;
	loop {
		match tokio_file.read(&mut buffer).await {
			Ok(0)	=> {break;}
			Ok(v)	=> {
				stdout.write_all(&buffer[..v])
					.map_err(StreamError::ConsoleIOError)
					?;
			}
			Err(e)	=> {
				return Err(StreamError::ConsoleIOError(e));
			}
		};
		set_title(&title, &mut stdout)
			?;
	};

	/*
		Perform a soft reset
	*/
	print!("\x1b[!p");

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

/**
	Duplicate and make a file non-blocking.

	This can be racy, for example, if the fd is closed when we did not clone it.

	It should only be used on standard input, standard output, standard errors etc.
*/
fn make_nonblock(fd: impl std::os::fd::AsFd) -> Result<std::fs::File, StreamError> {
	let fd = fd
		.as_fd()
		.try_clone_to_owned()
		.map_err(StreamError::StdinSetupError)
		?;

	//use std::os::fd::AsRawFd;

	let flags = nix::fcntl::fcntl(&fd, nix::fcntl::FcntlArg::F_GETFL)
		.map_err(StreamError::StdinSetupNixError)
		?;

	if flags == -1 {
		return Err(StreamError::StdinSetupError(std::io::Error::last_os_error()));
	}

	let mut flags = nix::fcntl::OFlag::from_bits_truncate(flags);

	flags.insert(nix::fcntl::OFlag::O_NONBLOCK);

	let result = nix::fcntl::fcntl(&fd, nix::fcntl::F_SETFL(flags))
		.map_err(StreamError::StdinSetupNixError)
		?;
	if result == -1 {
		return Err(StreamError::StdinSetupError(std::io::Error::last_os_error()));
	}

	Ok(fd.into())
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

	#[error("Error attempting console I/O: {0:#?}")]
	ConsoleTryIOError(tokio::io::unix::TryIoError),

	#[error("Error putting console into RAW mode: {0:#?}")]
	RawError(nix::Error),

	#[error("Could not contact stop worker: {0:#?}")]
	StopError(tokio::sync::mpsc::error::SendError<crate::stop::StopMessage>),

	#[error("Could not stream standard input: error manipulating file descriptor: {0}")]
	StdinSetupError(std::io::Error),

	#[error("Could not stream standard input: error manipulating file descriptor: {0}")]
	StdinSetupNixError(nix::Error),
}
