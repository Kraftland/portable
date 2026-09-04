mod config;

/**
	LoggingConfig designates the initial configuration for logging thread.

	It has a function get() implemented by config module.
*/
#[derive(Debug)]
pub enum LoggingConfig {
	Console {
		colour:	Option<ColourVariant>,
	},
	Plain,
}

#[derive(Debug)]
pub enum ColourVariant {
	Normal,
	Special,
}

pub enum LogLevel {
	Debug,
	Info,
	Warn,
	Fatal,
}

pub struct LogMessage {
	pub level:		LogLevel,
	pub message:		String,
}

pub type LogSender = tokio::sync::mpsc::Sender<LogMessage>;

pub async fn logger (
	mut log_rx:	tokio::sync::mpsc::Receiver<LogMessage>,
	stop_token:	tokio_util::sync::CancellationToken,
)
{
	let is_terminal = {
		let thread = tokio::task::spawn_blocking(
			|| get_termios(),
		)
		.await;

		let thread = match thread {
			Ok(v)	=> {v}
			Err(e)	=> {
				eprintln!("Could not spawn task: {e:#?}");
				panic!("{e:#?}")
			}
		};

		match thread {
			Some(_)	=> {
				true
			}
			None	=> {
				#[cfg(debug_assertions)]
				eprintln!("Could not detect terminal status");
				false
			}
		}
	};

	let allow_colour = {
		let thread = tokio::task::spawn_blocking(|| {get_no_color_preference()})
			.await.expect("Could not get colour preference:");
		thread
	};

	let (
		debug_fmt,

		info_fmt,
		warn_fmt,
		fatal_fmt,
	) = {
		if allow_colour && is_terminal {
			match is_pups_day() {
				true	=> {
					(
						"\x1b[38;2;213;161;115m[Debug]\x1b[0m:",

						"\x1b[38;2;213;161;115m[Info]\x1b[0m:",
						"\x1b[38;2;213;161;115m[Warn]\x1b[0m:",
						"\x1b[38;2;255;0;0m[Fatal]\x1b[0m:",
					)
				}
				false	=> {
					(
						"\x1b[38;2;125;241;118m[Debug]\x1b[0m:",

						"\x1b[38;2;119;222;250m[Info]\x1b[0m:",
						"\x1b[38;2;255;209;59m[Warn]\x1b[0m:",
						"\x1b[38;2;255;0;0m[Fatal]\x1b[0m:",
					)
				}
			}

		} else {
			(
				"[Debug]:",

				"[Info]:",
				"[Warn]:",
				"[Fatal]:",
			)
		}
	};

	loop {
		let msg = tokio::select! {
			biased;
			log_msg = log_rx.recv()			=> {
				match log_msg {
					Some(v)	=> v,
					None	=> {
						return;
					}
				}
			}
			_	= stop_token.cancelled()	=> {
				return ;
			}
		};

		match msg.level {
			LogLevel::Debug	=> {
				#[cfg(debug_assertions)]
				println!("{}\t{}", debug_fmt, msg.message);
			}
			LogLevel::Info => {
				println!("{}\t\t{}", info_fmt, msg.message);
			}
			LogLevel::Warn => {
				eprintln!("{}\t\t{}", warn_fmt, msg.message);
			}
			LogLevel::Fatal => {
				eprintln!("{}\t{}", fatal_fmt, msg.message);
			}
		}
	}


}

fn is_pups_day() -> bool {
	let time =jiff::Zoned::now();
	if time.month() == 12 && time.day() == 25 {
		true
	} else {
		false
	}
}

fn get_termios() -> Option<nix::sys::termios::Termios> {
	use std::os::fd::AsFd;
	match nix::sys::termios::tcgetattr(std::io::stdin().as_fd()) {
		Ok(v)	=> {
			return Some(v);
		}
		Err(_)	=> {
			return None;
		}
	}
}

fn get_no_color_preference() -> bool {
	match std::env::var("NO_COLOR") {
		Ok(_)	=> {
			false
		}
		Err(_)	=> {
			true
		}
	}
}
